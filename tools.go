package main

import (
	"bufio"
	"errors"
	"fmt"
	"html"
	"os"
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

func strToCommands(io, cmdPrefix string) (bool, []string) {
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
	if strings.HasPrefix(io, cmdPrefix) {
		str = strings.TrimPrefix(io, cmdPrefix)
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

func msgToIOdata(msg *discordgo.MessageCreate, cmdPrefix string) *IOdata {
	var dat IOdata
	u := msg.Author

	dat.command, dat.io = strToCommands(msg.Content, cmdPrefix)
	dat.cmdPrefix = cmdPrefix
	dat.input = msg.Content
	dat.user = &User{ID: u.ID, Username: u.Username, Discriminator: u.Discriminator, Bot: u.Bot}
	dat.msg = msg

	return &dat
}

func emojiIntToStr(decimal int) string {
	return html.UnescapeString("&#" + strconv.Itoa(decimal) + ";")
}

func sliceToIOdata(b *godbot.Core, s []string, cmdPrefix string) *IOdata {
	u := b.User
	var dat IOdata
	dat.user = &User{ID: u.ID, Username: u.Username, Discriminator: u.Discriminator, Bot: u.Bot}
	dat.command, dat.io = strToCommands(strings.Join(s, " "), cmdPrefix)

	return &dat
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

func (cfg *Config) ioHandler(dat *IOdata) error {
	var err error
	if len(dat.io) < 1 {
		// Not enough arguments to do anything.
		// Prevents accessing nil pointer.
		return nil
	}

	// Make sure the channel is allowed to have bot commmands.
	if !strings.Contains(dat.input, "admin channel") {
		ch := ChannelNew(dat.msg.ChannelID, dat.guild.Name)
		// TAG: TODO - Add support for having EITHER Administrator or Moderator.
		if (!dat.user.HasRoleType(dat.guildConfig, rolePermissionMod) && !dat.user.HasRoleType(dat.guildConfig, rolePermissionAdmin)) && !ch.Check() {
			dat.msgEmbed = embedCreator("Bot commands have been disabled here.", ColorGray)
			return nil
		}
	}

	// Check if an alias here
	alias := AliasNew(dat.io[0], "", dat.guild.ID, dat.user)
	link, err := alias.Check()
	if err != nil {
		if err != mgo.ErrNotFound {
			return err
		}
		err = nil
	} else {
		dat.io = aliasConv(dat, link)
	}

	command := dat.io[0]
	switch strings.ToLower(command) {
	case "help":
		dat.output = globalHelp()
	case "roll":
		dat.miscRoll()
	case "top10":
		dat.miscTop10()
	case "gen":
		dat.roomGen()
	case "sz":
		dat.msgEmbed = embedCreator(msgSize(dat.msg.Message), ColorYellow)
	case "invite":
		dat.msgEmbed = embedCreator(botInvite(), ColorGreen)
	case "ally":
		return cfg.CoreAlliance(dat)
	case "user":
		return dat.CoreUser()
	case "alias":
		return dat.CoreAlias()
	case "histo":
		return dat.histograph(cfg.Core.Session)
	case "event", "events":
		return dat.CoreEvent()
	case "ticket", "tickets":
		return dat.CoreTickets()
	case "cmd", "command":
		return dat.CoreDatabase()
	case "script", "scripts":
		return dat.CoreLibrary()
	case "clear", "delete":
		return dat.messageClear(cfg.Core.Session, "fast")
	case "clear-slow":
		return dat.messageClear(cfg.Core.Session, "slow")
	case "vote":
		return dat.CoreVote()
	case "admin":
		return cfg.CoreAdmin(dat)
	case "echo":
		dat.output = echoMsg(dat.io[1:])
		return nil
	}
	return nil
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
func (dat *IOdata) messageClear(s *discordgo.Session, method string) error {
	channelID := dat.msg.ChannelID
	messageID := dat.msg.ID

	// Need permModerator to remove.
	// TAG: TODO - check for greater then.
	if ok := dat.user.HasRoleType(dat.guildConfig, rolePermissionMod); !ok {
		return ErrBadPermissions
	}

	// Validate a good number is provided.
	if len(dat.io) < 2 {
		return errors.New("Invalid number provided")
	}

	var err error
	var amount int
	if amount, err = strconv.Atoi(dat.io[1]); err != nil {
		return err
	}

	// Delete the message that calls for deleting others.
	if err = s.ChannelMessageDelete(channelID, messageID); err != nil {
		fmt.Println(err)
		return err
	}

	if strings.ToLower(method) == "slow" {
		return dat.messageClearSlow(s, channelID, messageID, amount)
	}

	return dat.messageClearFast(s, channelID, messageID, amount)
}

// messageClearFast uses the bulk deletion for faster clearing. Has a limitation of 100
// messages at a time and they can't be older than 2 weeks.
func (dat *IOdata) messageClearFast(s *discordgo.Session, channelID, messageID string, amount int) error {
	var err error
	var processed int
	// While we have messages to delete, pull and remove.
	for amount > 0 {
		var toGet int
		if amount > 100 {
			toGet = 100
		} else {
			toGet = amount
		}

		var msgs []*discordgo.Message
		if msgs, err = s.ChannelMessages(channelID, toGet, messageID, "", ""); err != nil {
			fmt.Println(err)
			return err
		}

		var mIDs []string
		for _, m := range msgs {
			mIDs = append(mIDs, m.ID)
		}

		if err := s.ChannelMessagesBulkDelete(channelID, mIDs); err != nil {
			dat.output = fmt.Sprintf("Deleted **%d** messages.\n"+
				"`%sclear-slow` command can clear additional messages.", processed, dat.cmdPrefix)
			return err
		}

		processed += toGet
		amount -= toGet
	}

	dat.output = fmt.Sprintf("Deleted **%d** messages.\n", processed)
	return nil
}

// messageClearSlow removes X number of messages from the current server.
// This is the slow method that processes 1 message at a time and circumvents the
// 2week old and 100 message at a time rules.
func (dat *IOdata) messageClearSlow(s *discordgo.Session, channelID, messageID string, amount int) error {
	var err error
	var processed int
	// While we have messages to delete, pull and remove.
	for amount > 0 {
		var toGet int
		if amount > 100 {
			toGet = 100
		} else {
			toGet = amount
		}

		var msgs []*discordgo.Message
		if msgs, err = s.ChannelMessages(channelID, toGet, messageID, "", ""); err != nil {
			fmt.Println(err)
			return err
		}

		for _, m := range msgs {
			if err = s.ChannelMessageDelete(channelID, m.ID); err != nil {
				fmt.Println(err)
				dat.output = fmt.Sprintf("Deleted **%d** messages.\n", processed)
				return err
			}
		}

		processed++
		amount -= toGet
	}

	dat.output = fmt.Sprintf("Deleted **%d** messages.\n", processed)
	return nil
}

// botInvite returns information needed to add the bot to your server.
func botInvite() string {
	var msg string
	msg += fmt.Sprintf(
		"Invite me to your server!\n"+
			"Click to Add-> %s\n\n"+
			"Bot Support Server -> %s\n",
		"https://discordapp.com/oauth2/authorize?client_id=375083817565945867&scope=bot&permissions=469855422",
		envBotGuild,
	)
	return msg
}

// globalHelp prints vairous helps.
func globalHelp() string {
	var msg = "*Most commands have a '--help' or 'help' ability if typed after base command."
	for t, cmd := range cmds {
		msg += fmt.Sprintf("\n\n[ %s ]", t)
		for c, txt := range cmd {
			msg += fmt.Sprintf("\n\t%s\n\t\t%s", c, txt)
		}
	}

	var msg2 string
	msg2 += "\n\nThe easy-to-use Documentation can be found at: "
	return "```" + msg + "```" + msg2 + helpDocs
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

// printDebug creates a debug message to be displayed.
func printDebug(msg string) {
	if DEBUG {
		fmt.Println("DEBUG: " + msg)
	}
}

// newPause just waits for the enter key to be pressed.
func newPause() {
	buf := bufio.NewReader(os.Stdin)
	fmt.Print("Press [enter] to continue... ")
	_, err := buf.ReadBytes('\n')
	if err != nil {
		fmt.Println(err)
	}
}
