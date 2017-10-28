package main

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"unicode"

	"github.com/bwmarrin/discordgo"
	"github.com/d0x1p2/godbot"
	"gopkg.in/mgo.v2"
)

// Error constants.
var (
	ErrMsgEnding = errors.New("reached ending message")
)

// Color constants for embeded messages.
const (
	ColorMaroon = 0x800000
	ColorGreen  = 0x3B8040
	ColorBlue   = 0x5B6991
	ColorBlack  = 0x000000
	ColorGray   = 0x343434
	ColorYellow = 0xFEEB65
)

func strToCommands(io string) (bool, []string) {
	var cmd bool
	var slice []string

	lastQuote := rune(0)
	f := func(c rune) bool {
		switch {
		case c == lastQuote:
			lastQuote = rune(0)
			return false
		case lastQuote != rune(0):
			return false
		case c == '"':
			lastQuote = c
			return false
		default:
			return unicode.IsSpace(c)
		}
	}

	var str = io
	if strings.HasPrefix(io, envCMDPrefix) {
		str = strings.TrimPrefix(io, envCMDPrefix)
		cmd = true
	}

	s := strings.FieldsFunc(str, f)
	for _, w := range s {
		if strings.HasPrefix(w, "\"") {
			w = strings.TrimPrefix(w, "\"")
		}
		if strings.HasSuffix(w, "\"") {
			w = strings.TrimSuffix(w, "\"")
		}
		slice = append(slice, w)
	}

	return cmd, slice
}

// stripWhiteSpace cleans up nasty strings full of whitespace!.
func stripWhiteSpace(text string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) {
			return -1
		}
		return r
	}, text)
}

func msgToIOdat(msg *discordgo.MessageCreate) *IOdat {
	var io IOdat
	u := msg.Author

	io.command, io.io = strToCommands(msg.Content)
	io.input = msg.Content
	io.user = &User{ID: u.ID, Username: u.Username, Discriminator: u.Discriminator, Bot: u.Bot}
	io.msg = msg

	return &io
}

func sliceToIOdat(b *godbot.Core, s []string) *IOdat {
	u := b.User
	var io IOdat
	io.user = &User{ID: u.ID, Username: u.Username, Discriminator: u.Discriminator, Bot: u.Bot}
	io.command, io.io = strToCommands(strings.Join(s, " "))

	return &io
}

func tsConvert(ts string) string {
	a := strings.FieldsFunc(fmt.Sprintf("%s", ts), tsSplit)
	return fmt.Sprintf("%s %s", a[0], a[1])
}

func tsSplit(r rune) bool {
	return r == 'T' || r == '.' || r == '+'
}

func idSplit(r rune) bool {
	return r == '<' || r == '@' || r == '>' || r == ':' || r == ' '
}

func usernameAdd(username, discriminator string) string {
	return fmt.Sprintf("%s#%s", username, discriminator)
}

func usernameSplit(username string) []string {
	return strings.Split(username, "#")
}

func (cfg *Config) ioHandler(io *IOdat) (err error) {
	if len(io.io) < 1 {
		// Not enough arguments to do anything.
		// Prevents accessing nil pointer.
		return nil
	}

	// Make sure the channel is allowed to have bot commmands.
	if io.io[0] != "channel" {
		ch := ChannelNew(io.msg.ChannelID, io.guild.Name)
		if !io.user.HasPermission(io.guild.ID, permModerator) && !ch.Check() {
			io.msgEmbed = embedCreator("Bot commands have been disabled here.", ColorGray)
			return nil
		}
	}

	// Check if an alias here
	alias := AliasNew(io.io[0], "", io.user)
	link, err := alias.Check()
	if err != nil {
		if err != mgo.ErrNotFound {
			return err
		}
		err = nil
	} else {
		io.io = aliasConv(io.io[0], link, io.input)
	}

	command := io.io[0]
	switch strings.ToLower(command) {
	case "help":
		io.output = globalHelp()
	case "roll":
		io.miscRoll()
	case "top10":
		io.miscTop10()
	case "gen":
		io.roomGen()
	case "sz":
		io.msgEmbed = embedCreator(msgSize(io.msg.Message), ColorYellow)
	case "invite":
		io.msgEmbed = embedCreator(botInvite(), ColorGreen)
	case "ally":
		err = cfg.CoreAlliance(io)
	case "user":
		err = io.CoreUser()
	case "alias":
		err = io.CoreAlias()
	case "histo":
		err = io.histograph(cfg.Core.Session)
	case "channel":
		err = io.ChannelCore()
	case "event", "events":
		err = io.CoreEvent()
	case "ticket", "tickets":
		err = io.CoreTickets()
	case "cmd", "command":
		err = io.CoreDatabase()
	case "script", "scripts":
		err = io.CoreLibrary()
	case "clear", "delete":
		err = io.messageClear(cfg.DSession)
	case "echo":
		io.output = strings.Join(io.io[1:], " ")
		return
	}
	return
}

func embedCreator(description string, color int) *discordgo.MessageEmbed {
	return &discordgo.MessageEmbed{
		Author:      &discordgo.MessageEmbedAuthor{},
		Color:       color,
		Description: description,
		Fields:      []*discordgo.MessageEmbedField{},
	}
}

// messageClear removes X number of messages from the current server.
// TAG: TODO - support for the 100 message deleter (quicker)
func (io *IOdat) messageClear(s *discordgo.Session) error {
	channelID := io.msg.ChannelID
	messageID := io.msg.ID

	// Need permModerator to remove.
	if ok := io.user.HasPermission(io.guild.ID, permModerator); !ok {
		return ErrBadPermissions
	}

	// Validate a good number is provided.
	if len(io.io) < 2 {
		return errors.New("Invalid number provided")
	}

	var err error
	var amount int
	if amount, err = strconv.Atoi(io.io[1]); err != nil {
		return err
	}

	// Delete the message that calls for deleting others.
	if err = s.ChannelMessageDelete(channelID, messageID); err != nil {
		fmt.Println(err)
		return err
	}

	var msgs []*discordgo.Message
	// While we have messages to delete, pull and remove.
	for amount > 0 {
		var toGet int
		if amount > 100 {
			toGet = 100
		} else {
			toGet = amount
		}
		if msgs, err = s.ChannelMessages(channelID, toGet, messageID, "", ""); err != nil {
			fmt.Println(err)
			return err
		}

		for _, m := range msgs {
			if err = s.ChannelMessageDelete(channelID, m.ID); err != nil {
				fmt.Println(err)
				return err
			}
		}
		amount -= toGet
	}

	//s.ChannelMessageSendEmbed(channelID, embedCreator("Messages removed.", ColorMaroon))

	return nil
}

// botInvite returns information needed to add the bot to your server.
func botInvite() string {
	var msg string
	msg += fmt.Sprintf(
		"Invite me to your server!\n"+
			"Click to Add-> %s\n\n"+
			"Bot Support Server -> %s\n",
		"https://discordapp.com/oauth2/authorize?client_id=290843164892463104&scope=bot&permissions=469855422",
		envBotGuild,
	)
	return msg
}

// globalHelp prints vairous helps.
func globalHelp() string {
	var msg = "*Most commands have a '--help' ability."
	for t, cmd := range cmds {
		msg += fmt.Sprintf("\n\n[ %s ]", t)
		for c, txt := range cmd {
			msg += fmt.Sprintf("\n\t%s\n\t\t%s", c, txt)
		}
	}
	return "```" + msg + "```"
}

// msgSize is a small function intended to gauge a rough size of what a discord message is.
func msgSize(m *discordgo.Message) string {
	var sz int
	// Author sizes
	usr := func(u *discordgo.User) {
		sz += len(u.ID)
		sz += len(u.Username)
		sz += len(u.Avatar)
		sz += len(u.Discriminator)
		sz++ // Verified Bool
		sz++ // Bot account
	}

	msgE := func(e *discordgo.MessageEmbed) {
		sz += len(e.URL)
		sz += len(e.Type)
		sz += len(e.Title)
		sz += len(e.Description)
		sz += len(e.Timestamp)
	}

	sz += len(m.ID)
	sz += len(m.ChannelID)
	sz += len(m.Content[4:]) // Reduce for the ',sz '
	sz += len(m.Timestamp)
	sz += len(m.EditedTimestamp)
	for _, mr := range m.MentionRoles {
		sz += len(mr)
	}

	sz++ //Tts
	sz++ // Mention everyone
	usr(m.Author)
	for _, u := range m.Mentions {
		usr(u)
	}
	for _, e := range m.Embeds {
		msgE(e)
	}

	return fmt.Sprintf("\nContent:\n%s\n\nSize of message: %d bytes\n", m.Content[4:], sz)
}
