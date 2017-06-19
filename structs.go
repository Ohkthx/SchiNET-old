package main

import (
	"sync"

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
	help    bool // If HELP is in the input
	command bool // Flag toggling if it is a command or not.
	rm      bool // Remove initial message.
	io      []string
	input   string
	output  string

	user     *User
	guild    *godbot.Guild
	msg      *discordgo.MessageCreate
	msgEmbed *discordgo.MessageEmbed
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
	ID            string `bson:"id"`
	Username      string
	Discriminator string
	Access        int
	Bot           bool
	Credits       int
	CreditsTotal  int
}

// Command structure for User Defined commands.
type Command struct {
	ID bson.ObjectId `bson:"_id,omitempty"`
}
