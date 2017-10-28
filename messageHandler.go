package main

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/d0x1p2/godbot"
)

func (cfg *Config) msghandler(s *discordgo.Session, m *discordgo.MessageCreate) {
	var err error
	var c *godbot.Channel

	var io = msgToIOdat(m)
	cfg.DSession = s

	// Required for storing information in the correct database.
	if Bot != nil {
		// Prevents accessing nil pointers and crashing bot.
		if c = Bot.GetChannel(m.ChannelID); c == nil {
			fmt.Println("Nil channel prevented.")
			return
		}

		if io.guild = Bot.GetGuild(c.GuildID); io.guild == nil {
			fmt.Println("Nil guild prevented.")
			return
		}
	} else {
		fmt.Println("Nil Bot... returning")
		return
	}

	if m.Author.Bot {
		if m.Author.ID == Bot.User.ID {
			ts, _ := m.Timestamp.Parse()
			if err := UserUpdateSimple(c.GuildID, m.Author, 1, ts); err != nil {
				return
			}
		}
		return
	} else if c.Type == discordgo.ChannelTypeDM {
		// Handle private messages.
		if strings.Contains(m.Content, ",list") {
			s.ChannelMessageSend(c.ID, channelsTemp())
		}
		return
	}

	// Log message into Database
	if _, err := messageLog(io.guild.Name, io.guild.ID, c.Name, m.Message); err != nil {
		fmt.Println(err)
	}

	// Handle the message appropriately if it is a message between alliances.
	cfg.AllianceHandler(m.Message)

	// Handle potential WatchLogs
	cfg.WatchLogHandler(io.guild, m, c.Name)

	// Return due to not being a command and/or just an Embed.
	if io.command == false || len(io.io) == 0 {
		return
	}

	io.user = UserNew(m.Author)
	if err := io.user.Get(m.Author.ID); err != nil {
		fmt.Println(err)
		return
	}

	// Handle the parse the various commands the message can be.
	err = cfg.ioHandler(io)
	if err != nil {
		io.msgEmbed = embedCreator(fmt.Sprintf("%s", err.Error()), ColorMaroon)
	}
	// Prevention from attempting access of null pointer from console.
	if io.msg != nil && io.rm {
		s.ChannelMessageDelete(io.msg.ChannelID, io.msg.ID)
	}

	// Send message here.
	if io.output != "" {
		s.ChannelMessageSend(m.ChannelID, io.output)
	} else if io.msgEmbed != nil {
		s.ChannelMessageSendEmbed(m.ChannelID, io.msgEmbed)
	}

	return
}

// newUserHandler greets a new palyers to the channel.
func (cfg *Config) newUserHandler(s *discordgo.Session, nu *discordgo.GuildMemberAdd) {
	if Bot != nil {
		c := Bot.GetMainChannel(nu.GuildID)
		msg := fmt.Sprintf("Welcome to the server, __**%s**#%s__!", nu.User.Username, nu.User.Discriminator)
		s.ChannelMessageSendEmbed(c.ID, embedCreator(msg, ColorBlue))

		for _, ch := range Bot.Channels {
			if ch.Name == "internal" && ch.GuildID == nu.GuildID {
				tn := time.Now()
				msg := fmt.Sprintf("__**%s**#%s__ joined the server @ %s\n",
					nu.User.Username, nu.User.Discriminator, tn.Format(time.UnixDate))
				s.ChannelMessageSendEmbed(ch.ID, embedCreator(msg, ColorBlue))
			}
		}
	}
}

// delUserHandler notifies of a leaving user (NOT CURRENTLY WORKING)
func delUserHandler(s *discordgo.Session, du *discordgo.GuildMemberRemove) {
	if Bot != nil {
		for _, c := range Bot.Channels {
			if c.Name == "internal" {
				tn := time.Now()
				msg := fmt.Sprintf("__**%s**#%s__ left the server @ %s\n",
					du.User.Username, du.User.Discriminator, tn.Format(time.UnixDate))
				s.ChannelMessageSendEmbed(c.ID, embedCreator(msg, ColorBlue))
			}
		}
	}
}

// WatchLogHandler tracks if the message is under WatchLog.
func (cfg *Config) WatchLogHandler(guild *godbot.Guild, msg *discordgo.MessageCreate, channel string) {

	// Verify we have a watchlogger on this guild.
	var watched WatchLog
	for _, w := range cfg.watched {
		if w.guildID == guild.ID {
			watched = w
			break
		}
	}

	// Return since guild ID isn't being watched.
	if watched.guildID == "" {
		return
	}

	// Compose the message to send to the channel then the socket.
	var output = "--> "
	if watched.channelID == "" {
		output += "[" + channel + "]"
		// Check that we are monitoring this channel if any...
	} else if strings.ToLower(watched.channelName) != strings.ToLower(channel) {
		return
	}

	// Eventually add support for ContentWithMoreMentionsReplaced()
	output += "[" + msg.Author.Username + "#" + msg.Author.Discriminator + "] " + msg.ContentWithMentionsReplaced()

	// Send the composed message.
	watched.Talk(output)
}

// messageLog logs the supplied message into a local database.
func messageLog(database, databaseID, channel string, msg *discordgo.Message) (bool, error) {

	m := MessageNew(database, databaseID, channel, msg)
	if ok, err := m.Update(database); err != nil {
		return false, err
	} else if ok {
		ts, _ := msg.Timestamp.Parse()
		if err := UserUpdateSimple(databaseID, msg.Author, 1, ts); err != nil {
			fmt.Println("updating/adding user", err)
		}
		return true, nil
	}
	return false, nil
}

// MessageIntegrityCheck verifies the integrity of channels of a specific guild.
func (cfg *Config) MessageIntegrityCheck(gName string) (string, error) {
	var gID string
	var found bool

	// Find the guild in the currently accessible guilds.
	for _, g := range cfg.Core.Guilds {
		if strings.Contains(g.Name, gName) {
			gName = g.Name
			gID = g.ID
			found = true
			break
		}
	}

	// Return if could not be found.
	if !found {
		return "", errors.New("could not find guild ID")
	}

	var missed int
	// Process all channels within the guild (that are currently linked to the guild.)
	for _, c := range cfg.Core.Links[gID] {
		// If the channel is not a text channel, continue to next channel.
		if c.Type != 0 {
			continue
		}

		var mID string
		for {
			var bk bool
			// Grab 100 messages.
			msgs, err := cfg.Core.Session.ChannelMessages(c.ID, 100, mID, "", "")
			if err != nil {
				return "", err
			}

			// Update amount of messages actually received, less than 100 indicates there are less than 100 available.
			var cnt = len(msgs)
			if cnt == 0 {
				bk = true
				break
			}

			// Process each message individually and log it.
			for n, m := range msgs {
				mID = m.ID

				if ok, err := messageLog(gName, gID, c.Name, m); err != nil {
					fmt.Println("Error logging message", err.Error())
				} else if ok {
					missed++
				}

				// Break early if the last message (less than 100) is processed.
				if cnt < 100 && n+1 == cnt {
					bk = true
					break
				}
			}
			if bk {
				break
			}
		}
	}
	var str = fmt.Sprintf("No messages were skipped for %s in the past.\n", gName)
	if missed > 0 {
		str = fmt.Sprintf("%d messages have been added.\n", missed)
	}
	return str, nil
}

// MessageNew returns a new message object.
func MessageNew(database, databaseID, channel string, m *discordgo.Message) *Message {
	u := UserNew(m.Author)

	var ts, ets time.Time
	var err error
	if ts, err = m.Timestamp.Parse(); err != nil {
		// TAG: TODO
		// Log error that should never happen (according to the documentation)
		// Will be logged with a logger once added.
		fmt.Println(err)
	}

	if ets, err = m.EditedTimestamp.Parse(); err != nil {
		// TAG: TODO
		// Log error that should never happen (according to the documentation)
		// Will be logged with a logger once added.
		fmt.Println(err)
	}

	return &Message{
		ID:              m.ID,
		ChannelID:       m.ChannelID,
		ChannelName:     channel,
		Content:         m.Content,
		Timestamp:       ts,
		EditedTimestamp: ets,
		Author:          u.Basic(),
	}
}

// Update Checks and if not exists... Adds to the database.
func (m *Message) Update(database string) (bool, error) {
	var q = make(map[string]interface{})
	q["id"] = m.ID

	db := DBdatCreate(database, CollectionMessages, m, q, nil)
	if err := db.dbExists(); err != nil {
		if err == ErrNoDocument {
			// Insert the message into the database here.
			if err := db.dbInsert(); err != nil {
				return false, err
			}
			return true, nil
		}
		return false, err
	}
	return false, nil
}
