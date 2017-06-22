package main

import (
	"fmt"
	"os"

	mgo "gopkg.in/mgo.v2"

	"github.com/d0x1p2/godbot"
)

// Constants used to initiate and customize bot.
var (
	_version     = "0.4.3"
	envToken     = os.Getenv("BOT_TOKEN")
	envDBUrl     = os.Getenv("BOT_DBURL")
	envCMDPrefix = os.Getenv("BOT_PREFIX")
	envPBDK      = os.Getenv("BOT_PBDevKey")
	envPBPW      = os.Getenv("BOT_PBPW")
	envPB        = os.Getenv("BOT_PB")
)

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

	//binfo.Core = bot
	Bot = bot

	Mgo = cfg.DB

	/*
		err = messagesProcessStartup()
		if err != nil {
			fmt.Println("messageProcessStartup()", err)
			return
		}
	*/
	cfg.core()
}

func (cfg *Config) cleanup() {
	cfg.Core.Stop()
	cfg.DB.Close()
	fmt.Println("Bot stopped, exiting.")
	os.Exit(0)
}
