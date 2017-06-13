package main

import (
	"sync"

	"github.com/bwmarrin/discordgo"
	"github.com/d0x1p2/godbot"
)

// Bot is a wrapper for the godbot.Core
type bot struct {
	mu sync.Mutex
	*godbot.Core
}

// IOdat is input/output processed.
type IOdat struct {
	//err  error // Tracking errors.
	help   bool // If HELP is in the input
	io     []string
	input  string
	output string

	user godbot.User
	msg  *discordgo.MessageCreate
}
