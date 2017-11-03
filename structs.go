package main

import (
	"os/exec"
	"sync"

	mgo "gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"

	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/d0x1p2/godbot"
)

// Config holds information that needs to be readily accessible.
type Config struct {
	Core     *godbot.Core
	DB       *mgo.Session
	DSession *discordgo.Session

	// Server Configs
	GuildConf []*GuildConfig

	// Alliance slices
	Alliances []Alliance
	pending   []Alliance // Pending alliances.

	// Watched Guilds/Channels
	watched  []WatchLog
	children []*exec.Cmd
}

// Bot is a wrapper for the godbot.Core
type bot struct {
	mu sync.Mutex
	*godbot.Core
}

// IOdata is input/output processed.
type IOdata struct {
	//err  error // Tracking errors.
	cmdPrefix string
	command   bool // Flag toggling if it is a command or not.
	rm        bool // Remove initial message.
	io        []string
	input     string
	output    string

	user        *User
	guild       *godbot.Guild
	guildConfig *GuildConfig
	msg         *discordgo.MessageCreate
	msgEmbed    *discordgo.MessageEmbed
}

// GuildConfig is used to save basic information regarding if a guild is active or not.
type GuildConfig struct {
	ID     string
	Name   string
	Init   bool
	Roles  []Role
	Prefix string // Command prefix. Defaults to: ","
}

// Role is a struct containing special roles maintained by the bot.
type Role struct {
	ID    string `bson:"id"`
	Name  string
	Value int // Modified permissions
	Base  int // Stock permissions
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
	Author          UserBasic
	// AuthorMsg       int   // Removed for now (was message count)
}

// Event has information regarding upcoming events.
type Event struct {
	ID          bson.ObjectId `bson:"_id,omitempty"`
	Server      string
	Description string
	Day         string
	HHMM        string
	Time        time.Time
	Protected   bool
	AddedBy     UserBasic
}

// EventSmall -er version of Events, used for display.
type EventSmall struct {
	Hours       int
	Minutes     int
	Time        time.Time
	Description string
}

// User is a wrapper with additional functionality.
type User struct {
	ID            string `bson:"id"`
	Username      string
	Discriminator string
	Roles         []string
	Bot           bool
	Credits       int
	CreditsTotal  int
	LastSeen      time.Time `bson:"lastseen"`
}

// Access holds guild/server specific information about the user.
type Access struct {
	ServerID    string
	ServerName  string
	Permissions int
}

type chanBan struct {
	Name      string
	ChannelID string
	Comment   string
	By        *UserBasic
	Date      time.Time
}

// ChannelInfo represents a channel... lol
type ChannelInfo struct {
	ID      string
	Name    string
	Server  string
	Enabled bool
}

// UserBasic is just simple information to determine a user.
type UserBasic struct {
	ID            string
	Server        string
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
	AddedBy UserBasic     // Person who added alias.
}

// WatchLog holds basic information about a channel/server to be watched.
type WatchLog struct {
	guildID   string
	guildName string

	channelID   string
	channelName string
	channelAll  bool

	channel chan string
	pid     string
	port    int
}
