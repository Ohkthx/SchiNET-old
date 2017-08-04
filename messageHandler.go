package main

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"gopkg.in/mgo.v2"
)

func (cfg *Config) msghandler(s *discordgo.Session, m *discordgo.MessageCreate) {
	var err error
	var c *discordgo.Channel
	if c, err = s.Channel(m.ChannelID); err != nil {
		fmt.Println(err)
		return
	}

	var io = msgToIOdat(m)

	if Bot != nil {
		// Required for storing information in the correct database.
		io.guild = Bot.GetGuild(c.GuildID)
	}

	if m.Author.Bot {
		if m.Author.ID == Bot.User.ID {
			ts, _ := m.Timestamp.Parse()
			if err := UserUpdateSimple(io.guild.Name, m.Author, 1, ts); err != nil {
				return
			}
		}
		return
	} else if c.IsPrivate {
		if strings.Contains(m.Content, ",list") {
			s.ChannelMessageSend(c.ID, channelsTemp())
		}
		return
		//fmt.Printf("Content: %s\nMentions:%s\n", m.Content, m.Mentions)
	}

	// Log message into Database
	if _, err := messageLog(io.guild.Name, c.Name, m.Message); err != nil {
		fmt.Println(err)
	}
	// End logging message

	cfg.AllianceHandler(m.Message)

	// Return due to not being a command and/or just an Embed.
	if io.command == false || len(io.io) == 0 {
		return
	}

	var u = UserNew(io.guild.Name, m.Author)
	if err := u.Get(m.Author.ID); err != nil {
		fmt.Println(err)
		return
	}

	io.user = u

	var tko bool
	if io.io[0] == "takeover" {
		tko = true
	}
	if ok := cfg.takeoverCheck(m.ID, m.ChannelID, m.Content, tko, u); ok {
		return
	} else if ok := u.HasPermission(permNormal); !ok {
		return
	}

	err = cfg.ioHandler(io)
	if err != nil {
		io.msgEmbed = embedCreator(fmt.Sprintf("%s", err.Error()), ColorMaroon)
		//return
	}
	// Prevention from attempting access of null pointer from console.
	if io.msg != nil && io.rm {
		s.ChannelMessageDelete(io.msg.ChannelID, io.msg.ID)
	}

	// Send message here.
	if io.output != "" {
		_, _ = s.ChannelMessageSend(m.ChannelID, io.output)
	} else if io.msgEmbed != nil {
		s.ChannelMessageSendEmbed(m.ChannelID, io.msgEmbed)
	}

	return
}

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

func delUserHandler(s *discordgo.Session, du *discordgo.GuildMemberRemove) {
	fmt.Println("Got here.")
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

func messageLog(database, channel string, msg *discordgo.Message) (bool, error) {

	m := MessageNew(database, channel, msg)
	if ok, err := m.Update(database); err != nil {
		return false, err
	} else if ok {
		ts, _ := msg.Timestamp.Parse()
		if err := UserUpdateSimple(database, msg.Author, 1, ts); err != nil {
			fmt.Println("updating/adding user", err)
		}
		return true, nil
	}
	return false, nil
}

// MessageIntegrityCheck erifies the integrity of channels of a specific guild.
func (cfg *Config) MessageIntegrityCheck(gName string) (string, error) {
	var gID string
	var found bool
	for _, g := range cfg.Core.Guilds {
		if strings.Contains(g.Name, gName) {
			gName = g.Name
			gID = g.ID
			found = true
			break
		}
	}

	if !found {
		return "", errors.New("could not find guild ID")
	}
	var missed int
	for _, c := range cfg.Core.Links[gID] {
		var mID string
		for {
			var bk bool
			msgs, err := cfg.Core.Session.ChannelMessages(c.ID, 100, mID, "", "")
			if err != nil {
				return "", err
			}

			var cnt = len(msgs)
			if cnt == 0 {
				bk = true
				break
			}

			for n, m := range msgs {
				mID = m.ID

				if ok, err := messageLog(gName, c.Name, m); err != nil {
					fmt.Println("Error logging message", err.Error())
				} else if ok {
					missed++
				}

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
func MessageNew(database, channel string, m *discordgo.Message) *Message {
	u := UserNew(database, m.Author)
	if err := u.Get(m.Author.ID); err != nil {
		// Most likely not found or first message.
		if err1 := u.Update(); err1 != nil {
			// Database error. Log error.
		}
	}

	var ts, ets time.Time
	var err error
	if ts, err = m.Timestamp.Parse(); err != nil {
		// Log error.
	}

	if ets, err = m.EditedTimestamp.Parse(); err != nil {
		// Log error.
	}

	return &Message{
		//Database:    database,
		ID:              m.ID,
		ChannelID:       m.ChannelID,
		ChannelName:     channel,
		Content:         m.Content,
		Timestamp:       ts,
		EditedTimestamp: ets,
		Author:          u.Basic(),
		AuthorMsg:       u.CreditsTotal,
	}
}

// Update Checks and if not exists... Adds to the database.
func (m *Message) Update(database string) (bool, error) {
	var q = make(map[string]interface{})
	q["id"] = m.ID

	db := DBdatCreate(database, CollectionMessages, m, q, nil)
	if err := db.dbGet(Message{}); err != nil {
		if err == mgo.ErrNotFound {
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
