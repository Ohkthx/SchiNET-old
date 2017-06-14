package main

import (
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/d0x1p2/godbot"
)

func strToCommands(io string) (bool, []string) {

	var slice []string
	var help bool
	s := strings.Fields(io)

	for _, w := range s {
		if strings.HasPrefix(w, ",") {
			w = strings.TrimPrefix(w, ",")
		}
		if strings.ToLower(w) != "help" {
			slice = append(slice, w)
		} else {
			help = true
		}
	}

	return help, slice
}

func msgToIOdat(msg *discordgo.MessageCreate) *IOdat {
	var io IOdat

	io.help, io.io = strToCommands(msg.Content)
	io.input = msg.Content
	io.user.User = msg.Author
	io.msg = msg

	return &io
}

func sliceToIOdat(b *godbot.Core, s []string) *IOdat {
	var io IOdat
	io.user = b.GetBotUser()
	io.help, io.io = strToCommands(strings.Join(s, " "))

	return &io
}

func tsConvert(ts string) string {
	a := strings.FieldsFunc(fmt.Sprintf("%s", ts), tsSplit)
	return fmt.Sprintf("%s %s", a[0], a[1])
}

func tsSplit(r rune) bool {
	return r == 'T' || r == '.' || r == '+'
}

func (io *IOdat) ioHandler() (err error) {
	command := io.io[0]
	switch strings.ToLower(command) {
	case "roll":
		io.miscRoll()
	case "top10":
		io.miscTop10()
	case "echo":
		io.output = strings.Join(io.io[1:], " ")
		return
	}
	return
}
