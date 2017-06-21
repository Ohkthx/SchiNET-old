package main

import (
	"errors"
	"flag"
	"fmt"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"

	"github.com/d0x1p2/godbot"
	mgo "gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

// Error constants.
var (
	ErrBadUser        = errors.New("bad user supplied")
	ErrBadPermissions = errors.New("you do not have permissions to do that")
	ErrBanChanExists  = errors.New("user already has a ban for that channel")
	ErrBanNotFound    = errors.New("ban not found to clear")
)

// Permission scheme constants.
const (
	permAdmin = 1 << iota
	permModerator
	permAscended
	permNormal
)

const permAll = permAdmin | permModerator | permAscended | permNormal

// UserNew creates a new user instance based on accessing user.
func UserNew(u *discordgo.User) *User {
	if u == nil {
		u = &discordgo.User{
			ID:            "",
			Username:      "",
			Discriminator: "",
			Bot:           false,
		}
	}
	var user = User{
		ID:            u.ID,
		Username:      u.Username,
		Discriminator: u.Discriminator,
		Bot:           u.Bot,
		CreditsTotal:  1,
		Credits:       1,
		Access:        permNormal,
	}
	return &user
}

// UserUpdateSimple stream-lines the process for incrementing credits.
func UserUpdateSimple(database string, user *discordgo.User, inc int, ts time.Time) error {
	u := UserNew(user)

	if err := u.Get(database, u.ID); err != nil {
		if err == mgo.ErrNotFound {
			if err := u.Update(database); err != nil {
				return err
			}
			return nil
		}
		return err
	}

	if inc == 1 {
		u.CreditsTotal++
		u.Credits++
		u.LastSeen = ts
	} else {
		u.Credits += inc
	}

	if err := u.Update(database); err != nil {
		return err
	}

	return nil

}

// HasPermission returns True if have permissions for action, false if not.
func (u *User) HasPermission(access int) bool { return u.Access&access == access }

// PermissionAdd upgrades a user to new permissions
func (u *User) PermissionAdd(access int) { u.Access &= access }

// PermissionDelete strips a permission from an User
func (u *User) PermissionDelete(access int) { u.Access ^= access }

// Update pushes an update to the database.
func (u *User) Update(database string) error {
	var err error
	var q = make(map[string]interface{})
	var c = make(map[string]interface{})

	q["id"] = u.ID
	c["$set"] = bson.M{
		"creditstotal": u.CreditsTotal,
		"credits":      u.Credits,
		"access":       u.Access,
		"lastseen":     u.LastSeen}

	var dbdat = DBdatCreate(database, CollectionUsers, u, q, c)
	err = dbdat.dbEdit(User{})
	if err != nil {
		if err == mgo.ErrNotFound {
			// Add to DB since it doesn't exist.
			if err := dbdat.dbInsert(); err != nil {
				return err
			}
			return nil
		}
		return err
	}

	return nil
}

// Get a user from a database
func (u *User) Get(database, uID string) error {
	var q = make(map[string]interface{})

	q["id"] = uID

	var dbdat = DBdatCreate(database, CollectionUsers, User{}, q, nil)
	err := dbdat.dbGet(User{})
	if err != nil {
		return err
	}

	var user = User{}
	user = dbdat.Document.(User)
	u.ID = user.ID
	u.Username = user.Username
	u.Discriminator = user.Discriminator
	u.Bot = user.Bot
	u.Credits = user.Credits
	u.CreditsTotal = user.CreditsTotal
	u.Access = user.Access
	u.LastSeen = user.LastSeen

	return nil
}

// EmbedCreate returns an embed object with User information.
func (u *User) EmbedCreate() *discordgo.MessageEmbed {
	tn := time.Now()
	dur := tn.Sub(u.LastSeen)

	ta := fmt.Sprintf("**%d** hours, **%d** minutes.", int(dur.Hours()), int(dur.Minutes())%60)

	description := fmt.Sprintf(
		"__ID: %s__\n"+
			"**Username**:   %-18s **%s**: %-10d\n"+
			"**Reputation**: %d\n\n"+
			"**Last Seen**: %s",
		u.ID,
		fmt.Sprintf("%s#%s", u.Username, u.Discriminator), strings.Title(GambleCredits), u.Credits,
		u.CreditsTotal,
		ta)

	return &discordgo.MessageEmbed{
		Author:      &discordgo.MessageEmbedAuthor{},
		Color:       ColorBlue,
		Description: description,
		Fields:      []*discordgo.MessageEmbedField{},
	}
}

// Ban will remove a player from the server
func (u *User) Ban(database, cID string, io []string) (string, error) {
	var msg, idp, id, comment string        // Message to send and IDs
	var soft, hard, remove, list, help bool // Types of ban.
	var err error

	fl := flag.NewFlagSet("ban", flag.ContinueOnError)
	fl.StringVar(&idp, "user", "", "Username")
	fl.StringVar(&comment, "c", "", "Ban Comment")
	fl.BoolVar(&soft, "soft", false, "Type: Soft Ban")
	fl.BoolVar(&hard, "hard", false, "Type: Hard Ban")
	fl.BoolVar(&help, "help", false, "This message")
	fl.BoolVar(&remove, "remove", false, "Remove a type of ban")
	fl.BoolVar(&list, "list", false, "List all bans.")
	if err = fl.Parse(io[1:]); err != nil {
		return "", err
	}

	if idp != "" {
		ids := strings.FieldsFunc(idp, idSplit)
		id = ids[0]
	}

	if err = u.Get(database, u.ID); err != nil {
		return "", err
	}

	if ok := u.HasPermission(permAdmin); !ok {
		return "", ErrBadPermissions
	} else if help {
		prefix := "**Need** __type__ and __username__.\n\n"
		return Help(fl, prefix), nil
	} else if list {
		return banList(database)
	}

	if (soft || hard) && (id != "") {
		// Find user.
		criminal := UserNew(nil)
		if err = criminal.Get(database, id); err != nil {
			return "", err
		}

		// Create a new ban.
		var b = criminal.BanNew()
		b.ByLast = &UserBasic{ID: u.ID, Name: u.Username, Discriminator: u.Discriminator}

		if soft {
			if msg, err = b.banSoft(database, cID, comment, remove); err != nil {
				return "", err
			}
		} else if hard {
			if msg, err = b.banHard(database, cID, comment, remove); err != nil {
				return "", err
			}
		}
	} else {
		prefix := "**Need** __type__ and __username__.\n\n"
		return Help(fl, prefix), nil
	}

	return msg, nil

}

func (b *Ban) banSoft(database, cID, comment string, remove bool) (string, error) {
	var msg string
	if err := b.Get(database); err != nil {
		if err == mgo.ErrNotFound {
			if err := b.Update(database); err != nil {
				return "", err
			}
		} else {
			return "", err
		}
	}

	if !remove {
		for _, c := range b.Channels {
			if c.ChannelID == cID {
				return "", ErrBanChanExists
			}
		}

		c := Bot.GetChannel(cID)

		cNew := chanBan{
			Name:      c.Name,
			ChannelID: c.ID,
			Comment:   comment,
			By:        b.ByLast,
			Date:      time.Now(),
		}

		b.Amount++
		b.Channels = append(b.Channels, cNew)
		b.Last = time.Now()
		b.ByLast = nil

		// Attempt permission changes here.
		err := Bot.Session.ChannelPermissionSet(cID, b.User.ID, "member", 0, 2048)
		if err != nil {
			return "", err
		}

		if err := b.Update(database); err != nil {
			return "", err
		}

		msg = fmt.Sprintf("<@%s> has **soft banned** <@%s> from the <#%s> channel.\n",
			cNew.By.ID, b.User.ID, cID)

	} else {
		var found bool
		var adminID string
		for n, c := range b.Channels {
			if c.ChannelID == cID {
				found = true
				copy(b.Channels[n:], b.Channels[n+1:])
				b.Channels[len(b.Channels)-1] = chanBan{}
				b.Channels = b.Channels[:len(b.Channels)-1]
				adminID = c.By.ID
			}
		}
		if !found {
			return "", ErrBanNotFound
		}
		if err := Bot.Session.ChannelPermissionDelete(cID, b.User.ID); err != nil {
			return "", err
		}

		if err := b.Update(database); err != nil {
			return "", err
		}

		msg = fmt.Sprintf("<@%s> has **removed** <@%s>'s soft ban for the <#%s> channel.\n",
			adminID, b.User.ID, cID)
	}

	return msg, nil
}

func (b *Ban) banHard(database, cID, comment string, remove bool) (string, error) {
	var msg string
	if err := b.Get(database); err != nil {
		if err == ErrScriptNotFound {
			if err := b.Update(database); err != nil {
				return "", err
			}
		}
		return "", err
	}

	gID, err := Bot.GetGuildID(cID)
	if err == godbot.ErrNotFound {
		return "", errors.New("Seems something happened getting Guild ID")
	}

	if !remove {
		if err := Bot.Session.GuildBanCreateWithReason(gID, b.User.ID, comment, 1); err != nil {
			return "", err
		}
		msg = fmt.Sprintf("<@%s> has **banned** %s from the server.\n Sucks to suck.",
			b.ByLast.ID, b.User.Name)
		return msg, nil
	}

	if err := Bot.Session.GuildBanDelete(gID, b.User.ID); err != nil {
		return "", err
	}
	msg = fmt.Sprintf("<@%s> has **banned** %s from the server.\n Sucks to suck.",
		b.ByLast.ID, b.User.Name)

	return msg, nil
}

// BanNew creates a new instance of a Ban.
func (u *User) BanNew() *Ban {
	var ub = &UserBasic{
		ID:            u.ID,
		Name:          u.Username,
		Discriminator: u.Discriminator,
	}
	return &Ban{
		User:   ub,
		ByLast: nil,
		Amount: 0,
	}
}

// Get a ban object from Database.
func (b *Ban) Get(database string) error {
	var q = make(map[string]interface{})
	q["user.id"] = b.User.ID

	dbdat := DBdatCreate(database, CollectionBlacklist, Ban{}, q, nil)
	err := dbdat.dbGet(Ban{})
	if err != nil {
		if err == mgo.ErrNotFound {
			return err
		}
		return err
	}

	var ban Ban
	ban = dbdat.Document.(Ban)
	if len(ban.Channels) > 0 {
		for _, c := range ban.Channels {
			b.Channels = append(b.Channels, c)
		}
	}

	b.User = ban.User
	b.Amount = ban.Amount
	b.Last = ban.Last

	return nil
}

// Update database with new information.
func (b *Ban) Update(database string) error {
	var q = make(map[string]interface{})
	var c = make(map[string]interface{})
	q["user.id"] = b.User.ID
	c["$set"] = bson.M{
		"channels": b.Channels,
		"amount":   b.Amount,
		"bylast":   nil,
		"last":     b.Last,
	}

	dbdat := DBdatCreate(database, CollectionBlacklist, b, q, c)
	err := dbdat.dbEdit(Ban{})
	if err != nil {
		if err == mgo.ErrNotFound {
			if err := dbdat.dbInsert(); err != nil {
				return err
			}
			return nil
		}
		return err
	}

	return nil
}

func banList(database string) (string, error) {
	var msg string

	db := DBdatCreate(database, CollectionBlacklist, Ban{}, nil, nil)
	if err := db.dbGetAll(Ban{}); err != nil {
		return "", err
	}

	msg = "List of current Soft Bans:\n\n"
	// Convert here.
	if len(db.Documents) == 0 {
		return "List is empty.", nil
	}

	var b Ban
	var found bool
	for _, criminal := range db.Documents {
		b = criminal.(Ban)
		if len(b.Channels) > 0 {
			msg += fmt.Sprintf("\t**__%s#%s__**\n", b.User.Name, b.User.Discriminator)
			for _, c := range b.Channels {
				found = true
				msg += fmt.Sprintf("\t\tChannel: **%s**, Comment: **%s**\n", c.Name, c.Comment)
			}
		}
	}

	if !found {
		return "List is empty.", nil
	}
	return msg, nil
}
