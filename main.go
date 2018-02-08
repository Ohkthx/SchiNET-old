package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"strconv"

	mgo "gopkg.in/mgo.v2"

	"github.com/bwmarrin/discordgo"
	"github.com/d0x1p2/godbot"
)

// Constants used to initiate and customize bot.
var (
	_version       = "0.9.4"
	ConfigFile     ConfigJSON
	helpDocs       = "https://github.com/d0x1p2/SchiNET/blob/master/docs/README.md"
	consoleDisable bool   // Argument to run in background or to run as a console.
	DEBUG          bool   // Argument as to if this is a DEBUGGED session.
	watcherEnabled bool   // Argument for WatchLog being enabled or disabled.
	watcherPort    string // Argument for WatchLog Port.
	watcherHost    string // Argument for WatachLog Host.
	execute        string // Argument for Execute a command in a new window.
	cmds           map[string]map[string]string

	Mgo *mgo.Session // Public access to the MGO drivers.
)

func init() {
	flag.BoolVar(&consoleDisable, "console-disable", false, "Disable Console.")
	flag.BoolVar(&watcherEnabled, "watcher", false, "Watch a Guild/Channel.")
	flag.BoolVar(&DEBUG, "debug", false, "Debugging turned on.")
	flag.StringVar(&watcherPort, "port", "", "Port to connect on for watcher.")
	flag.StringVar(&watcherHost, "host", "", "Host to the watcher.")
	flag.StringVar(&execute, "exec", "", "Execute a console command and exit.")
	flag.Parse()

	// Init commands.
	cmds = make(map[string]map[string]string)
	cmds["admin"] = make(map[string]string)
	cmds["mod"] = make(map[string]string)
	cmds["normal"] = make(map[string]string)

	cmds["admin"]["histo"] = "Prints out server message statistics."
	cmds["admin"]["admin"] = "Allows performing various admin related tasks."
	cmds["mod"]["ticket"] = "Modify trouble tickets placed by users."

	cmds["mod"]["abuse"] = "Add a bot abuser to restrict access."
	cmds["mod"]["event"] = "Add/Edit/Remove server events."
	cmds["mod"]["alias"] = "Add/Remove command aliases."
	cmds["mod"]["clear"] = "Clears messages from current channel. Specify a number."
	cmds["mod"]["ally"] = "Ally another guild."

	cmds["normal"]["script"] = "Add/Edit/Remove scripts for the local server."
	cmds["normal"]["event"] = "View events that are currently scheduled."
	cmds["normal"]["user"] = "Displays stastics of a specified user."
	cmds["normal"]["echo"] = "Echos a message given."
	cmds["normal"]["roll"] = "How's your luck? Rolls 2 6d"
	cmds["normal"]["top10"] = "Are you amongst the great?"
	cmds["normal"]["gen"] = "Generate a pseudo 21x21 map."
	cmds["normal"]["sz"] = "Returns the size of a message."
	cmds["normal"]["invite"] = "Bot invite information!"
	cmds["normal"]["ticket"] = "Add a bug to the ticket system!"
}

func main() {
	// Check if our configuration exists. If not create it.
	ConfigFile = ConfigJSON{}
	if ok := ConfigFile.Processor(); !ok {
		newPause()
		return
	}

	// Use the DEBUG token if launching in debug mode.
	if DEBUG && ConfigFile.Token != "" {
		ConfigFile.Token = ConfigFile.TokenDebug
	} else if DEBUG && ConfigFile.Token == "" {
		fmt.Println("'conf.json' needs debug token specified.")
		return
	}

	// If it is a watcher, just start the client and return once complete.
	if watcherEnabled {
		// Return if we don't have the information to connect.
		if watcherHost == "" || watcherPort == "" {
			return
		}
		clientLaunch()
		os.Exit(0)
	}

	var err error
	var cfg = &Config{}

	// Connect to our Database.
	cfg.DB, err = mgo.Dial(ConfigFile.DBURL)
	if err != nil {
		fmt.Println("Connection to database:", err)
		return
	}

	// Assign ugly globals
	// TAG: TODO - fix this by finding an alternative.
	Mgo = cfg.DB

	// Create a new instance of the bot.
	cfg.Core, err = godbot.New(ConfigFile.Token)
	if err != nil {
		fmt.Println(err)
		return
	}

	if execute != "" {
		cfg.Core.LiteMode = true
		if err = cfg.Core.Start(); err != nil {
			fmt.Println(err)
			newPause()
			return
		}
		cfg.OneTimeExec(execute)
	}

	// Handlers for message changes and additions.
	cfg.Core.MessageCreateHandler(cfg.messageCreateHandler)
	cfg.Core.MessageUpdateHandler(cfg.messageUpdateHandler)

	// Handlers for guild changes.
	cfg.Core.GuildCreateHandler(cfg.guildCreateHandler)
	cfg.Core.GuildRoleUpdateHandler(cfg.guildRoleUpdateHandler)
	cfg.Core.GuildRoleDeleteHandler(cfg.guildRoleDeleteHandler)

	// Handlers for member changes.
	cfg.Core.GuildMemberAddHandler(cfg.guildMemberAddHandler)
	cfg.Core.GuildMemberUpdateHandler(cfg.guildMemberUpdateHandler)
	cfg.Core.GuildMemberRemoveHandler(cfg.guildMemberRemoveHandler)

	// Handlers for channels.
	cfg.Core.ChannelUpdateHandler(cfg.channelUpdateHandler)
	cfg.Core.ChannelDeleteHandler(cfg.channelDeleteHandler)

	// Start the bot
	if err = cfg.Core.Start(); err != nil {
		fmt.Println(err)
		return
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

	// Process all guild configurations and verify...
	for _, g := range cfg.GuildConf {
		// ... verify roles are still correct.
		if err = g.RoleCorrection(cfg.Core.Session); err != nil {
			fmt.Println("Role Correction: " + err.Error())
		}

		// ... verify internal channel is still correct.
		if err = cfg.InternalCorrection(g.ID); err != nil {
			fmt.Println("Internal Correction: " + err.Error())
		}

		// Update guild... even in failure (failed shouldn't be saved).
		if err = g.Update(); err != nil {
			fmt.Println("Role Correction, updating: " + err.Error())
		}

		// Process the default bot command aliases.
		if err := cfg.defaultAliases(g.ID); err != nil {
			fmt.Println("Setting default aliases: ", err.Error())
		}
	}

	// Member Roles update in Database:
	if err = cfg.MemberCorrection(); err != nil {
		fmt.Println("Member Correction: " + err.Error())
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
func (cfg *Config) defaultAliases(serverID string) error {

	type aliasSimple struct {
		caller string
		linked string
	}

	var aliases [5]aliasSimple
	aliases[0] = aliasSimple{"gamble", "user --gamble -n"}
	aliases[1] = aliasSimple{"abuse", "user --abuse --user"}
	aliases[2] = aliasSimple{"xfer", "user --xfer"}
	aliases[3] = aliasSimple{"me", "user"}
	aliases[4] = aliasSimple{"beep", "echo Beep Boop..."}

	for _, a := range aliases {
		user := UserNew(cfg.Core.User)
		alias := AliasNew(a.caller, a.linked, serverID, user)
		if err := alias.Update(); err != nil {
			return err
		}
	}
	return nil
}

// dmAdmin sends a whisper to the Admin about the newly added bot. Outfitted with minor instructions- it should help.
func (cfg *Config) dmAdmin(s *discordgo.Session, uID, server string) error {
	var err error
	var msg = fmt.Sprintf("Greetings <@%s>! You have been granted **Admin** privileges for this bot in the "+
		"**%s** server! You can grant additional permissions to other users by using the roles created by the bot.\n\n"+
		"To invoke commands, they must be entered on a server channel.\n"+
		"An example of how to display basic user information:\n"+
		"`,user`\n\n"+
		"If you have additional questions, you can:\n+ Use the `,help` command\n+ Join us at %s\n+ Read the Easy-To-Use Documentation: %s",
		uID, server, ConfigFile.GuildURL, helpDocs)

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

// Processor is required to load or create a configuration.
func (config *ConfigJSON) Processor() bool {
	fmt.Println("Loading the configuration file: 'config.json'")
	// Check if our configuration exists.
	file, err := os.Open("config.json")
	if err != nil {
		fmt.Println("Error occured! Going to create a configuration file for you.")

		// Create out temporary config.
		DummyConfig := ConfigJSON{DBURL: "127.0.0.1", Prefix: ","}

		*config = DummyConfig

		// Save this temporary config.
		if ok := config.Save(); !ok {
			// Configuration Save failed... pause and return.
			return false
		}

		return false
	}

	// Process our configuration file.
	decoder := json.NewDecoder(file)
	if err = decoder.Decode(&config); err != nil {
		fmt.Println("Error occured while loading configuration:", err)
		return false
	}

	// Validate the configuration.
	if ok := config.Validator(); !ok {
		return false
	}

	return true
}

// Validator confirms a load configuration is good for use.
func (config *ConfigJSON) Validator() bool {
	// Check that our configuration file has data.
	// TODO: Disable features not specified instead of quitting/exiting.
	if ConfigFile.Token == "" {
		fmt.Println("'conf.json' needs a proper discord token.")
	} else if ConfigFile.DBURL == "" {
		fmt.Println("'conf.json' needs a mongodb URL/IP specified.")
	} else if ConfigFile.Prefix == "" {
		fmt.Println("'conf.json' needs a command prefix specified.")
	} else if ConfigFile.GuildURL == "" {
		fmt.Println("'conf.json' needs a URL for the main guild specified.")
	} else if ConfigFile.PastebinAcct == "" {
		fmt.Println("'conf.json' needs a Pastebin user account specified.")
	} else if ConfigFile.PastebinPW == "" {
		fmt.Println("'conf.json' needs your Pastebin password specified.")
	} else if ConfigFile.PastebinToken == "" {
		fmt.Println("'conf.json' needs the Pastebin token.")
	} else {
		return true
	}

	return false
}

// Save the configuration.
func (config *ConfigJSON) Save() bool {
	// Convert our configuration to characters with proper indention.
	prettyJSON, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		fmt.Println("Error occured while converting converting configuration:", err)
		return false
	}

	// Write the converted configuration.
	if err := ioutil.WriteFile("config.json", prettyJSON, 0644); err != nil {
		fmt.Println("Error occured while writing to configuration file:", err)
		return false
	}

	fmt.Println("Configuration saved to: 'config.json'")

	return true
}
