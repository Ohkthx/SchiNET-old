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

// Message holds basic information related to a specific message.
type Message struct {
	ID              string
	ChannelName     string
	ChannelID       string
	Content         string
	EditedContent   []string
	Timestamp       time.Time
	EditedTimestamp time.Time
	Author          *User
	AuthorMsg       int
}

// Event has information regarding upcoming events.
type Event struct {
	ID          bson.ObjectId `bson:"_id,omitempty"`
	Description string
	Day         string
	HHMM        string
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
	Server        string
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

// Alias contains information regarding an alias link.
type Alias struct {
	ID      bson.ObjectId `bson:"_id,omitempty"`
	Caller  string        // String that calls the alias.
	Linked  string        // What the real command is.
	AddedBy *User         // Person who added alias.
}
