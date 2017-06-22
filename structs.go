package main

import (
	"sync"

	mgo "gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"

	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/d0x1p2/godbot"
)

// Config holds information that needs to be readily accessible.
type Config struct {
	Takeover   bool
	TakeoverID string
	Core       *godbot.Core
	DB         *mgo.Session
}

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
	LastSeen      time.Time `bson:"lastseen"`
}

// Ban contains data pertaining to a ban.
type Ban struct {
	User     *UserBasic
	Channels []chanBan
	Amount   int
	ByLast   *UserBasic
	Last     time.Time `bson:"last"`
}

type chanBan struct {
	Name      string
	ChannelID string
	Comment   string
	By        *UserBasic
	Date      time.Time
}

// UserBasic is just simple information to determine a user.
type UserBasic struct {
	ID            string
	Name          string
	Discriminator string
}

// Command structure for User Defined commands.
type Command struct {
	ID bson.ObjectId `bson:"_id,omitempty"`
}
