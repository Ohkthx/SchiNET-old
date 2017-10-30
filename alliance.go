package main

import (
	"errors"
	"fmt"

	mgo "gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"

	"strings"

	"github.com/bwmarrin/discordgo"
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
func (cfg *Config) CoreAlliance(dat *IOdata) error {
	var name, key string
	var help, list, init, delete bool

	fl := getopt.New()

	fl.FlagLong(&delete, "delete", 'd', "Delete an Alliance.")
	fl.FlagLong(&name, "name", 'n', "Alliance Name.")
	fl.FlagLong(&key, "key", 'k', "Key to Join Alliance.")
	fl.FlagLong(&init, "init", 0, "Initialize a new Alliance.")
	fl.FlagLong(&help, "help", 'h', "This menus")
	fl.FlagLong(&list, "list", 'l', "List Guilds available.")

	if err := fl.Getopt(dat.io, nil); err != nil {
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

	// Handle the various commands: LIST, HELP, INIT, AND DELETE
	if list {
		dat.msgEmbed = embedCreator(cfg.Core.GuildsString(), ColorBlue)
		return nil
	} else if help {
		dat.output = Help(fl, "", "")
		return nil
	} else if init {
		if err := cfg.AllianceInit(name, dat.guild); err != nil {
			return err
		}
		passkey := fmt.Sprintf("Pass this key to other guild:\n**%s**", dat.guild.ID)
		dat.msgEmbed = embedCreator(passkey, ColorGreen)
		return nil
	} else if delete {
		if ok := dat.user.HasPermissionGTE(dat.guild.Name, permModerator); !ok {
			return ErrBadPermissions
		}
		if err := cfg.AllianceBreak(name); err != nil {
			return err
		}
	} else if key != "" {
		if err := cfg.AllianceJoin(name, key, dat.guild); err != nil {
			return err
		}
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

	var ally = Alliance{
		Name: name,
		Key:  guild.ID,
		PartyA: Channel{
			GuildID:   guild.ID,
			GuildName: guild.Name,
		},
	}
	cfg.pending = append(cfg.pending, ally)

	return nil
}

// AllianceJoin allows you to use another's key to join.
func (cfg *Config) AllianceJoin(name, key string, guild *godbot.Guild) error {
	if name == "" {
		return errors.New("need to supply an alliance name ('--name')")
	} else if ok := cfg.AlliancePending(name, key); !ok {
		return errors.New("alliance hasn't been initialized with ('--init') or ('--key') is bad")
	} else if key == guild.ID {
		return errors.New("you cannot ally yourself")
	}

	var ally Alliance
	var loc int
	for n, a := range cfg.pending {
		if a.Key == key {
			ally = a
			loc = n
		}
	}

	// Create Channel here.
	cha, err := cfg.Core.Session.GuildChannelCreate(ally.PartyA.GuildID, name, "text")
	if err != nil {
		return err
	}

	// Create channel in parent guild.
	ch1, err := cfg.Core.Session.GuildChannelCreate(guild.ID, name, "text")
	if err != nil {
		return err
	}

	ally.PartyA.ChannelID = cha.ID
	ally.PartyA.ChannelName = cha.Name

	var party1 = Channel{
		GuildID:     guild.ID,
		GuildName:   guild.Name,
		ChannelID:   ch1.ID,
		ChannelName: ch1.Name,
	}

	ally.Party1 = party1

	// Remove from pending list.
	cfg.pending = append(cfg.pending[:loc], cfg.pending[loc+1:]...)
	cfg.Alliances = append(cfg.Alliances, ally)
	if err := ally.Update(); err != nil {
		return err
	}

	var msg = fmt.Sprintf("The [**%s**] alliance been created!", ally.Name)
	embed := embedCreator(msg, ColorGreen)
	cfg.Core.Session.ChannelMessageSendEmbed(ally.Party1.GuildID, embed)
	cfg.Core.Session.ChannelMessageSendEmbed(ally.PartyA.GuildID, embed)

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
func (cfg *Config) allianceHandler(m *discordgo.Message) error {
	var ally Alliance
	var found bool
	cID := m.ChannelID
	username := m.Author.Username

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

	var rcvID string
	if ally.PartyA.ChannelID == cID {
		rcvID = ally.Party1.ChannelID
	} else {
		rcvID = ally.PartyA.ChannelID
	}

	// Convert from an @mention to plain text
	// TAG: TODO

	// Scan for @mentions
	var content string
	cSplit := strings.Split(m.Content, " ")
	for _, w := range cSplit {
		// Potentially a user.
		if strings.HasPrefix(w, "@") {
			un := strings.TrimPrefix(w, "@")
			u := UserNew(nil)
			if err := u.GetByName(un); err == nil {
				w = "<@" + u.ID + ">"
			}
		}
		content += w + " "
	}

	var nc = "[ally]**" + username + "** --> " + content + "\n"
	for _, a := range m.Attachments {
		nc += "\n" + a.URL
	}
	if _, err := cfg.Core.Session.ChannelMessageSend(rcvID, nc); err != nil {
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

	// Remove the alliance from the current maintained alliances.
	cfg.Alliances = append(cfg.Alliances[:cnt], cfg.Alliances[cnt+1:]...)
	if err := ally.Delete(); err != nil {
		return err
	}

	// Send out the notification to both server.
	var msg = fmt.Sprintf("The [**%s**] alliance has fallen!", ally.Name)
	embed := embedCreator(msg, ColorMaroon)
	cfg.Core.Session.ChannelMessageSendEmbed(ally.Party1.GuildID, embed)
	cfg.Core.Session.ChannelMessageSendEmbed(ally.PartyA.GuildID, embed)

	// Cleanup the channels and remove the alliance channel from each server.
	// TAG: TODO - Error handling incase deletion fails.
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

	// Construct the the query for the database.
	var dbdat = DBdataCreate("config", CollectionAlliances, ally, q, c)

	// Edit the databases version of the alliance.
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
	var dbdat = DBdataCreate("config", CollectionAlliances, ally, q, nil)
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
	dbdat := DBdataCreate("config", CollectionAlliances, Alliance{}, nil, nil)
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
