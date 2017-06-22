package main

import (
	"errors"
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
	"gopkg.in/mgo.v2"
)

// Message just holds a message.
type Message struct {
	*discordgo.Message
	Database    string
	ChannelName string
}

func msghandler(s *discordgo.Session, m *discordgo.MessageCreate) {
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
	} else {
		// Log message into Database
		if err := messageLog(io.guild.Name, c.Name, m.Message); err != nil {
			fmt.Println(err)
		}
		// End logging message

		if io.command == false {
			ts, _ := m.Timestamp.Parse()
			err = UserUpdateSimple(io.guild.Name, m.Author, 1, ts)
			if err != nil {
				fmt.Println("updating users credits", err)
			}
		}
		var d *DBMsg
		d, err = messagesGet(m.ChannelID)
		if err != nil {
			if err != mgo.ErrNotFound {
				fmt.Println(err)
			} else {
				d = &DBMsg{
					MIDr:    m.ID,
					MIDf:    m.ID,
					Content: m.Content,
					ID:      m.ChannelID,
					MTotal:  1,
				}
				err = messagesUpdate(d)
				if err != nil {
					fmt.Println(err)
				}
			}
		} else {
			d.MIDr = m.ID
			d.MTotal++
			d.Content = m.Content
			err = messagesUpdate(d)
			if err != nil {
				fmt.Println(err)
			}
		}
	}

	if io.command == false {
		return
	}

	var u = UserNew(m.Author)
	if err := u.Get(io.guild.Name, m.Author.ID); err != nil {
		fmt.Println(err)
		return
	}

	if ok := u.HasPermission(permNormal); !ok {
		return
	}
	err = io.ioHandler()
	if err != nil {
		io.msgEmbed = embedCreator(fmt.Sprintf("%s", err.Error()), ColorMaroon)
		//return
	}

	// Prevention from attempting access of null pointer from console.
	if io.msg != nil && io.rm {
		s.ChannelMessageDelete(io.msg.ChannelID, io.msg.ID)
	}

	// Send message here.
	if len(io.io) > 0 && io.output != "" && io.msgEmbed == nil {
		_, _ = s.ChannelMessageSend(m.ChannelID, io.output)
	} else if io.msgEmbed != nil {
		s.ChannelMessageSendEmbed(m.ChannelID, io.msgEmbed)
	}

	return
}

func newUserHandler(s *discordgo.Session, nu *discordgo.GuildMemberAdd) {
	if Bot != nil {
		c := Bot.GetMainChannel(nu.GuildID)
		msg := fmt.Sprintf("Welcome to the server, <@%s>!", nu.User.ID)
		s.ChannelMessageSendEmbed(c.ID, embedCreator(msg, ColorBlue))
	}
}

func messageLog(database, channel string, m *discordgo.Message) error {
	db := DBdatCreate(database, CollectionMessages(channel), m, nil, nil)
	if err := db.dbInsert(); err != nil {
		return err
	}
	return nil
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
			var ok, bk bool
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
				msg := MessageNew(gName, c.Name, m)
				if ok, err = msg.Update(); err != nil {
					return "", err
				}
				if ok {
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
	return &Message{
		Database:    database,
		ChannelName: channel,
		Message:     m,
	}
}

// Update Checks and if not exists... Adds to the database.
func (m *Message) Update() (bool, error) {
	var q = make(map[string]interface{})
	q["id"] = m.ID

	db := DBdatCreate(m.Database, CollectionMessages(m.ChannelName), m.Message, q, nil)
	if err := db.dbGet(discordgo.Message{}); err != nil {
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
