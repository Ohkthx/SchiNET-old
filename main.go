package main

import (
	"bufio"
	"flag"
	"fmt"
	"net"
	"os"
	"strconv"

	mgo "gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"

	"github.com/bwmarrin/discordgo"
	"github.com/d0x1p2/godbot"
)

// Constants used to initiate and customize bot.
var (
	_version       = "0.7.2"
	envToken       = os.Getenv("BOT_TOKEN")
	envDBUrl       = os.Getenv("BOT_DBURL")
	envCMDPrefix   = os.Getenv("BOT_PREFIX")
	envPBDK        = os.Getenv("BOT_PBDevKey")
	envPBPW        = os.Getenv("BOT_PBPW")
	envPB          = os.Getenv("BOT_PB")
	envBotGuild    = os.Getenv("BOT_GUILD")
	consoleDisable bool
	watcherEnabled bool
	watcherPort    string
	watcherHost    string
	cmds           map[string]map[string]string
)

func init() {
	flag.BoolVar(&consoleDisable, "console-disable", false, "Disable Console.")
	flag.BoolVar(&watcherEnabled, "watcher", false, "Watch a Guild/Channel.")
	flag.StringVar(&watcherPort, "port", "", "Port to connect on for watcher.")
	flag.StringVar(&watcherHost, "host", "", "Host to the watcher.")
	flag.Parse()

	// Init commands.
	cmds = make(map[string]map[string]string)
	cmds["admin"] = make(map[string]string)
	cmds["mod"] = make(map[string]string)
	cmds["normal"] = make(map[string]string)

	cmds["admin"]["permission"] = "Add and Remove permissions for a user."
	cmds["admin"]["ban"] = "Soft/Hard/Bot ban a user."
	cmds["admin"]["histo"] = "Prints out server message statistics."

	cmds["mod"]["event"] = "Add/Edit/Remove server events."
	cmds["mod"]["alias"] = "Add/Remove command aliases."
	cmds["mod"]["channel"] = "Enable/Disable commands in current channel."
	cmds["mod"]["clear"] = "Clears messages from current channel. Specify a number."

	cmds["normal"]["script"] = "Add/Edit/Remove scripts for the local server."
	cmds["normal"]["event"] = "View events that are currently scheduled."
	cmds["normal"]["user"] = "Displays stastics of a specified user."
	cmds["normal"]["echo"] = "Echos a message given."
	cmds["normal"]["roll"] = "How's your luck? Rolls 2 6d"
	cmds["normal"]["top10"] = "Are you amongst the great?"
	cmds["normal"]["ally"] = "Ally another guild."
	cmds["normal"]["gen"] = "Generate a pseudo 21x21 map."
	cmds["normal"]["sz"] = "Returns the size of a message."
	cmds["normal"]["invite"] = "Bot invite information!"
	cmds["normal"]["ticket"] = "Add a bug to the ticket system!"
}

// Bot Global interface for pulling discord information.
var Bot *godbot.Core

// Mgo is for the global database session.
var Mgo *mgo.Session

func main() {
	// If it is a watcher, just start the client and return once complete.
	if watcherEnabled {
		// Return if we don't have the information to connect.
		if watcherHost == "" || watcherPort == "" {
			return
		}
		clientLaunch()
		os.Exit(0)
	}

	var cfg = &Config{}

	if envToken == "" {
		return
	}

	var err error
	// Connect to our Database.
	cfg.DB, err = mgo.Dial(envDBUrl)
	if err != nil {
		fmt.Println(err)
		return
	}

	// Create a new instance of the bot.
	bot, err := godbot.New(envToken)
	if err != nil {
		fmt.Println(err)
		return
	}
	cfg.Core = bot

	// Handlers for message changes and additions.
	bot.MessageCreateHandler(cfg.messageCreateHandler)
	bot.MessageUpdateHandler(cfg.messageUpdateHandler)

	// Handlers for guild changes.
	bot.GuildCreateHandler(cfg.guildCreateHandler)
	bot.GuildMemberAddHandler(cfg.guildMemberAddHandler)
	bot.GuildMemberRemoveHandler(cfg.guildMemberRemoveHandler)

	// Start the bot
	err = bot.Start()
	if err != nil {
		fmt.Println(err)
	}

	// Update the Nickname in all of the channels with VERSION appended.
	for _, g := range bot.Guilds {
		err = bot.SetNickname(g.ID, fmt.Sprintf("(v%s)", _version), true)
		if err != nil {
			fmt.Println(err)
		}
	}

	// Assign ugly globals
	// TAG: TODO - fix this by finding an alternative.
	Bot = bot
	Mgo = cfg.DB

	// Process the default bot command aliases.
	if err := cfg.defaultAliases(); err != nil {
		fmt.Println(err)
		os.Exit(0)
	}

	// Load all alliances so that servers will be bridged correctly.
	if err := cfg.AlliancesLoad(); err != nil {
		fmt.Println(err)
		cfg.cleanup()
	}

	// Load all guild configurations to remove frequent database access per message.
	if err = cfg.GuildConfigLoad(); err != nil {
		fmt.Println(err)
		return
	}

	// Run in either silent mode with no output (for background) or with interactive console.
	if !consoleDisable {
		cfg.core()
	} else {
		select {}
	}
}

// cleanup children and stop the bot correctly.
func (cfg *Config) cleanup() {
	// Kill the child processes for the guilds/channels being watched.
	for _, w := range cfg.watched {
		w.Talk(w.pid + "die")
	}

	cfg.Core.Stop()
	cfg.DB.Close()
	fmt.Println("\nBot stopped, exiting.")
	os.Exit(0)
}

// clientLaunch creates a new instance whose purpose is to listen for incoming messages.
func clientLaunch() {
	fmt.Print("Connecting to " + watcherHost + ":" + watcherPort + "... ")
	conn, err := net.Dial("tcp", watcherHost+":"+watcherPort)
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println("Connected!")
	pID := strconv.Itoa(os.Getpid())

	// Send our PID (childs) to the spawning (parent) process.
	conn.Write([]byte(pID + "\n"))

	fmt.Println("<-- " + "Exchanged our PID: " + pID)
	killswitch := pID + "die"

	// Print messages recieved... and break if we get the killswitch "[PID]die"
	for {
		message, _ := bufio.NewReader(conn).ReadString('\n')
		fmt.Print(string(message))
		if stripWhiteSpace(string(message)) == killswitch {
			break
		}

	}

	// Close the socket and exit.
	conn.Close()
}

// Used to verify/register default aliases.
func (cfg *Config) defaultAliases() error {

	type aliasSimple struct {
		caller string
		linked string
	}

	var aliases [4]aliasSimple
	aliases[0] = aliasSimple{"gamble", "user --gamble -n"}
	aliases[1] = aliasSimple{"ban", "user --ban"}
	aliases[2] = aliasSimple{"permission", "user --permission"}
	aliases[3] = aliasSimple{"xfer", "user --xfer"}

	for _, a := range aliases {
		user := UserNew(cfg.Core.User)
		alias := AliasNew(a.caller, a.linked, user)
		if err := alias.Update(); err != nil {
			return err
		}
	}
	return nil
}

// guildCreateHandler Handles newly added guilds that have invited the bot to the server.
func (cfg *Config) guildCreateHandler(s *discordgo.Session, ng *discordgo.GuildCreate) {
	if Bot != nil {
		if err := Bot.UpdateConnections(); err != nil {
			fmt.Println(err)
			return
		}

		user, err := cfg.Core.Session.GuildMember(ng.Guild.ID, ng.OwnerID)
		if err != nil {
			fmt.Println(err)
			return
		}

		var admin = UserNew(user.User)
		if err := admin.Get(admin.ID); err != nil {
			fmt.Println(err)
			return
		}

		// Grant Admin permissions to server Owner.
		admin.PermissionAdd(ng.ID, permAdmin|permModerator|permNormal)
		if err := admin.Update(); err != nil {
			fmt.Println(err)
			return
		}

		var guildConfig = GuildConfig{ID: ng.ID, Name: ng.Name, Init: false, Prefix: envCMDPrefix}
		if err = guildConfig.Get(); err != nil {
			if err != mgo.ErrNotFound {
				fmt.Println("Initiating a new guild: " + err.Error())
			}
		}

		// If it is not initated- notify the chat and the owner.
		if guildConfig.Init == false {
			guildConfig.Init = true

			// Update/Add guild to database and current running config.
			if err = cfg.GuildConfigManager(guildConfig); err != nil {
				fmt.Println("Updating guild on new guild added: " + err.Error())
			}

			// Assign new nickname with current version.
			if err = Bot.SetNickname(ng.ID, fmt.Sprintf("(v%s)", _version), true); err != nil {
				fmt.Println(err)
			}

			// Notify the main channel that the bot has been added and WHO has the ultimate admin privledges.
			embedMsg := embedCreator(fmt.Sprintf("Hello all, nice to meet you!\n<@%s> has been given the **Admin** privledges for <@%s> on this server.\n", admin.ID, cfg.Core.User.ID), ColorYellow)
			s.ChannelMessageSendEmbed(ng.ID, embedMsg)

			// Send a greeting to the Admin informing of the addition.
			if err := cfg.dmAdmin(s, ng.OwnerID, ng.Name); err != nil {
				fmt.Println(err)
			}
		}
	}
}

// GuildConfigLoad loads guild configs into memory for quicker access.
func (cfg *Config) GuildConfigLoad() error {

	// Scan current guilds
	for _, g := range cfg.Core.Guilds {
		var gc = GuildConfig{ID: g.ID}
		if err := gc.Get(); err != nil {
			if err == mgo.ErrNotFound {
				gc.Name = g.Name
				gc.Prefix = envCMDPrefix
				if err = gc.Update(); err != nil {
					return err
				}
				return nil
			}
			return err
		}

		// Add it to the current config structure
		// TAG: TODO - it's updating into DB in GuildConfigManager- potentially remove.
		if err := cfg.GuildConfigManager(gc); err != nil {
			return err
		}
	}
	return nil
}

// GuildConfigManager will append guilds if they're not already in the running config.
func (cfg *Config) GuildConfigManager(guild GuildConfig) error {
	// Find guild and replace with updated version.
	for n, g := range cfg.GuildConf {
		if g.ID == guild.ID {
			cfg.GuildConf[n] = guild
			if err := guild.Update(); err != nil {
				return err
			}
			return nil
		}
	}

	if err := guild.Update(); err != nil {
		return err
	}

	// Guild wasn't found, needs to be appended.
	cfg.GuildConf = append(cfg.GuildConf, guild)
	return nil
}

// GuildConfigByID will search the running guild configurations and return a matching instance.
// If it isn't found, return an empty GuildConfig{}.
func (cfg *Config) GuildConfigByID(gID string) GuildConfig {
	// Scan the current configs.
	for n, g := range cfg.GuildConf {
		if g.ID == gID {
			return cfg.GuildConf[n]
		}
	}

	// Wasn't found- return nil. Caller should check nil value.
	return GuildConfig{}
}

// Get a guild from DB
func (g *GuildConfig) Get() error {
	var q = make(map[string]interface{})

	q["id"] = g.ID

	var dbdat = DBdataCreate(g.Name, CollectionConfig, GuildConfig{}, q, nil)
	err := dbdat.dbGet(GuildConfig{})
	if err != nil {
		return err
	}

	var guild = GuildConfig{}
	guild = dbdat.Document.(GuildConfig)

	if guild.Prefix == "" {
		guild.Prefix = envCMDPrefix
	}

	*g = guild

	return nil
}

// Update a guild's config.
func (g *GuildConfig) Update() error {
	var err error
	var q = make(map[string]interface{})
	var c = make(map[string]interface{})

	if g.Prefix == "" {
		g.Prefix = envCMDPrefix
	}

	q["id"] = g.ID
	c["$set"] = bson.M{
		"id":     g.ID,
		"name":   g.Name,
		"init":   g.Init,
		"prefix": g.Prefix,
	}

	var dbdat = DBdataCreate(g.Name, CollectionConfig, g, q, c)
	err = dbdat.dbEdit(User{})
	if err != nil {
		if err == mgo.ErrNotFound {
			// Add to DB since it doesn't exist.
			if err := dbdat.dbInsert(); err != nil {
				return err
			}
			return nil
		}
		return err
	}

	return nil

}

// dmAdmin sends a whisper to the Admin about the newly added bot. Outfitted with minor instructions- it should help.
func (cfg *Config) dmAdmin(s *discordgo.Session, uID, server string) error {
	var err error
	var msg = fmt.Sprintf("Greetings <@%s>! You have been granted **Admin** privledges for this bot for the "+
		"**%s** server! You can grant additional permissions to other users by using the `,permission` command.\n\n"+
		"To invoke commands, they must be entered on a server channel.\n"+
		"An example of how to grant a moderator permission to another user:\n"+
		"`,permission  --add  --type Moderator  --user <@%s>\n\n"+
		"If you have additional questions, you can always use the `,help` command or join us at %s",
		cfg.Core.User.ID, uID, server, envBotGuild)

	// Create the DM channel
	var channel *discordgo.Channel
	channel, err = s.UserChannelCreate(uID)
	if err != nil {
		return err
	}

	// Send notification/Greeting over the DM channel.
	if _, err = s.ChannelMessageSend(channel.ID, msg); err != nil {
		return err
	}
	return nil
}
