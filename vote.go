package main

import (
	"errors"
	"fmt"

	"github.com/bwmarrin/discordgo"
	"github.com/pborman/getopt/v2"
)

// Constants for producing helpful text for normal command operations.
const (
	voteSyntaxAdd  = ",vote   --title \"Title Here\"\n"
	voteSyntaxDesc = ",vote   --title \"Title Here\"   -d \"Description Here\"\n"
	voteSyntaxAll  = voteSyntaxAdd + voteSyntaxDesc
)

// CoreVote processes all voting related additions.
func (dat *IOdata) CoreVote() error {

	if ok := dat.user.HasRoleType(dat.guildConfig, rolePermissionMod); !ok {
		return ErrBadPermissions
	}

	fl := getopt.New()
	var title, description string
	var help bool

	// Generics
	fl.FlagLong(&title, "title", 't', "Title of the poll.")
	fl.FlagLong(&description, "description", 'd', "Description")
	fl.FlagLong(&help, "help", 'h', "This message")

	if err := fl.Getopt(dat.io, nil); err != nil {
		return err
	}
	if fl.NArgs() > 0 {
		if err := fl.Getopt(fl.Args(), nil); err != nil {
			return err
		}
	}

	if title != "" {
		// Create #vote here and create the poll.
		return dat.voteCreate(title, description)
	}

	// Print issue + help
	prefix := "**Need** __title__.\n\n"
	suffix := "\n\nExamples:\n" + voteSyntaxAll
	dat.output = Help(fl, prefix, suffix)
	return nil
}

// voteCreate setups the channel and the poll's message.
func (dat *IOdata) voteCreate(title, description string) error {
	s := dat.session
	// Get our channels from the server.
	channels, err := s.GuildChannels(dat.guild.ID)
	if err != nil {
		return err
	}

	// Check if the channel exists already.
	var ch *discordgo.Channel
	for _, c := range channels {
		if c.Name == "vote" {
			ch = c
			break
		}
	}

	// Channel doesn't exists. Create it.
	if ch == nil {
		if ch, err = s.GuildChannelCreate(dat.guild.ID, "vote", "text"); err != nil {
			return errors.New("Error creating poll, try again")
		}
		// Set the permissions to disable adding messages and additional emojis
		if err := s.ChannelPermissionSet(ch.ID, dat.guild.ID, "role", 0, 0x00000840); err != nil {
			return errors.New("Error creating poll, try again")
		}
	}

	// Create the polls message based on the title and description.
	var msgTxt = "[POLL]  **" + title + "**"
	if description != "" {
		msgTxt += fmt.Sprintf("\n```%s```", description)
	}

	var msg *discordgo.Message
	if msg, err = s.ChannelMessageSend(ch.ID, msgTxt); err != nil {
		return err
	}

	// Add our reactions (thumbsup and thumbsdown)
	// Thumbs up: U+1F44D / 128077
	if err = s.MessageReactionAdd(ch.ID, msg.ID, emojiIntToStr(128077)); err != nil {
		if err1 := s.ChannelMessageDelete(ch.ID, msg.ID); err1 != nil {
			return err
		}
		return errors.New("Error creating poll, try again")
	}

	// Thumps down: U+1F44E / 128078
	if err = s.MessageReactionAdd(ch.ID, msg.ID, emojiIntToStr(128078)); err != nil {
		if err1 := s.ChannelMessageDelete(ch.ID, msg.ID); err1 != nil {
			return err
		}
		return errors.New("Error creating poll, try again")
	}

	// Respond to the channel it was created in that the poll now exists.
	dat.output = "Poll created! Check out <#" + ch.ID + "> to participate."

	return nil
}
