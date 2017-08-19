package main

import (
	"flag"
	"fmt"
	"os"

	mgo "gopkg.in/mgo.v2"

	"github.com/bwmarrin/discordgo"
	"github.com/d0x1p2/godbot"
)

// Constants used to initiate and customize bot.
var (
	_version       = "0.7.0"
	envToken       = os.Getenv("BOT_TOKEN")
	envDBUrl       = os.Getenv("BOT_DBURL")
	envCMDPrefix   = os.Getenv("BOT_PREFIX")
	envPBDK        = os.Getenv("BOT_PBDevKey")
	envPBPW        = os.Getenv("BOT_PBPW")
	envPB          = os.Getenv("BOT_PB")
	consoleDisable bool
	cmds           map[string]map[string]string
)

func init() {
	flag.BoolVar(&consoleDisable, "console-disable", false, "Disable Console.")
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
	//var binfo bot
	var cfg = &Config{}

	if envToken == "" {
		return
	}

	bot, err := godbot.New(envToken)
	if err != nil {
		fmt.Println(err)
		return
	}

	cfg.Core = bot
	cfg.DB, err = mgo.Dial(envDBUrl)
	if err != nil {
		fmt.Println(err)
		return
	}

	bot.MessageHandler(cfg.msghandler)
	bot.NewUserHandler(cfg.newUserHandler)
	bot.GuildCreateHandler(cfg.newGuildHandler)
	//bot.RemUserHandler(delUserHandler)
	err = bot.Start()
	if err != nil {
		fmt.Println(err)
	}

	for _, g := range bot.Guilds {
		err = bot.SetNickname(g.ID, fmt.Sprintf("(v%s)", _version), true)
		if err != nil {
			fmt.Println(err)
		}
	}

	Bot = bot
	Mgo = cfg.DB
	if err := cfg.defaultAliases(); err != nil {
		fmt.Println(err)
		os.Exit(0)
	}

	if err := cfg.AlliancesLoad(); err != nil {
		fmt.Println(err)
		cfg.cleanup()
	}

	if !consoleDisable {
		cfg.core()
	} else {
		select {}
	}
}

func (cfg *Config) cleanup() {
	cfg.Core.Stop()
	cfg.DB.Close()
	fmt.Println("Bot stopped, exiting.")
	os.Exit(0)
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
	aliases[3] = aliasSimple{"xfer", "user --xfer "}

	for _, a := range aliases {
		user := UserNew(cfg.Core.User)
		alias := AliasNew(a.caller, a.linked, user)
		if err := alias.Update(); err != nil {
			return err
		}
	}
	return nil
}

func (cfg *Config) newGuildHandler(s *discordgo.Session, ng *discordgo.GuildCreate) {
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
		admin.PermissionAdd(ng.ID, permAdmin|permModerator|permNormal)
		if err := admin.Update(); err != nil {
			fmt.Println(err)
			return
		}
	}
}
