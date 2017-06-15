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
	ErrBadAmount = errors.New("not enough to gamble")
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
	toGamble, err := strconv.Atoi(io.io[1])
	if err != nil {
		return ErrBadArgs
	}
	if toGamble < GambleMin {
		return ErrBadAmount
	}
	// Get user's credits from Database

	u, err := userGet(io.msg.ChannelID, io.user.ID)
	if err != nil {
		return err
	}

	if toGamble > u.Credits {
		return ErrBadAmount
	}

	var msg string
	spoils := creditsGambleResult(toGamble)
	if spoils <= 0 {
		u.Credits -= toGamble
		msg = fmt.Sprintf("<@%s> lost %d %s.\n%10d remaining.", io.user.ID, toGamble, GambleCredits, u.Credits)
		spoils = -toGamble
		err = userUpdate(io.msg.ChannelID, Bot.User, toGamble)
		if err != nil {
			return err
		}
	} else {
		u.Credits += spoils
		msg = fmt.Sprintf("<@%s> won %d %s.\n%10d in the bank.", io.user.ID, spoils, GambleCredits, u.Credits)
	}

	err = userUpdate(io.msg.ChannelID, io.msg.Author, spoils)
	if err != nil {
		return err
	}

	io.msgEmbed = embedCreator(msg, ColorYellow)

	return nil
}

func creditsGambleResult(credits int) int {
	s1 := rand.NewSource(time.Now().UnixNano())
	r1 := rand.New(s1)

	num := r1.Intn(100)
	if num < 60 {
		// Lose all
		credits = 0
	} else if num >= 60 && num < 98 {
		// Win x2
		credits *= 2
	} else if num >= 98 {
		// Win x3
		credits *= 3
	}

	return credits
}

func (io *IOdat) creditsTransfer() error {
	amt, err := strconv.Atoi(io.io[2])
	if err != nil {
		return ErrBadAmount
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
