package main

import (
	"sync"

	mgo "gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"

	"time"

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

	user     *User
	guild    *godbot.Guild
	msg      *discordgo.MessageCreate
	msgEmbed *discordgo.MessageEmbed
}

// DBdat passes information as to what to store into a database.
type DBdat struct {
	Handler    *mgo.Session
	Database   string
	Collection string
	Document   interface{}
	Documents  []interface{}
	Query      bson.M
	Change     bson.M
}

// Event has information regarding upcoming events.
type Event struct {
	ID          bson.ObjectId `bson:"_id,omitempty"`
	Description string
	Day         string
	Time        time.Time
	Protected   bool
	AddedBy     *User
}

// User is a wrapper with additional functionality.
type User struct {
	ID            string
	Username      string
	Discriminator string
	Bot           bool
	Credits       int
	CreditsTotal  int
}

// DBMsg stores information on messages last processed.
type DBMsg struct {
	ID      string
	MTotal  int
	MIDr    string // Message ID of most recent.
	MIDf    string // Message ID of first message.
	Content string
}

//DBHandler Stores a MongoDB connection.
type DBHandler struct {
	*mgo.Session
}

// Command structure for User Defined commands.
type Command struct {
	ID bson.ObjectId `bson:"_id,omitempty"`
}
