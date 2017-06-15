package main

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
)

func msghandler(s *discordgo.Session, m *discordgo.MessageCreate) {
	var err error
	c, err := s.Channel(m.ChannelID)
	if err != nil {
		return
	}

	if m.Author.Bot {
		return
	} else if c.IsPrivate == false {
		var d *DBMsg
		d, err = messagesGet(m.ChannelID)
		if err == nil {
			d.MIDr = m.ID
			d.MTotal++
			d.Content = m.Content
			err = messagesUpdate(d)
		}
		if err != nil {
			fmt.Println("updating recent message db", err)
		}
	}

	var io = msgToIOdat(m)
	err = io.ioHandler()
	if err != nil {
		// Log error here.
		return
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
