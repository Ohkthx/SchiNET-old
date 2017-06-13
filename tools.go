package main

import (
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/d0x1p2/godbot"
)

func strToSlice(io string) []string {

	var slice []string
	s := strings.Fields(io)

	for _, w := range s {
		if strings.HasPrefix(w, ",") {
			w = strings.TrimPrefix(w, ",")
		}
		slice = append(slice, w)
	}

	return slice
}

func msgToIOdat(msg *discordgo.MessageCreate) *IOdat {
	var io IOdat

	io.io = strToSlice(msg.Content)
	io.input = msg.Content
	io.user.User = msg.Author
	io.msg = msg

	// Extract a help value.
	for _, w := range io.io {
		if strings.ToLower(w) == "help" {
			io.help = true
			break
		}
	}

	return &io
}

func sliceToIOdat(b *godbot.Core, s []string) *IOdat {
	var io IOdat
	io.user = b.GetBotUser()
	io.io = strToSlice(strings.Join(s, " "))

	// Extract a help value.
	for _, w := range io.io {
		if strings.ToLower(w) == "help" {
			io.help = true
			break
		}
	}

	return &io
}

func (io *IOdat) ioHandler() (err error) {
	command := io.io[0]
	switch strings.ToLower(command) {
	case "echo":
		io.output = strings.Join(io.io[1:], " ")
		return
	}
	return
}
