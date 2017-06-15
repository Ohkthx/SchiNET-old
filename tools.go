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

func strToCommands(io string) (bool, []string) {

	var slice []string
	var help bool
	var quoted bool
	var quote string

	s := strings.Fields(io)

	for _, w := range s {
		if strings.HasPrefix(w, envCMDPrefix) {
			w = strings.TrimPrefix(w, envCMDPrefix)
		} else if strings.HasPrefix(w, "\"") && quoted == false {
			w = strings.TrimPrefix(w, "\"")
			quote += w + " "
			quoted = true
		} else if strings.HasSuffix(w, "\"") && quoted {
			w = strings.TrimSuffix(w, "\"")
			quote += w
			w = quote
			quoted = false

		} else if quoted {
			quote += w + " "
		}

		if strings.ToLower(w) == "help" {
			help = true
		} else if quoted == false {
			slice = append(slice, w)
		}
	}

	return help, slice
}

func msgToIOdat(msg *discordgo.MessageCreate) *IOdat {
	var io IOdat
	u := msg.Author

	io.help, io.io = strToCommands(msg.Content)
	io.input = msg.Content
	io.user = &User{ID: u.ID, Username: u.Username, Discriminator: u.Discriminator, Bot: u.Bot}
	io.msg = msg

	return &io
}

func sliceToIOdat(b *godbot.Core, s []string) *IOdat {
	u := b.User
	var io IOdat
	io.user = &User{ID: u.ID, Username: u.Username, Discriminator: u.Discriminator, Bot: u.Bot}
	io.help, io.io = strToCommands(strings.Join(s, " "))

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
	return r == '<' || r == '@' || r == '>'
}

func (io *IOdat) ioHandler() (err error) {
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

	for {
		m, err := s.ChannelMessages(cID, 100, mID, "", "")
		if err != nil {
			return nil, err
		}
		var msgAmt = len(m)
		if msgAmt == 0 {
			return nil, ErrMsgEnding
		}
		for n, m := range m {
			msgTotal++
			// Update user here.
			err = userUpdate(m.ChannelID, m.Author, 1)
			if err != nil {
				fmt.Println("updating users credits", err)
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
			fmt.Printf("\r%7d messages processed.", msgTotal)
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
			err = userUpdate(m.ChannelID, m.Author, 1)
			if err != nil {
				fmt.Println("updating users credits", err)
			}
			// Break at Ending Message.
			if msgAmt < 100 && msgAmt == n+1 {
				return dbm, ErrMsgEnding
			}

			mID = m.ID
			fmt.Printf("\r%7d messages processed.", msgTotal)
		}
	}
}

func messagesProcessStartup() error {

	for n, c := range Bot.Channels {
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

		fmt.Printf("\n[%d] Processing [%s]", n, c.ID)

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
		fmt.Printf("\r[%d] Processed  [%s]", n, c.ID)

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

func userUpdate(cID string, us *discordgo.User, spoils int) error {
	var q = make(map[string]interface{})
	var c = make(map[string]interface{})

	gID, err := Bot.GetGuildID(cID)
	if err != nil {
		return err
	}
	g := Bot.GetGuild(gID)

	var totalInc = 1
	if spoils != 1 {
		totalInc = 0
	}

	var u = User{
		ID:            us.ID,
		Username:      us.Username,
		Discriminator: us.Discriminator,
		Bot:           us.Bot,
		CreditsTotal:  1,
		Credits:       1,
	}

	q["id"] = u.ID
	c["$inc"] = bson.M{"creditstotal": totalInc, "credits": spoils}
	var dbdat = DBdatCreate(g.Name, CollectionUsers, u, q, c)

	err = dbdat.dbEdit(User{})
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

func userGet(cID, uID string) (*User, error) {
	var q = make(map[string]interface{})

	gID, err := Bot.GetGuildID(cID)
	if err != nil {
		return nil, err
	}
	g := Bot.GetGuild(gID)

	q["id"] = uID
	var dbdat = DBdatCreate(g.Name, CollectionUsers, User{}, q, nil)
	err = dbdat.dbGet(User{})
	if err != nil {
		return nil, err
	}

	var u User
	u = dbdat.Document.(User)

	return &u, nil
}

func userEmbedCreate(u *User) *discordgo.MessageEmbed {
	description := fmt.Sprintf("```C\n"+
		"%-13s: %-18s %7d %s\n"+
		"%-13s: %-18s %7d %s```",
		"Username", fmt.Sprintf("%s#%s", u.Username, u.Discriminator), u.Credits, GambleCredits,
		"ID", u.ID, u.CreditsTotal, "rep")

	return &discordgo.MessageEmbed{
		Author:      &discordgo.MessageEmbedAuthor{},
		Color:       ColorBlue,
		Description: description,
		Fields:      []*discordgo.MessageEmbedField{},
	}
}
