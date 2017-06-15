package main

import (
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/d0x1p2/godbot"
)

// Color constants for embeded messages.
const (
	ColorMaroon = 0x800000
	ColorGreen  = 0x3B8040
	ColorBlue   = 0x5B6991
	ColorBlack  = 0x000000
	ColorGray   = 0x343434
	ColorYellow = 0xFEEB65
)

func strToCommands(io string) (bool, []string) {

	var slice []string
	var help bool
	var quoted bool
	var quote string

	s := strings.Fields(io)

	for _, w := range s {
		if strings.HasPrefix(w, ",") {
			w = strings.TrimPrefix(w, ",")
		} else if strings.HasPrefix(w, "\"") && quoted == false {
			w = strings.TrimPrefix(w, "\"")
			quote += w + " "
			quoted = true
		} else if strings.HasSuffix(w, "\"") && quoted {
			w = strings.TrimSuffix(w, "\"")
			quote += w
			w = quote
			quoted = false

		} else if quoted {
			quote += w + " "
		}

		if strings.ToLower(w) == "help" {
			help = true
		} else if quoted == false {
			slice = append(slice, w)
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
	io.user.User = b.User
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
	case "event", "events":
		fallthrough
	case "add", "del", "edit":
		err = io.dbCore()
	case "echo":
		io.output = strings.Join(io.io[1:], " ")
		return
	}
	return
}

func embedCreator(description string, color int) *discordgo.MessageEmbed {
	return &discordgo.MessageEmbed{
		Author:      &discordgo.MessageEmbedAuthor{},
		Color:       color,
		Description: description,
		Fields:      []*discordgo.MessageEmbedField{},
	}
}
