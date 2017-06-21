package main

import (
	"errors"
	"flag"
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/d0x1p2/go_pastebin"
)

// Error constants for misc functions.
var (
	ErrGambleBadAmount = errors.New("bad amount of credits for action")
	ErrGambleNotMin    = func(a int) error { return fmt.Errorf("did not provide enough to reach the minimum of %d", a) }
	ErrGambleNotEnough = errors.New("not enough credits")
)

// Constants for Gambling.
const (
	GambleCredits = "gold"
	GambleMin     = 100
)

func (io *IOdat) miscRoll() {
	var roll1, roll2 int
	s := rand.NewSource(time.Now().UnixNano())
	r := rand.New(s)
	roll1 = r.Intn(6) + 1
	roll2 = r.Intn(6) + 1

	msg := fmt.Sprintf("*%s rolls %d, %d*", io.user.Username, roll1, roll2)
	io.msgEmbed = embedCreator(msg, ColorBlue)

	io.rm = true
	io.output = msg
	return
}

func (io *IOdat) miscTop10() {
	var roll int
	s := rand.NewSource(time.Now().UnixNano())
	r := rand.New(s)

	roll = r.Intn(100)
	if roll <= 25 {
		io.output = fmt.Sprintf("**%s** is top 10!", io.user.Username)
		return
	}

	io.output = fmt.Sprintf("**%s** is **NOT** top 10.", io.user.Username)

	return
}

func (io *IOdat) creditsGamble() error {
	if len(io.io) < 2 {
		return ErrBadArgs
	}

	var all bool
	var twealth, toGamble, spoils int //, mod int
	var err error

	if strings.ToLower(io.io[1]) == "all" {
		all = true
	} else {
		toGamble, err = strconv.Atoi(io.io[1])
		if err != nil {
			return ErrBadArgs
		}
		if toGamble < GambleMin {
			return ErrGambleNotMin(GambleMin)
		}
	}
	// Get user's credits from Database
	u := UserNew(io.msg.Author)
	if err = u.Get(io.guild.Name, io.user.ID); err != nil {
		return err
	}

	if all {
		if u.Credits < (GambleMin / 2) {
			return ErrGambleNotMin(GambleMin / 2)
		}
		toGamble = u.Credits
		twealth = 0
		spoils = creditsGambleResult(57, 41, 2, toGamble)
	} else {
		if toGamble > u.Credits {
			return ErrGambleNotEnough
		}
		twealth = u.Credits - toGamble
		spoils = creditsGambleResult(59, 39, 2, toGamble)
	}

	var msg string
	if spoils <= 0 {
		msg = fmt.Sprintf("<@%s> gambled **%d** %s\n"+
			"Result: **loss**\n"+
			"%s remaining in bank: **%d**.",
			io.user.ID, toGamble, GambleCredits, strings.Title(GambleCredits), twealth)
		bu := UserNew(Bot.User)
		bu.Get(io.guild.Name, bu.ID)
		bu.Credits += toGamble
		err = bu.Update(io.guild.Name)
		if err != nil {
			return err
		}
	} else {
		if all {
			twealth = spoils
		} else {
			twealth += spoils
		}
		msg = fmt.Sprintf("<@%s> gambled **%d** %s\n"+
			"Result: **Won**    spoils: **%d**\n"+
			"%s remaining in bank: **%d**.",
			io.user.ID, toGamble, GambleCredits, spoils, strings.Title(GambleCredits), twealth)
	}

	// twealth has new player bank amount.
	// Need to get difference and increment.
	u.Credits = twealth

	err = u.Update(io.guild.Name)
	if err != nil {
		return err
	}

	io.msgEmbed = embedCreator(msg, ColorYellow)

	return nil
}

func creditsGambleResult(l, d, t, credits int) int {
	s1 := rand.NewSource(time.Now().UnixNano())
	r1 := rand.New(s1)

	total := l + d + t
	if total != 100 {
		l = 60
		d = 38
		t = 2
	}

	num := r1.Intn(100)
	if num < l {
		// Lose all
		credits = 0
	} else if num >= l && num < l+d {
		// Win x2
		credits *= 2
	} else if num >= l+d {
		// Win x3
		credits *= 3
	}

	return credits
}

func (io *IOdat) creditsTransfer() error {
	database := io.guild.Name
	amt, err := strconv.Atoi(io.io[2])
	if err != nil {
		return ErrGambleBadAmount
	}

	s := strings.FieldsFunc(io.io[1], idSplit)
	u2ID := s[0]

	if amt < 0 {
		msg := "The authorities have been alerted with your attempt of theft!"
		io.msgEmbed = embedCreator(msg, ColorMaroon)
		return nil
	} else if amt == 0 || u2ID == io.user.ID {
		msg := "What do you plan on accomplishing with this?"
		io.msgEmbed = embedCreator(msg, ColorMaroon)
		return nil
	}

	u1 := UserNew(io.msg.Author)
	if err := u1.Get(database, io.user.ID); err != nil {
		return err
	}

	u2 := UserNew(io.msg.Author)
	if err := u2.Get(database, u2ID); err != nil {
		return err
	}

	if u1.Credits >= amt {
		u1.Credits -= amt
		if err := u1.Update(database); err != nil {
			return err
		}

		u2.Credits += amt
		if err := u2.Update(database); err != nil {
			return err
		}

		io.msgEmbed = embedCreator(
			fmt.Sprintf("**%s** has transferred __**%d**__ %s to **%s**.",
				u1.Username, amt, GambleCredits, u2.Username),
			ColorGreen,
		)

		return nil
	}
	io.msgEmbed = embedCreator(
		fmt.Sprintf("You can't afford %d %s, you only have %d.",
			amt, GambleCredits, u1.Credits),
		ColorMaroon,
	)

	return nil
}

func (io *IOdat) creditsPrint() error {
	var msg string
	var err error
	if len(io.io) < 2 {
		return ErrBadArgs
	} else if len(io.io) == 3 && strings.ToLower(io.io[0]) == "xfer" {
		err = io.creditsTransfer()
		if err != nil {
			return err
		}
		return nil
	} else if strings.Contains(io.io[1], "reset") {
		if msg, err = creditsReset(io.guild.Name, io.msg.Author); err != nil {
			return err
		}
		io.msgEmbed = embedCreator(msg, ColorGreen)
		return nil
	}

	s := strings.FieldsFunc(io.io[1], idSplit)
	id := s[0]

	u := UserNew(io.msg.Author)
	if err := u.Get(io.guild.Name, id); err != nil {
		return err
	}

	io.msgEmbed = u.EmbedCreate()
	return nil

}

func creditsReset(database string, dgu *discordgo.User) (string, error) {
	var msg = "Users reset:\n\n"
	uID := dgu.ID
	user := UserNew(dgu)
	if err := user.Get(database, uID); err != nil {
		return "", err
	}

	if ok := user.HasPermission(permAll); !ok {
		return "", ErrBadPermissions
	}

	db := DBdatCreate(database, CollectionUsers, User{}, nil, nil)
	if err := db.dbGetAll(User{}); err != nil {
		return "", err
	}

	if len(db.Documents) == 0 {
		return "", nil
	}

	var found bool
	var doc User
	for _, u := range db.Documents {
		doc = u.(User)
		if doc.Credits != doc.CreditsTotal {
			found = true
			msg += fmt.Sprintf("\t__**%s**#%s__: %d -> %d\n",
				doc.Username, doc.Discriminator, doc.Credits, doc.CreditsTotal)
			doc.Credits = doc.CreditsTotal
			if err := doc.Update(database); err != nil {
				return "", err
			}
		}
	}
	if !found {
		msg = "No users updated."
	}
	return msg, nil
}

func pasteIt(msg, title string) (string, error) {
	p := go_pastebin.NewPastebin(envPBDK)
	pb, err := p.GenerateUserSession(envPB, envPBPW)
	if err != nil {
		return "", err
	}

	paste, err := pb.Paste(msg, title, "c", "10M", "1")
	if err != nil {
		fmt.Println(err)
		return "", err
	}

	return paste.String(), nil
}

// Help prints help information for accessing script library.
func Help(f *flag.FlagSet, prefix string) string {
	var s string

	s += prefix

	f.VisitAll(func(fflag *flag.Flag) {
		s += fmt.Sprintf("  -%s", fflag.Name) // Two spaces before -; see next two comments.
		name, usage := flag.UnquoteUsage(fflag)
		if len(name) > 0 {
			s += " " + name
		}
		// Boolean flags of one ASCII letter are so common we
		// treat them specially, putting their usage on the same line.
		if len(s) <= 4 { // space, space, '-', 'x'.
			s += "\t"
		} else {
			// Four spaces before the tab triggers good alignment
			// for both 4- and 8-space tab stops.
			s += "\n    \t"
		}
		s += usage + "\n"
	})
	return s
}

func channelsTemp() string {
	var msg string
	msg += "```C\n"
	for k, c := range Bot.Links {
		gg := Bot.GetGuild(k)
		if strings.Contains(gg.Name, "verse") {
			msg += fmt.Sprintf("Guild: [%s] [%s]\n", gg.Name, gg.ID)
			for n, cc := range c {
				msg += fmt.Sprintf("[%2d] Channel: [%16s] [%5s] [%s]\n", n, cc.Name, cc.Type, cc.ID)
			}
		}
	}
	return msg + "```"
}
