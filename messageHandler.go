package main

import (
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
	"gopkg.in/mgo.v2"
)

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
