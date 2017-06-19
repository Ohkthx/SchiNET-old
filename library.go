package main

import (
	"errors"
	"fmt"
	"time"

	"github.com/bwmarrin/discordgo"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

// Error constants for script library.
var (
	ErrScriptNotFound = errors.New("script appears to not be in database, check username and script name")
)

// Script contains information pretaining to a specific script from a database.
type Script struct {
	ID           bson.ObjectId `bson:"_id,omitempty"`
	Server       string
	Name         string
	Author       *discordgo.User
	Content      string
	Length       int
	URL          string
	DateAdded    time.Time
	DateModified time.Time
	DateAccessed time.Time
}

func (io *IOdat) scriptCore() error {
	var msg string
	var err error
	switch io.io[0] {
	case "add", "edit":
		// Add script in Database,
		msg, err = scriptAdd(io)
	case "del":
		// Delete script in Database.
		err = errors.New("current not working")
	case "script", "scripts":
		// Get script from Database.
		if len(io.io) != 3 {
			return ErrBadArgs
		}
		var s *Script
		s, err = scriptGet(io.guild.Name, io.io[1], io.io[2], true)
		msg = s.String()
	}

	if err != nil {
		return err
	}

	io.msgEmbed = embedCreator(msg, ColorGreen)

	return nil
}

func scriptAdd(io *IOdat) (string, error) {
	var script *Script
	var err error
	if len(io.io) != 4 {
		return "", ErrBadArgs
	}

	script, err = scriptNew(io.guild.Name, io.io[2], io.io[3], io.msg.Author)
	if err != nil {
		return "", err
	}

	s, err := scriptGet(script.Server, script.Author.Username, script.Name, false)
	if err != nil {
		if err == ErrScriptNotFound {
			// Get URL
			script.URL, err = pasteIt(script.Content, script.Name)
			if err != nil {
				return "", err
			}

			// Add here since doesn't exists
			dbdat := DBdatCreate(io.guild.Name, CollectionScripts, script, nil, nil)
			err = dbdat.dbInsert()
			if err != nil {
				return "", err
			}

			msg := script.String()
			return msg, err
		}
		// Another error occured, return it.
		return "", err
	}

	if io.io[0] == "edit" {
		return scriptEdit(s)
	}

	return "already exists, try editing.", nil
}

func scriptDel(io *IOdat) error {
	return nil
}

func scriptEdit(s *Script) (string, error) {
	var err error
	var q = make(map[string]interface{})
	var c = make(map[string]interface{})

	/*
		_, err := scriptGet(gname, s.Author.Username, s.Name)
		if err != nil {
			if err != mgo.ErrNotFound {
				return "", err
			}
			return "", ErrScriptNotFound
		}
	*/

	q["$and"] = []bson.M{bson.M{"name": s.Name}, bson.M{"author.username": s.Author.Username}}

	tn := time.Now()
	// Get URL of New Paste.
	s.URL, err = pasteIt(s.Content, s.Name)
	if err != nil {
		return "", err
	}

	c["$set"] = bson.M{
		"url":          s.URL,
		"content":      s.Content,
		"length":       len(s.Content),
		"datemodified": tn,
		"dateaccessed": tn,
	}

	dbdat := DBdatCreate(s.Server, CollectionScripts, s, q, c)
	err = dbdat.dbEdit(Script{})
	if err != nil {
		return "", err
	}

	return s.String(), nil
}

// Print returns a string of information about a script.
func (s *Script) String() string {
	if s == nil {
		return "script information could not be obtained."
	}

	var uA = usernameAdd(s.Author.Username, s.Author.Discriminator)

	msg := fmt.Sprintf(
		"__Script Name: %s__\n\n"+
			"**Added by**: %s\n"+
			"**Date Added**: %s\n"+
			"**Date Modified**: %s\n\n"+
			"**URL**: [%s](%s) by %s\n"+
			"**(Script will only be avaible for __10 minutes__.)*",
		s.Name,
		uA,
		s.DateAdded.Format(time.UnixDate),
		s.DateModified.Format(time.UnixDate),
		s.Name, s.URL, uA,
	)
	return msg
}

// SetAccessed updates database with new timestamp.
func (s *Script) SetAccessed() error {
	var q = make(map[string]interface{})
	var c = make(map[string]interface{})

	q["$and"] = []bson.M{bson.M{"name": s.Name}, bson.M{"author.username": s.Author.Username}}

	c["$set"] = bson.M{
		"url":          s.URL,
		"dateaccessed": time.Now(),
	}

	dbdat := DBdatCreate(s.Server, CollectionScripts, s, q, c)
	err := dbdat.dbEdit(Script{})
	if err != nil {
		return err
	}

	return nil
}

func scriptGet(database, username, name string, requested bool) (*Script, error) {
	var q = make(map[string]interface{})
	q["$and"] = []bson.M{bson.M{"name": name}, bson.M{"author.username": username}}

	dbdat := DBdatCreate(database, CollectionScripts, Script{}, q, nil)
	err := dbdat.dbGet(Script{})
	if err != nil {
		if err == mgo.ErrNotFound {
			return nil, ErrScriptNotFound
		}
		return nil, err
	}

	var s Script
	var script *Script
	s = dbdat.Document.(Script)
	script = &s

	if requested {
		tn := time.Now()
		d := tn.Sub(script.DateAccessed)

		if int(d.Minutes()) > 10 {
			// Get URL of New Paste.
			s.URL, err = pasteIt(script.Content, script.Name)
			if err != nil {
				return nil, err
			}
			err = script.SetAccessed()
			if err != nil {
				return nil, err
			}
		}
	}

	return &s, nil
}

func scriptNew(gID, name, content string, author *discordgo.User) (*Script, error) {
	if name == "" || author == nil || content == "" {
		return nil, ErrBadArgs
	}

	tn := time.Now()

	var script = &Script{
		Name:         name,
		Server:       gID,
		Author:       author,
		Content:      content,
		Length:       len(content),
		DateAdded:    tn,
		DateModified: tn,
		DateAccessed: tn,
	}

	return script, nil
}
