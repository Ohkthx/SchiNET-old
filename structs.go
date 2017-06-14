package main

import (
	"sync"

	"gopkg.in/mgo.v2/bson"

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
	rm     bool // Remove initial message.
	io     []string
	input  string
	output string

	user godbot.User
	msg  *discordgo.MessageCreate
}

// DBdat passes information as to what to store into a database.
type DBdat struct {
	Database   string
	Collection string
	Document   interface{}
	Query      bson.M
}

// Event has information regarding upcoming events.
type Event struct {
	Description string
	Day         string
	Time        string
	AddedBy     godbot.User
}
