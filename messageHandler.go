package main

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"gopkg.in/mgo.v2/bson"

	"github.com/bwmarrin/discordgo"
	"github.com/d0x1p2/godbot"
)

// messageCreateHandler is called when a new message appears in discord server
// that is accessible by the bot.
func (cfg *Config) messageCreateHandler(s *discordgo.Session, m *discordgo.MessageCreate) {
	var err error
	var c *godbot.Channel
	var g *godbot.Guild
	cfg.DSession = s

	// Required for storing information in the correct database.
	if Bot != nil {
		// Prevents accessing nil pointers and crashing bot.
		if c = Bot.GetChannel(m.ChannelID); c == nil {
			c = &godbot.Channel{}
			// If c is nil, it is most likely a private channel, grab via API.
			if c.Channel, err = s.Channel(m.ChannelID); err != nil {
				fmt.Println("Getting channel, possibly private: " + err.Error())
				return
			}

			// If couldn't find channel via API, return.
			if c.Channel == nil {
				fmt.Println("Nil channel prevented.")
				return
			}
		}
	} else {
		fmt.Println("Nil Bot... returning")
		return
	}

	if m.Author.Bot {
		if m.Author.ID == Bot.User.ID {
			ts, err := m.Timestamp.Parse()
			// TAG: TODO - fix timestamp
			if err != nil {
				fmt.Println(err)
			}
			if err := UserUpdateSimple(c.GuildID, m.Author, 1, ts); err != nil {
				return
			}
		}
		return
	} else if c.Type == discordgo.ChannelTypeDM {
		// Handle private messages.
		if strings.Contains(m.Content, ",list") {
			s.ChannelMessageSend(c.ID, channelsTemp())
		} else if strings.Contains(m.Content, ",help") {
			s.ChannelMessageSend(c.ID, globalHelp())
		}

		// Log message into Database
		if _, err := messageLogger("private", c.ID, "", m.Message); err != nil {
			fmt.Println(err)
		}

		// Check if it's being watched by WatchLogger
		cfg.watchLogHandler(nil, m, "private")

		return
	}

	// Get the correct guild. Used for Database.
	if Bot != nil && c.Type != discordgo.ChannelTypeDM {
		// Get the guild from known guild structures.
		if g = Bot.GetGuild(c.GuildID); g == nil {
			g = &godbot.Guild{}
			// Guild is not know, pull via API.
			if g.Guild, err = s.Guild(c.GuildID); err != nil {
				fmt.Println("Getting guild, possibly private: " + err.Error())
				return
			}

			// Guild is still nil, even from API- return.
			if g.Guild == nil {
				fmt.Println("Nil guild prevented.")
				return
			}
		}
	}

	// Load guild config. Add and update if not found. Shouldn't happen due to new
	// guilds being processed upon adding.
	gConf := cfg.GuildConfigByID(g.ID)
	if gConf.ID == "" {
		gConf = GuildConfig{ID: g.ID, Name: g.Name, Prefix: envCMDPrefix}
		if err = cfg.GuildConfigManager(gConf); err != nil {
			fmt.Println(err)
			return
		}
	}

	// Log message into Database
	if _, err := messageLogger(g.Name, g.ID, c.Name, m.Message); err != nil {
		fmt.Println(err)
	}

	var io = msgToIOdata(m, gConf.Prefix)
	io.guild = g

	// Handle the message appropriately if it is a message between alliances.
	cfg.allianceHandler(m.Message)

	// Handle potential WatchLogs
	cfg.watchLogHandler(io.guild, m, c.Name)

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

// messageUpdateHandler takes care of message edits and reflects the modification into the database.
// TAG: TODO make this edit the bots Alliance messages as well.
func (cfg *Config) messageUpdateHandler(s *discordgo.Session, mu *discordgo.MessageUpdate) {
	var channel *godbot.Channel
	var guild *godbot.Guild
	var database string
	var err error

	// MessageUpdate event is being triggered by embeds and attachments.
	if len(mu.Embeds) > 0 {
		return
	} else if len(mu.Attachments) > 0 {
		return
	}

	// Required for storing information in the correct database.
	if Bot != nil {
		// Prevents accessing nil pointers and crashing bot.
		if channel = Bot.GetChannel(mu.ChannelID); channel == nil {
			channel = &godbot.Channel{}
			// If c is nil, it is most likely a private channel, grab via API.
			if channel.Channel, err = s.Channel(mu.ChannelID); err != nil {
				fmt.Println("Getting channel, possibly private: " + err.Error())
				return
			}

			// If couldn't find channel via API, return.
			if channel.Channel == nil {
				fmt.Println("Nil channel prevented.")
				return
			}
		}

		if channel.Type != discordgo.ChannelTypeDM {
			// Get the guild from known guild structures.
			if guild = Bot.GetGuild(channel.GuildID); guild == nil {
				guild = &godbot.Guild{}
				// Guild is not know, pull via API.
				if guild.Guild, err = s.Guild(channel.GuildID); err != nil {
					fmt.Println("Getting guild, possibly private: " + err.Error())
					return
				}

				// Guild is still nil, even from API- return.
				if guild.Guild == nil {
					fmt.Println("Nil guild prevented.")
					return
				}
			}

			database = guild.Name
		} else {
			// Account for private messages.
			database = "private"
		}
	} else {
		// Bot isn't ready (most likely just started), return.
		fmt.Println("Nil Bot... returning")
		return
	}

	var msg Message
	msg.ID = mu.ID

	// Pull message from database
	if err = msg.Get(database); err != nil {
		fmt.Println("Attempting to load message from database: " + err.Error())
		return
	}

	// Update timestampa
	if msg.EditedTimestamp, err = mu.EditedTimestamp.Parse(); err != nil {
		// TAG: TODO - parsing timestamp: may change.
		fmt.Println("Editing timestamp: " + err.Error())
		return
	}

	// Add current content to existing content.
	msg.EditedContent = append(msg.EditedContent, msg.Content)
	// Make the current content reflect the edited value.
	msg.Content = mu.Content

	// Create queries to place the new message into database.
	q := make(map[string]interface{})
	c := make(map[string]interface{})
	q["id"] = msg.ID
	c["$set"] = bson.M{
		"content":         msg.Content,
		"editedcontent":   msg.EditedContent,
		"editedtimestamp": msg.EditedTimestamp,
	}

	// Edit the current message in the database.
	db := DBdataCreate(database, CollectionMessages, msg, q, c)
	if err = db.dbEdit(Message{}); err != nil {
		fmt.Println("Editing message in DB: " + err.Error())
		return
	}

	return
}

// guildMemberAddHandler greets a new palyers to the channel.
func (cfg *Config) guildMemberAddHandler(s *discordgo.Session, nu *discordgo.GuildMemberAdd) {
	if Bot != nil {
		c := Bot.GetMainChannel(nu.GuildID)
		msg := fmt.Sprintf("Welcome to the server, __**%s**#%s__!", nu.User.Username, nu.User.Discriminator)
		s.ChannelMessageSendEmbed(c.ID, embedCreator(msg, ColorBlue))

		for _, ch := range Bot.Channels {
			if ch.Name == "internal" && ch.GuildID == nu.GuildID {
				tn := time.Now()
				msg := fmt.Sprintf("__**%s**#%s__ [ID: %s] joined the server @ %s\n",
					nu.User.Username, nu.User.Discriminator, nu.User.ID, tn.Format(time.UnixDate))
				s.ChannelMessageSendEmbed(ch.ID, embedCreator(msg, ColorGreen))
				return
			}
		}
	}
}

// guildMemberRemoveHandler notifies of a leaving user (NOT CURRENTLY WORKING)
func (cfg *Config) guildMemberRemoveHandler(s *discordgo.Session, du *discordgo.GuildMemberRemove) {
	if Bot != nil {
		for _, c := range Bot.Channels {
			if c.Name == "internal" && c.GuildID == du.GuildID {
				tn := time.Now()
				msg := fmt.Sprintf("__**%s**#%s__ [ID: %s] left the server @ %s\n",
					du.User.Username, du.User.Discriminator, du.User.ID, tn.Format(time.UnixDate))
				s.ChannelMessageSendEmbed(c.ID, embedCreator(msg, ColorMaroon))
				return
			}
		}
	}
}

// watchLogHandler tracks if the message is under WatchLog.
func (cfg *Config) watchLogHandler(guild *godbot.Guild, msg *discordgo.MessageCreate, channel string) {
	// TAG: TODO - account for watched servers that have same guild, but not same channel.
	// Verify we have a watchlogger on this guild.
	var guildID string
	var watched WatchLog
	// Check if private messages are being WatchLogged.
	if guild == nil && channel == "private" {
		guildID = "private"
		channel = ""
	} else if guild == nil {
		// Guild is nil, return to prevent accessing nil memory.
		return
	} else {
		guildID = guild.ID
	}

	// Cycle thru our WatchLogs.
	for _, w := range cfg.watched {
		if w.guildID == guildID {
			watched = w
			break
		}
	}

	// Return since guild ID isn't being watched.
	if watched.guildID == "" {
		return
	}

	// If this isn't the correct channel, return.
	if strings.ToLower(watched.channelName) != strings.ToLower(channel) && !watched.channelAll {
		return
	}

	// Create output.
	output := watched.MessageCreate(msg.Author.Username, msg.Author.Discriminator, channel, msg.ContentWithMentionsReplaced())

	// Send the composed message.
	watched.Talk(output)
}

// messageLogger logs the supplied message into a local database.
func messageLogger(database, databaseID, channel string, msg *discordgo.Message) (bool, error) {

	m := MessageNew(databaseID, channel, msg)
	if ok, err := m.Update(database); err != nil {
		return false, err
	} else if ok {
		ts, err := msg.Timestamp.Parse()
		// TAG: TODO - timestamp error.
		if err != nil {
			fmt.Println("messageLog():" + err.Error())
		}
		if err := UserUpdateSimple(databaseID, msg.Author, 1, ts); err != nil {
			fmt.Println("updating/adding user", err)
		}
		return true, nil
	}
	return false, nil
}

// messageIntegrityCheck verifies the integrity of channels of a specific guild.
func (cfg *Config) messageIntegrityCheck(gName string) (string, error) {
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

				if ok, err := messageLogger(gName, gID, c.Name, m); err != nil {
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
func MessageNew(databaseID, channelName string, m *discordgo.Message) *Message {
	u := UserNew(m.Author)

	var ts, ets time.Time
	var err error
	if ts, err = m.Timestamp.Parse(); err != nil {
		// TAG: TODO
		// Log error that should never happen (according to the documentation)
		// Will be logged with a logger once added.
		fmt.Println("MessageNew() " + err.Error())
	}

	// Check EditedTimestamp if it isn't blank.
	if m.EditedTimestamp != "" {
		if ets, err = m.EditedTimestamp.Parse(); err != nil {
			// TAG: TODO
			// Log error that should never happen (according to the documentation)
			// Will be logged with a logger once added.
			fmt.Println("MessageNew() " + err.Error())
		}
	}

	return &Message{
		ID:              m.ID,
		ChannelID:       m.ChannelID,
		ChannelName:     channelName,
		Content:         m.Content,
		Timestamp:       ts,
		EditedTimestamp: ets,
		Author:          u.Basic(),
	}
}

// Get a message from the database.
func (m *Message) Get(database string) error {
	var q = make(map[string]interface{})
	q["id"] = m.ID

	db := DBdataCreate(database, CollectionMessages, Message{}, q, nil)
	if err := db.dbGet(Message{}); err != nil {
		return err
	}

	// Convert the DB interface{} to a message.
	var msg = Message{}
	msg = db.Document.(Message)
	*m = msg

	return nil
}

// Update Checks and if not exists... Adds to the database.
func (m *Message) Update(database string) (bool, error) {
	var q = make(map[string]interface{})
	q["id"] = m.ID

	db := DBdataCreate(database, CollectionMessages, m, q, nil)
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
