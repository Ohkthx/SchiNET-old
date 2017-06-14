package main

import "github.com/bwmarrin/discordgo"

func msghandler(s *discordgo.Session, m *discordgo.MessageCreate) {

	if m.Author.Bot {
		return
	}

	var io = msgToIOdat(m)
	var err = io.ioHandler()
	if err != nil {
		// Log error here.
		return
	}

	// Prevention from attempting access of null pointer from console.
	if io.msg != nil && io.rm {
		s.ChannelMessageDelete(io.msg.ChannelID, io.msg.ID)
	}

	// Send message here.
	if len(io.io) > 0 && io.output != "" {
		_, _ = s.ChannelMessageSend(m.ChannelID, io.output)
	}

	return
}
