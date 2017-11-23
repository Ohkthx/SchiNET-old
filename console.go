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
	cmdPrefix := envCMDPrefix

	for {
		fmt.Printf("[%s] > ", time.Now().Format(time.Stamp))
		input, _ := reader.ReadString('\n')
		_, s := strToCommands(input, cmdPrefix)
		iodat := sliceToIOdata(cfg.Core, s, cmdPrefix)

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
	case "spawn":
		if !multi {
			return ErrNotEnough
		}
		return con.Spawner(con.input[1:])
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

// Spawner launches a new command prompt to perform the command in.
func (con *console) Spawner(input []string) error {
	// Launch a new window (WINDOWS SPECIFIC- NEED CHANGING.)
	// TAG: TODO - *nix compatible.
	cmd := exec.Command("cmd", "/C", "start", "SchiNET.exe", "-exec", "\""+strings.Join(input, " ")+"\"")

	if err := cmd.Start(); err != nil {
		return err
	}

	return nil
}

// OneTimeExec a series of commands then promptly exit.
func (cfg *Config) OneTimeExec(input string) {
	var err error
	cmdPrefix := envCMDPrefix

	_, s := strToCommands(input, cmdPrefix)
	iodat := sliceToIOdata(cfg.Core, s, cmdPrefix)

	if len(iodat.io) > 0 {
		con := console{config: cfg, input: iodat.io}
		if err = con.Parser(); err != nil {
			fmt.Println(err)
		}
	}

	newPause()

	// Cleanup here from "exit"
	cfg.cleanup()
}

// channelCheck archives a particular guild's messages into a local database.
func (con *console) channelCheck(guild string) error {
	msg, err := con.config.messageIntegrityCheck(guild)
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
	fmt.Printf(" [%2d] %s\n", 0, "Private Messages")
	for n, g := range guilds {
		fmt.Printf(" [%2d] %s\n", n+1, g.Name)
	}

	var num = -1
	var input string
	var err error
	reader := bufio.NewReader(os.Stdin)

	// Run until either ",quit" or a valid option is selected.
	for num < 0 {
		fmt.Print("Guild number [type 'exit' to exit]: ")
		input, _ = reader.ReadString('\n')
		input = stripWhiteSpace(input)
		num, err = strconv.Atoi(input)
		if strings.ToLower(input) == "exit" {
			break
		} else if err != nil {
			num = -1
			continue
		} else if num >= 0 && num <= len(guilds) {
			break
		}
	}

	if strings.ToLower(input) == "exit" {
		return nil
	}

	var watched WatchLog
	if num == 0 {
		watched.guildID = "private"
		watched.guildName = "private"
	} else {

		watched.guildID = guilds[num-1].ID
		watched.guildName = guilds[num-1].Name
		// Reset num (for channel check.)
		num = -1
	}

	var channels []*discordgo.Channel
	// This will be skipped if opted for private messages due to num not being reset.
	if num < 0 {
		var i = 1

		var channelsTotal = con.config.Core.Links[watched.guildID]
		fmt.Println("Select a channel by number: ")
		for _, c := range channelsTotal {
			if c.Type == 0 {
				fmt.Printf(" [%2d] %s\n", i, c.Name)
				i++
				channels = append(channels, c)
			}
		}
	}

	// Run until either ",quit", ",all", or a valid integer is provided.
	for num < 0 {
		fmt.Print("Channel number ['0' for all / 'exit' to exit]: ")
		input, _ = reader.ReadString('\n')
		input = stripWhiteSpace(input)
		num, err = strconv.Atoi(input)
		if strings.ToLower(input) == "exit" {
			break
		} else if err != nil {
			num = -1
			continue
		} else if num >= 0 && num <= len(channels) {
			break
		}
	}

	if strings.ToLower(input) == "exit" {
		return nil
	} else if num > 0 && num <= len(channels) {
		watched.channelID = channels[num-1].ID
		watched.channelName = channels[num-1].Name
	} else {
		// Number should be 0 to indicate "all"
		watched.channelAll = true
	}

	num = -1
	for num < 0 {
		fmt.Print("Amount of messages to pull from database: ")
		input, _ = reader.ReadString('\n')
		input = stripWhiteSpace(input)
		if num, err = strconv.Atoi(input); err != nil {
			continue
		}
	}

	// Start the channel (for communicating to go routine),
	// add to our list of watchers and start the actual server.
	watched.channel = make(chan string)
	con.config.watched = append(con.config.watched, watched)

	var serverStarted chan string
	serverStarted = make(chan string)

	go con.watchServer(watched, serverStarted, num)

	msg := <-serverStarted
	close(serverStarted)
	fmt.Print("Server status: " + msg + "\n")

	// Send archived messages desired.
	if err := con.watchGetLast(watched, num); err != nil {
		fmt.Println("Processing archived messages: " + err.Error())
	}

	return nil
}

// WatchKill a particular watch/logger.
func (con *console) WatchKill() error {
	fmt.Println("Select a watcher to kill")
	for n, w := range con.config.watched {
		fmt.Printf(" [%2d] %s ", n+1, w.guildName)
		if w.channelName != "" {
			fmt.Printf("-> %s", w.channelName)
		}
		fmt.Print("\n")
	}

	var input string
	var num = 0
	reader := bufio.NewReader(os.Stdin)

	// Run until a valid number is provided to kill OR "exit" to cancel.
	for num < 1 {
		fmt.Print("Number to kill ['exit' to exit]: ")
		input, _ = reader.ReadString('\n')
		num, _ = strconv.Atoi(stripWhiteSpace(input))

		// Break and return to main console loop.
		if strings.ToLower(stripWhiteSpace(input)) == "exit" {
			break
		}

		// Send the "[PID]die" string to tell the distant logger to exit and close.
		if num > 0 && num <= len(con.config.watched) {
			w := con.config.watched[num-1]
			w.Talk(w.pid + "die")

			break
		}
	}
	return nil
}

// watchServer launches a need server that a client can connect to- to get information.
func (con *console) watchServer(watch WatchLog, serverStarted chan string, amount int) {
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
	// TAG: TODO - *nix compatible.
	cmd := exec.Command("cmd", "/C", "start", "SchiNET.exe", "-watcher", "-host", hostS, "-port", portS)
	if err := cmd.Start(); err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println("Granting " + hostS + ":" + portS + " to connect too.")
	ln, err := net.Listen("tcp", ":"+portS)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer ln.Close()

	conn, _ := ln.Accept()

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

	// Initiated text sent to client.
	var init = "Initiated: " + watch.guildName
	if watch.channelName != "" {
		init += " on " + watch.channelName
	}

	// Send our "Initiated" string to child.
	conn.Write([]byte("--> " + init + "\n\n"))

	serverStarted <- "ok"

	// Loop until "[PID]die" is supplied to terminate connection.
	for msg := range watch.channel {
		_, err = conn.Write([]byte(msg + "\n"))
		if err != nil {
			fmt.Println(err)
		}
		if msg == pID+"die" {
			break
		}
		// Small pause, no pause resulted in lost messages.
		time.Sleep(time.Millisecond)
	}

	// Remove it from the active watchers slices.
	if len(con.config.watched) == 1 {
		con.config.watched = nil
	} else {
		con.config.watched[watchedN] = con.config.watched[len(con.config.watched)-1]
		con.config.watched = con.config.watched[:len(con.config.watched)-1]
	}

	// Close the socket.
	close(watch.channel)
	conn.Close()
	fmt.Println("PID [" + pID + "] killed.")
}

// watchGetLast retrieves X amount of messages from database to forward to watch client.
func (con *console) watchGetLast(watch WatchLog, amount int) error {
	// Prevent attempting bad number of messages.
	if amount <= 0 {
		return nil
	}

	var q = make(map[string]interface{})
	if watch.channelID != "" {
		q["channelid"] = watch.channelID
	} else {
		q = nil
	}

	var err error
	dbdat := DBdataCreate(watch.guildID, CollectionMessages, Message{}, q, nil)
	if err = dbdat.dbGetWithLimit(Message{}, []string{"-timestamp"}, amount); err != nil {
		return err
	}

	var msgs []Message
	var msg Message
	for _, m := range dbdat.Documents {
		msg = m.(Message)
		msgs = append(msgs, msg)
	}

	for i, j := 0, len(msgs)-1; i < j; i, j = i+1, j-1 {
		msgs[i], msgs[j] = msgs[j], msgs[i]
	}

	for _, m := range msgs {
		output := watch.MessageCreate(m.Author.Name, m.Author.Discriminator, m.ChannelName, m.Content)
		watch.Talk(output)
	}

	return nil
}

// MessageCreate converts a message to proper output for Talk.
func (watch WatchLog) MessageCreate(username, discriminator, channel, content string) string {
	// Compose the message to send to the channel then the socket.
	var output = "--> "
	if watch.channelAll && channel != "" {
		output += "[" + channel + "]"
		// Check that we are monitoring this channel if any...
	}

	// Eventually add support for ContentWithMoreMentionsReplaced()
	output += "[" + username + "#" + discriminator + "] " + content

	return output
}

// Talk sends a message over a channel.
func (watch WatchLog) Talk(msg string) {
	watch.channel <- msg
}

func consoleHelp() string {
	text := [...]string{"check", "watch", "reset", "kill", "help", "exit"}

	var retText string
	for n, w := range text {
		retText += fmt.Sprintf("%10s ", w)
		if (n+1)%4 == 0 && n != 0 {
			retText += "\n"
		}
	}
	return retText + "\n"
}
