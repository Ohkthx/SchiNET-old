package main

import (
	"errors"
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

// Error constants for misc functions.
var (
	ErrGambleBadAmount = errors.New("bad amount of credits for action")
	ErrGambleNotMin    = fmt.Errorf("did not provide enough to reach minimum the of %d", GambleMin)
	ErrGambleNotEnough = errors.New("not enough credits")
)

// Constants for Gambling.
const (
	GambleCredits = "shards"
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
	var toGamble, spoils int
	var err error

	if strings.ToLower(io.io[1]) == "all" {
		all = true
	} else {
		toGamble, err = strconv.Atoi(io.io[1])
		if err != nil {
			return ErrBadArgs
		}
		if toGamble < GambleMin {
			return ErrGambleNotMin
		}
	}
	// Get user's credits from Database

	u, err := userGet(io.msg.ChannelID, io.user.ID)
	if err != nil {
		return err
	}

	if all != true {
		if toGamble > u.Credits {
			return ErrGambleNotEnough
		}
		spoils = creditsGambleResult(65, 34, 1, toGamble)
	} else {
		toGamble = u.Credits
		spoils = creditsGambleResult(59, 35, 2, u.Credits)
		//spoils -= u.Credits
	}

	var msg string
	if spoils <= 0 {
		u.Credits -= toGamble
		msg = fmt.Sprintf("<@%s> gambled **%d** %s\n"+
			"Result: **loss**\n"+
			"%s remaining in bank: **%d**.",
			io.user.ID, toGamble, GambleCredits, strings.Title(GambleCredits), u.Credits)
		spoils = -toGamble
		err = userUpdate(io.msg.ChannelID, Bot.User, toGamble)
		if err != nil {
			return err
		}
	} else {
		if all {
			u.Credits = spoils
			spoils -= toGamble
		} else {
			u.Credits += spoils
		}
		msg = fmt.Sprintf("<@%s> gambled **%d** %s\n"+
			"Result: **Won**    spoils: **%d**\n"+
			"%s remaining in bank: **%d**.",
			io.user.ID, toGamble, GambleCredits, spoils, strings.Title(GambleCredits), u.Credits)

	}

	err = userUpdate(io.msg.ChannelID, io.msg.Author, spoils)
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
	amt, err := strconv.Atoi(io.io[2])
	if err != nil {
		return ErrGambleBadAmount
	}

	s := strings.FieldsFunc(io.io[1], idSplit)
	u2ID := s[0]

	u1, err := userGet(io.msg.ChannelID, io.user.ID)
	if err != nil {
		return err
	}

	u2, err := userGet(io.msg.ChannelID, u2ID)
	if err != nil {
		return err
	}

	if u1.Credits >= amt {
		err = userUpdate(io.msg.ChannelID, io.msg.Author, -amt)
		if err != nil {
			return err
		}
		dg := &discordgo.User{ID: u2.ID, Username: u2.Username, Discriminator: u2.Discriminator, Bot: u2.Bot}
		err = userUpdate(io.msg.ChannelID, dg, amt)
		if err != nil {
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
	if len(io.io) < 2 {
		return ErrBadArgs
	} else if len(io.io) == 3 && strings.ToLower(io.io[0]) == "xfer" {
		err := io.creditsTransfer()
		if err != nil {
			return err
		}
		return nil
	}

	s := strings.FieldsFunc(io.io[1], idSplit)
	id := s[0]

	u, err := userGet(io.msg.ChannelID, id)
	if err != nil {
		return err
	}

	io.msgEmbed = userEmbedCreate(u)
	return nil

}
