package main

import (
	"errors"
	"fmt"
	"strings"

	mgo "gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"

	"github.com/bwmarrin/discordgo"
	"github.com/d0x1p2/godbot"
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

func strToCommands(io string) ([2]bool, []string) {

	var slice []string
	var quoted bool
	var quote string
	var res [2]bool

	s := strings.Fields(io)

	for _, w := range s {
		if strings.HasPrefix(w, envCMDPrefix) {
			w = strings.TrimPrefix(w, envCMDPrefix)
			res[0] = true
		} else if strings.HasPrefix(w, "\"") && strings.HasSuffix(w, "\"") {
			w = strings.TrimPrefix(w, "\"")
			w = strings.TrimSuffix(w, "\"")
		} else if strings.HasPrefix(w, "\"") && quoted == false {
			w = strings.TrimPrefix(w, "\"")
			quote += w + " "
			quoted = true
		} else if strings.HasSuffix(w, "\"") && quoted {
			w = strings.TrimSuffix(w, "\"")
			quote += w
			w = quote
			quoted = false
			quote = ""
		} else if quoted {
			quote += w + " "
		}

		if strings.ToLower(w) == "help" {
			res[1] = true
		} else if quoted == false {
			slice = append(slice, w)
		}
	}

	return res, slice
}

func msgToIOdat(msg *discordgo.MessageCreate) *IOdat {
	var io IOdat
	u := msg.Author
	var b [2]bool

	b, io.io = strToCommands(msg.Content)
	io.input = msg.Content
	io.user = &User{ID: u.ID, Username: u.Username, Discriminator: u.Discriminator, Bot: u.Bot}
	io.msg = msg
	io.command = b[0]
	io.help = b[1]

	return &io
}

func sliceToIOdat(b *godbot.Core, s []string) *IOdat {
	u := b.User
	var io IOdat
	var bol [2]bool
	io.user = &User{ID: u.ID, Username: u.Username, Discriminator: u.Discriminator, Bot: u.Bot}
	bol, io.io = strToCommands(strings.Join(s, " "))
	io.command = bol[0]
	io.help = bol[1]

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

func (io *IOdat) ioHandler() (err error) {
	var s string
	if len(io.io) < 1 {
		// Not enough arguments to do anything.
		// Prevents accessing nil pointer.
		return nil
	}

	command := io.io[0]
	switch strings.ToLower(command) {
	case "roll":
		io.miscRoll()
	case "top10":
		io.miscTop10()
	case "gamble":
		err = io.creditsGamble()
	case "credits", "user", "xfer":
		err = io.creditsPrint()
	case "event", "events":
		fallthrough
	case "add", "del", "edit":
		err = io.dbCore()
	case "script", "scripts":
		s, err = scriptCore(io.guild.Name, io.msg.Author, io.io, io.help)
		io.msgEmbed = embedCreator(s, ColorGreen)
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

// Archives messages from most RECENT to OLDEST. Returns last processed.
func messagesToPast(cID, mID, emID string) (*DBMsg, error) {
	// cID  - Channel ID to search.
	// mID  - Message ID to START from.
	// emID - Ending Message ID to STOP at.
	s := Bot.Session
	var msgTotal int
	var dbm = &DBMsg{ID: cID}

	channel := Bot.GetChannel(cID)
	guild := Bot.GetGuild(channel.GuildID)

	cl, err := Bot.ChannelLockCreate(cID)
	if err == nil {
		err = cl.ChannelLock(false)
		if err != nil {
			return nil, err
		}
		defer cl.ChannelUnlock()
	}

	for {
		msgs, err := s.ChannelMessages(cID, 100, mID, "", "")
		if err != nil {
			return nil, err
		}
		var msgAmt = len(msgs)
		if msgAmt == 0 {
			return nil, ErrMsgEnding
		}
		for n, m := range msgs {
			msgTotal++
			// Update user here.
			ts, _ := m.Timestamp.Parse()
			err = UserUpdateSimple(guild.Name, m.Author, 1, ts)
			if err != nil {
				fmt.Println(err)
			}

			if dbm.MIDr == "" {
				dbm.MIDr = m.ID
				dbm.Content = m.Content
			}
			// Break at Ending Message.
			if m.ID == emID && emID != "" {
				dbm.MTotal = msgTotal
				dbm.MIDf = m.ID
				return dbm, ErrMsgEnding
			} else if msgAmt < 100 && msgAmt == n+1 {
				dbm.MTotal = msgTotal
				dbm.MIDf = m.ID
				return dbm, ErrMsgEnding
			} else {
				mID = m.ID
			}
			//fmt.Printf("\r%s -> %7d messages processed.", cID, msgTotal)
		}
	}
}

// Gets messages from OLDEST to most RECENT. Returns last processed.
func messagesToPresent(dbm *DBMsg) (*DBMsg, error) {
	// cID  - Channel ID to search.
	// mID  - Message to START from.
	// emID - Message to STOP at.
	s := Bot.Session
	var msgTotal int
	var mID = dbm.MIDr

	channel := Bot.GetChannel(dbm.ID)
	guild := Bot.GetGuild(channel.GuildID)

	cl, err := Bot.ChannelLockCreate(dbm.ID)
	if err == nil {
		err = cl.ChannelLock(false)
		if err != nil {
			return nil, err
		}
		defer cl.ChannelUnlock()
	}

	for {
		// before, after, around
		m, err := s.ChannelMessages(dbm.ID, 100, "", mID, "")
		if err != nil {
			return nil, err
		}
		var msgAmt = len(m)
		if msgAmt == 0 {
			// No messages, nil will not update.
			return dbm, ErrMsgEnding
		}
		for n, m := range m {
			if msgTotal == 0 {
				dbm.Content = m.Content
				dbm.MIDr = m.ID
			}
			dbm.MTotal++
			msgTotal++
			// Update users credits.
			ts, _ := m.Timestamp.Parse()
			err = UserUpdateSimple(guild.Name, m.Author, 1, ts)
			if err != nil {
				fmt.Println(err)
			}
			// Break at Ending Message.
			if msgAmt < 100 && msgAmt == n+1 {
				return dbm, ErrMsgEnding
			}

			mID = m.ID
			//fmt.Printf("\r%7d messages processed.", msgTotal)
		}
	}
}

func messagesProcessStartup() error {

	for _, c := range Bot.Channels {
		if c.Type != "text" {
			continue
		}
		var lastID string
		var dbm *DBMsg

		dbmsg, err := messagesGet(c.ID)
		if err != nil {
			if err != mgo.ErrNotFound {
				return err
			}
			lastID = ""
		} else if dbmsg != nil {
			lastID = dbmsg.MIDr
		}

		//fmt.Printf("\n[%d] Processing [%s]", n, c.ID)

		if lastID == "" {
			dbm, err = messagesToPast(c.ID, lastID, "")
			if err != nil {
				if err != ErrMsgEnding {
					//return err
					continue
				}
			}
		} else {
			dbm, err = messagesToPresent(dbmsg)
			if err != nil {
				if err != ErrMsgEnding {
					//return err
					continue
				}
			}
		}
		//fmt.Printf("\r[%d] Processed  [%s]", n, c.ID)

		// Update Database with last processed message.
		if dbm != nil {
			err = messagesUpdate(dbm)
			if err != nil {
				return err
			}
		}
	}
	fmt.Println()

	return nil

}

func messagesUpdate(dbm *DBMsg) error {
	var q = make(map[string]interface{})
	var c = make(map[string]interface{})

	gID, err := Bot.GetGuildID(dbm.ID)
	if err != nil {
		return err
	}
	g := Bot.GetGuild(gID)

	q["id"] = dbm.ID
	c["$set"] = bson.M{"id": dbm.ID, "mtotal": dbm.MTotal, "midr": dbm.MIDr, "content": dbm.Content}
	var dbdat = DBdatCreate(g.Name, CollectionConfigs, dbm, q, c)

	err = dbdat.dbEdit(DBMsg{})
	if err != nil {
		if err != mgo.ErrNotFound {
			return err
		}
		// Add to DB since it doesn't exist.
		err = dbdat.dbInsert()
		if err != nil {
			return err
		}
	}

	return nil
}

func messagesGet(cID string) (*DBMsg, error) {
	var q = make(map[string]interface{})

	gID, err := Bot.GetGuildID(cID)
	if err != nil {
		return nil, err
	}
	g := Bot.GetGuild(gID)

	q["id"] = cID
	var dbdat = DBdatCreate(g.Name, CollectionConfigs, DBMsg{}, q, nil)
	err = dbdat.dbGet(DBMsg{})
	if err != nil {
		return nil, err
	}

	var u DBMsg
	u = dbdat.Document.(DBMsg)

	return &u, nil
}
