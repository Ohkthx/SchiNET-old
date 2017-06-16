package main

import (
	"fmt"
	"os"

	mgo "gopkg.in/mgo.v2"

	"github.com/d0x1p2/godbot"
)

// Constants used to initiate and customize bot.
var (
	_version     = "0.2.1"
	envToken     = os.Getenv("BOT_TOKEN")
	envDBUrl     = os.Getenv("BOT_DBURL")
	envCMDPrefix = os.Getenv("BOT_PREFIX")
)

// Bot Global interface for pulling discord information.
var Bot *godbot.Core

// Mgo is for the global database session.
var Mgo *mgo.Session

func main() {

	var binfo bot

	if envToken == "" {
		return
	}

	bot, err := godbot.New(envToken)
	if err != nil {
		fmt.Println(err)
		return
	}

	bot.MessageHandler(msghandler)
	err = bot.Start()
	if err != nil {
		fmt.Println(err)
	}

	binfo.Core = bot
	Bot = bot
	Mgo, err = mgo.Dial(envDBUrl)
	if err != nil {
		fmt.Println(err)
		return
	}

	err = messagesProcessStartup()
	if err != nil {
		fmt.Println("messageProcessStartup()", err)
		return
	}

	binfo.core()
}

func (b *bot) cleanup() {
	b.Stop()
	Mgo.Close()
	fmt.Println("Bot stopped, exiting.")
	os.Exit(0)
}
