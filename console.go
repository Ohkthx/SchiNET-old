package main

import (
	"bufio"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

// Hold console information
type console struct {
	config *Config
	input  []string
}

func (cfg *Config) core() {
	var err error
	reader := bufio.NewReader(os.Stdin)

	//channelsTemp()

	for {
		fmt.Printf("[%s] > ", time.Now().Format(time.Stamp))
		input, _ := reader.ReadString('\n')
		_, s := strToCommands(input)
		iodat := sliceToIOdat(cfg.Core, s)

		if len(iodat.io) > 0 {
			if iodat.io[0] == "exit" {
				break
			} else {
				con := console{config: cfg, input: iodat.io}
				if err = con.Parser(); err != nil {
					fmt.Println(err)
					continue
				}
			}
			/*
				err = cfg.ioHandler(iodat)
				if err != nil {
					fmt.Println(err)
				}
			*/
		}
	}

	// Cleanup here from "exit"
	cfg.cleanup()
}

// Parser is decides the correct action to be taken based on input into the console.
func (con *console) Parser() error {
	var multi bool
	// Validate input is sufficient.
	if len(con.input) < 1 {
		return nil
	} else if len(con.input) > 1 {
		multi = true
	}

	var ErrNotEnough = errors.New("Not enough commands")
	switch strings.ToLower(con.input[0]) {
	// Check archives a Guild's(Server) messages into local database.
	case "check":
		if !multi {
			return ErrNotEnough
		}
		return con.channelCheck(con.input[1])

	// Reset can be used to override things such as credits, etc
	case "reset":
		if !multi {
			return ErrNotEnough
		}
		return con.Reset(con.input[1])

	// Watch will set a logger on a particular Guild(Server) and a specific channel.
	case "watch":
		return con.Watch()

	// Kill a watcher/logger.
	case "kill":
		return con.WatchKill()

	case "help":
		fallthrough
	default:
		fmt.Print(consoleHelp())
	}

	return nil
}

// channelCheck archives a particular guild's messages into a local database.
func (con *console) channelCheck(guild string) error {
	msg, err := con.config.MessageIntegrityCheck(guild)
	if err != nil {
		return err
	}
	fmt.Println(msg)
	return nil
}

// Resets credits or whatever else is added here eventually.
func (con *console) Reset(item string) error {
	if con.input[1] == "credits" {
		if _, err := creditsReset(); err != nil {
			return err
		}
		fmt.Println("Complete.")
		return nil
	}
	return errors.New("unknown thing to reset")
}

// Watch -es a Guild and a Channel (if specified). It will launch a new window
// with output from what is selected.
func (con *console) Watch() error {

	var guilds = con.config.Core.Guilds
	// List available guilds.
	fmt.Println("Select a guild by number: ")
	for n, g := range guilds {
		fmt.Printf(" [%2d] %s\n", n, g.Name)
	}

	var num = -1
	var input string
	var err error
	reader := bufio.NewReader(os.Stdin)

	// Run until either ",quit" or a valid option is selected.
	for num < 0 {
		fmt.Print("Guild number [,exit to exit]: ")
		input, _ = reader.ReadString('\n')
		input = stripWhiteSpace(input)
		num, err = strconv.Atoi(input)
		if strings.ToLower(input) == ",quit" {
			break
		} else if err != nil {
			num = -1
			continue
		} else if num >= 0 && num < len(guilds) {
			break
		}
	}

	if strings.ToLower(input) == ",quit" {
		return nil
	}

	var watched WatchLog
	watched.guildID = guilds[num].ID
	watched.guildName = guilds[num].Name

	var i int
	var channels []*discordgo.Channel
	var channelsTotal = con.config.Core.Links[watched.guildID]

	fmt.Println("Select a channel by number: ")
	for _, c := range channelsTotal {
		if c.Type == 0 {
			fmt.Printf(" [%2d] %s\n", i, c.Name)
			i++
			channels = append(channels, c)
		}
	}

	num = -1

	// Run until either ",quit", ",all", or a valid integer is provided.
	for num < 0 {
		fmt.Print("Channel number [,all or all / ,exit to exit]: ")
		input, _ = reader.ReadString('\n')
		input = stripWhiteSpace(input)
		num, err = strconv.Atoi(input)
		if strings.ToLower(input) == ",all" || strings.ToLower(input) == ",exit" {
			break
		} else if err != nil {
			num = -1
			continue
		} else if num >= 0 && num < len(channels) {
			break
		}
	}

	if strings.ToLower(input) == ",exit" {
		return nil
	} else if strings.ToLower(input) != ",all" && num >= 0 && num < len(channels) {
		watched.channelID = channels[num].ID
		watched.channelName = channels[num].Name
	} else {
		// ",all" or a bad number somehow provided.
		watched.channelID = ""
		watched.channelName = ""
	}

	// Start the channel (for communicating to go routine),
	// add to our list of watchers and start the actual server.
	watched.channel = make(chan string)
	con.config.watched = append(con.config.watched, watched)
	go con.watchServer(watched)

	return nil
}

// WatchKill a particular watch/logger.
func (con *console) WatchKill() error {
	fmt.Println("Select a watcher to kill")
	for n, w := range con.config.watched {
		fmt.Printf(" [%2d] %s ", n, w.guildName)
		if w.channelName != "" {
			fmt.Printf("-> %s", w.channelName)
		}
		fmt.Print("\n")
	}

	var input string
	var num = -1
	reader := bufio.NewReader(os.Stdin)

	// Run until a valid number is provided to kill OR ",exit" to cancel.
	for num < 0 {
		fmt.Print("Number to kill: ")
		input, _ = reader.ReadString('\n')
		num, _ = strconv.Atoi(stripWhiteSpace(input))

		// Break and return to main console loop.
		if strings.ToLower(stripWhiteSpace(input)) == ",exit" {
			break
		}

		// Send the "[PID]die" string to tell the distant logger to exit and close.
		if num >= 0 && num < len(con.config.watched) {
			w := con.config.watched[num]
			w.Talk(w.pid + "die")

			break
		}
	}
	return nil
}

// watchServer launches a need server that a client can connect to- to get information.
func (con *console) watchServer(watch WatchLog) {
	fmt.Println("Starting new WatchLog...")

	var portBase = 8444
	// Get an open port.
	var portN = portBase
	var passes = 0
	for {
		passes++
		for _, w := range con.config.watched {
			if w.port == portN {
				portN++
			}
		}
		if passes > 2 {
			break
		}
	}

	hostS := "127.0.0.1"
	portS := strconv.Itoa(portN)

	// Launch a new window (WINDOWS SPECIFIC- NEED CHANGING.)
	cmd := exec.Command("cmd", "/C", "start", "usmbot.exe", "-watcher", "-host", hostS, "-port", portS)
	if err := cmd.Start(); err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println("Granting " + hostS + ":" + portS + " to connect too.")
	ln, _ := net.Listen("tcp", ":"+portS)

	conn, _ := ln.Accept()

	var init = "Initiated: " + watch.guildName
	if watch.channelName != "" {
		init += " on " + watch.channelName
	}

	// Retrieve the client's (CHILD) PID.
	pID, _ := bufio.NewReader(conn).ReadString('\n')
	pID = stripWhiteSpace(pID)
	cmd.Process.Pid, _ = strconv.Atoi(pID)

	// Unneeded currently.
	//con.config.children = append(con.config.children, cmd)

	fmt.Println("Child's PID: " + pID)

	var watchedN int
	// Find the correct watcher, and assign the PID supplied by child.
	for n, w := range con.config.watched {
		if w.guildID == watch.guildID {
			watchedN = n
			con.config.watched[n].pid = pID
			con.config.watched[n].port = portN
		}
	}

	// Send our "Initiated" string to child.
	conn.Write([]byte("--> " + init + "\n"))

	// Loop until "[PID]die" is supplied to terminate connection.
	for msg := range watch.channel {
		conn.Write([]byte(msg + "\n"))
		if msg == pID+"die" {
			break
		}
	}

	// Remove it from the active watchers slices.
	con.config.watched[watchedN] = con.config.watched[len(con.config.watched)-1]
	con.config.watched = con.config.watched[:len(con.config.watched)-1]

	// Close the socket.
	close(watch.channel)
	conn.Close()
	fmt.Println("PID [" + pID + "] killed.")
}

// Talk sends a message over a channel.
func (watch WatchLog) Talk(msg string) {
	watch.channel <- msg
}

func consoleHelp() string {
	text := [...]string{"check", "watch", "reset", "kill", "help"}

	var retText string
	for n, w := range text {
		retText += fmt.Sprintf("%10s ", w)
		if (n+1)%4 == 0 && n != 0 {
			retText += "\n"
		}
	}
	return retText + "\n"
}
