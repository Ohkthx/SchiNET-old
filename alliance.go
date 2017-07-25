package main

import (
	"errors"
	"fmt"

	mgo "gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"

	"strings"

	"github.com/d0x1p2/godbot"
	"github.com/pborman/getopt/v2"
)

// Channel holds a channels GuildID and ChannelID
type Channel struct {
	GuildID     string
	GuildName   string
	ChannelID   string
	ChannelName string
}

// Alliance holds alliance data between two channels.
type Alliance struct {
	ID     bson.ObjectId `bson:"_id,omitempty"`
	Name   string
	Key    string
	PartyA Channel
	Party1 Channel
}

// Errors
var (
	ErrAllianceInit = errors.New("alliance is already initialised")
)

// CoreAlliance handles all alliance COMMAND actions
func (cfg *Config) CoreAlliance(io *IOdat) error {
	var name, key string
	var help, list, init, delete bool

	fl := getopt.New()

	fl.FlagLong(&delete, "delete", 'd', "Delete an Alliance.")
	fl.FlagLong(&name, "name", 'n', "Alliance Name.")
	fl.FlagLong(&key, "key", 'k', "Key to Join Alliance.")
	fl.FlagLong(&init, "init", 0, "Initialize a new Alliance.")
	fl.FlagLong(&help, "help", 'h', "This menus")
	fl.FlagLong(&list, "list", 'l', "List Guilds available.")

	if err := fl.Getopt(io.io, nil); err != nil {
		return err
	}
	if fl.NArgs() > 0 {
		if err := fl.Getopt(fl.Args(), nil); err != nil {
			return err
		}
	}

	// Prevent issues with mistyping case.
	if name != "" {
		name = strings.ToLower(name)
	}

	// Handle 'list' and 'help'
	if list {
		io.msgEmbed = embedCreator(cfg.Core.GuildsString(), ColorBlue)
		return nil
	} else if help {
		io.output = Help(fl, "", "")
		return nil
	} else if init {
		if err := cfg.AllianceInit(name, io.guild); err != nil {
			return err
		}
		passkey := fmt.Sprintf("Pass this key to other guild:\n**%s**", io.guild.ID)
		io.msgEmbed = embedCreator(passkey, ColorGreen)
		return nil
	} else if delete {
		if err := cfg.AllianceBreak(name); err != nil {
			return err
		}
	} else if key != "" {
		if err := cfg.AllianceJoin(name, key, io.guild); err != nil {
			return err
		}
		io.msgEmbed = embedCreator("You have joined the alliance.", ColorGreen)
		return nil
	}

	return nil
}

// AllianceInit creates a new alliance.
func (cfg *Config) AllianceInit(name string, guild *godbot.Guild) error {
	// Check if Initialized Alliance
	if name == "" {
		return errors.New("need to supply an alliance name ('--name')")
	}

	if ok := cfg.AlliancePending(name, guild.ID); ok {
		return ErrAllianceInit
	}

	if ok := cfg.AllianceExists(name); ok {
		return errors.New("an alliance with that name already exists")
	}

	// Create channel in parent guild.
	ch, err := cfg.Core.Session.GuildChannelCreate(guild.ID, name, "text")
	if err != nil {
		return err
	}

	var ally = Alliance{
		Name: name,
		Key:  guild.ID,
		PartyA: Channel{
			GuildID:     guild.ID,
			GuildName:   guild.Name,
			ChannelID:   ch.ID,
			ChannelName: ch.Name,
		},
	}
	cfg.pending = append(cfg.pending, ally)

	return nil
}

// AllianceJoin allows you to use another's key to join.
func (cfg *Config) AllianceJoin(name, key string, guild *godbot.Guild) error {
	if name == "" {
		return errors.New("need to supply an alliance name ('--name')")
	}
	if ok := cfg.AlliancePending(name, key); !ok {
		return errors.New("alliance hasn't been initialized with ('--init') or ('--key') is bad")
	}

	var ally Alliance
	for _, a := range cfg.pending {
		if a.Key == key {
			ally = a
		}
	}

	// Create Channel here.
	ch, err := cfg.Core.Session.GuildChannelCreate(guild.ID, name, "text")
	if err != nil {
		return err
	}

	var party = Channel{
		GuildID:     guild.ID,
		GuildName:   guild.Name,
		ChannelID:   ch.ID,
		ChannelName: ch.Name,
	}

	ally.Party1 = party
	cfg.Alliances = append(cfg.Alliances, ally)
	if err := ally.Update(); err != nil {
		return err
	}

	return nil
}

// AllianceExists looks up if an alliance already exists.
func (cfg Config) AllianceExists(name string) bool {
	for _, a := range cfg.Alliances {
		/*
			if a.PartyA.GuildID == partyA.GuildID || a.PartyA.GuildID == party1.GuildID {
				if a.Party1.GuildID == partyA.GuildID || a.Party1.GuildID == party1.GuildID {
					return true
				}
			}*/
		if a.Name == name {
			return true
		}
	}
	return false
}

// AlliancePending checks if the alliance is in pending status.
func (cfg *Config) AlliancePending(name, key string) bool {
	for _, a := range cfg.pending {
		if a.Name == name && a.Key == key {
			return true
		}
	}
	return false
}

// AllianceHandler will check if channel is an alliance and process messages.
func (cfg *Config) AllianceHandler(cID, content, username string) error {
	var ally Alliance
	var found bool
	for _, a := range cfg.Alliances {
		if a.PartyA.ChannelID == cID || a.Party1.ChannelID == cID {
			ally = a
			found = true
		}
	}

	// Not found? Return with no error.
	if !found {
		return nil
	}

	var rcvr string
	if ally.PartyA.ChannelID == cID {
		rcvr = ally.Party1.ChannelID
	} else {
		rcvr = ally.PartyA.ChannelID
	}

	var nc = "[ally]**" + username + "** --> " + content
	if _, err := cfg.Core.Session.ChannelMessageSend(rcvr, nc); err != nil {
		return err
	}
	return nil
}

// AllianceBreak cancels an alliance.
func (cfg *Config) AllianceBreak(name string) error {

	if ok := cfg.AllianceExists(name); !ok {
		return errors.New("no alliance exists with that name")
	}

	var cnt int
	var ally Alliance
	for n, a := range cfg.Alliances {
		if a.Name == name {
			ally = a
			cnt = n
		}
	}

	cfg.Alliances = append(cfg.Alliances[:cnt], cfg.Alliances[cnt+1:]...)
	if err := ally.Delete(); err != nil {
		return err
	}

	var msg = fmt.Sprintf("The [**%s**] alliance has fallen!", ally.Name)
	embed := embedCreator(msg, ColorMaroon)
	cfg.Core.Session.ChannelMessageSendEmbed(ally.Party1.GuildID, embed)
	cfg.Core.Session.ChannelMessageSendEmbed(ally.PartyA.GuildID, embed)

	cfg.Core.Session.ChannelDelete(ally.PartyA.ChannelID)
	cfg.Core.Session.ChannelDelete(ally.Party1.ChannelID)

	return nil
}

// Update replicates changes to a database for a particular alliance.
func (ally *Alliance) Update() error {
	var err error
	var q = make(map[string]interface{})
	var c = make(map[string]interface{})

	q["name"] = ally.Name
	c["$set"] = bson.M{
		"name":   ally.Name,
		"key":    ally.Key,
		"partya": ally.PartyA,
		"party1": ally.Party1,
	}

	var dbdat = DBdatCreate("config", CollectionAlliances, ally, q, c)
	err = dbdat.dbEdit(Alliance{})
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

// Delete removes a particular alliance from a database.
func (ally *Alliance) Delete() error {
	var q = make(map[string]interface{})

	q["name"] = ally.Name
	var dbdat = DBdatCreate("config", CollectionAlliances, ally, q, nil)
	if err := dbdat.dbGet(Alliance{}); err != nil {
		if err == mgo.ErrNotFound {
			return nil
		}
		return err
	}

	var a = dbdat.Document.(Alliance)
	if err := dbdat.dbDeleteID(a.ID); err != nil {
		return err
	}

	return nil
}

// AlliancesLoad grabs all current alliances from database.
func (cfg *Config) AlliancesLoad() error {
	dbdat := DBdatCreate("config", CollectionAlliances, Alliance{}, nil, nil)
	err := dbdat.dbGetAll(Alliance{})
	if err != nil {
		if err == mgo.ErrNotFound {
			return errors.New("no alliances in database")
		}
		return err
	}

	var doc Alliance
	for _, d := range dbdat.Documents {
		doc = d.(Alliance)
		cfg.Alliances = append(cfg.Alliances, doc)
	}
	return nil
}
