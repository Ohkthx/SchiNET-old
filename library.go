package main

import (
	"errors"
	"fmt"
	"time"

	"flag"

	"github.com/bwmarrin/discordgo"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

// Error constants for script library.
var (
	ErrScriptNotFound = errors.New("script appears to not be in database, check username and script name")
	//ErrBadUsername    = fmt.Errorf("bad user name supplied\n%s", scriptReqSyntax)
	//ErrBadScript      = fmt.Errorf("bad script name supplied\n%s", scriptReqSyntax)
	//ErrBadArgs        = errors.New("bad arguments supplied")
)

const (
	scriptSyntaxGet  = ",script  -user \"Username\"   -name \"Name Here\""
	scriptSyntaxAdd  = ",script  -add   -name \"Name Here\"   -script \"Text goes here\""
	scriptSyntaxEdit = ",script  -edit   -name \"Name Here\"   -script \"Text goes here\""
	scriptSyntaxDel  = ",script  -del   -name \"Name Here\"   -script \"Text goes here\""

	collectionName = "scripts"

	argAdd = 1 << iota
	argEdit
	argDel
	argHelp
	argList
	argUser
	argName
	argScript
)

// Library holds session data for accessing the information.
type Library struct {
	Arguments map[string]string // Contains arguments such as -h, -u, -s
	Args      int
	ArgHelp   bool
	ArgAdd    bool
	ArgEdit   bool
	ArgDel    bool
	ArgList   bool
	Database  string // Database to store information on.
	Flags     *flag.FlagSet
	Script    *Script
}

// Script contains information pretaining to a specific script from a database.
type Script struct {
	ID           bson.ObjectId `bson:"_id,omitempty"`
	Name         string
	Author       *discordgo.User
	Content      string
	Length       int
	URL          string
	DateAdded    time.Time
	DateModified time.Time
	DateAccessed time.Time
}

// Core handles all initial requests.
func scriptCore(server string, user *discordgo.User, io []string, help bool) (string, error) {
	var msg string
	var err error
	var lib *Library

	lib, err = New(server, user, io, help)
	if err != nil {
		return "", err
	}

	switch {
	case lib.ArgAdd, lib.ArgEdit:
		// Add script in Database,
		msg, err = lib.Add()
	case lib.ArgDel && lib.Args&argName == argName:
		// Delete script in Database.
		msg, err = lib.Delete()
	case lib.ArgList:
		// Get script from Database.
		msg, err = lib.List()
	case lib.Args&(argUser|argName) == (argUser | argName):
		msg, err = lib.Get()
	case lib.ArgHelp:
		fallthrough
	default:
		msg = lib.Help()
	}

	if err != nil {
		return "", err
	}

	// Embeded object here? Or return msg.
	//io.msgEmbed = embedCreator(msg, ColorGreen)

	return msg, nil
}

// New creates a new instance of Script.
func New(database string, user *discordgo.User, io []string, help bool) (*Library, error) {
	tn := time.Now()

	var lib = &Library{
		Database: database,
		ArgHelp:  help,
	}

	err := lib.setArguments(io)
	if err != nil {
		return nil, err
	}

	var script = &Script{
		Name:         lib.Arguments["name"],
		Author:       user,
		Content:      lib.Arguments["script"],
		Length:       len(lib.Arguments["script"]),
		DateAdded:    tn,
		DateModified: tn,
		DateAccessed: tn,
	}

	lib.Script = script

	return lib, nil
}

// Add a script to the library.
func (lib *Library) Add() (string, error) {
	var err error

	if lib.Args&(argName|argScript) != (argName | argScript) {
		if lib.ArgAdd {
			return scriptSyntaxAdd, nil
		}
		return scriptSyntaxEdit, nil
	}

	s, err := lib.find(false)
	if err != nil {
		if err == ErrScriptNotFound {
			// Get URL
			lib.Script.URL, err = pasteIt(lib.Script.Content, lib.Script.Name)
			if err != nil {
				return "", err
			}

			// Add here since doesn't exists
			dbdat := DBdatCreate(lib.Database, CollectionScripts, lib.Script, nil, nil)
			err = dbdat.dbInsert()
			if err != nil {
				return "", err
			}

			msg := lib.Script.String()
			return msg, err
		}
		// Another error occured, return it.
		return "", err
	}

	if lib.ArgEdit {
		return lib.Edit(s)
	}

	return "already exists, try editing.", nil
}

// Edit a script in the library.
func (lib *Library) Edit(changes *Script) (string, error) {
	s := lib.Script
	var err error
	var q = make(map[string]interface{})
	var c = make(map[string]interface{})

	q["$and"] = []bson.M{bson.M{"name": s.Name}, bson.M{"author.username": s.Author.Username}}

	tn := time.Now()
	// Get URL of New Paste.
	s.URL, err = pasteIt(s.Content, s.Name)
	if err != nil {
		return "", err
	}

	/*
		s.URL = changes.URL
		s.Content = changes.Content
		s.Length = len(changes.Content)
	*/
	c["$set"] = bson.M{
		"url":          s.URL,
		"content":      s.Content,
		"length":       s.Length,
		"datemodified": tn,
		"dateaccessed": tn,
	}

	dbdat := DBdatCreate(lib.Database, CollectionScripts, s, q, c)
	err = dbdat.dbEdit(Script{})
	if err != nil {
		return "", err
	}

	uA := usernameAdd(s.Author.Username, s.Author.Discriminator)
	msg := fmt.Sprintf(
		"__**%s** edited **%s**__"+
			"**Added by**: %s\n"+
			"**Date Added**: %s\n"+
			"**Date Modified**: %s\n\n"+
			"**URL**: [%s](%s) by %s\n"+
			"**(Script will only be avaible for __10 minutes__.)*",
		uA, s.Name,
		uA,
		s.DateAdded.Format(time.UnixDate),
		s.DateModified.Format(time.UnixDate),
		s.Name, s.URL, uA,
	)

	return msg, nil
}

// Delete a script from a library.
func (lib *Library) Delete() (string, error) {

	if lib.Args&(argName|argScript) != (argName | argScript) {
		return scriptSyntaxDel, nil
	}

	s := lib.Script
	var err error
	var q = make(map[string]interface{})

	q["$and"] = []bson.M{bson.M{"name": s.Name}, bson.M{"author.username": s.Author.Username}}
	dbdat := DBdatCreate(lib.Database, CollectionScripts, lib.Script, q, nil)
	err = dbdat.dbDelete()
	if err != nil {
		return "", err
	}

	msg := fmt.Sprintf("**%s** deleted -> **%s**\n  It will be missed...", s.Author.Username, lib.Script.Name)
	return msg, nil
}

func (lib *Library) find(requested bool) (*Script, error) {
	s := lib.Script
	var q = make(map[string]interface{})

	user := lib.Arguments["user"]
	if user == "" {
		user = s.Author.Username
	}

	q["$and"] = []bson.M{bson.M{"name": s.Name}, bson.M{"author.username": user}}

	dbdat := DBdatCreate(lib.Database, CollectionScripts, Script{}, q, nil)
	err := dbdat.dbGet(Script{})
	if err != nil {
		if err == mgo.ErrNotFound {
			return nil, ErrScriptNotFound
		}
		return nil, err
	}

	var script Script
	script = dbdat.Document.(Script)
	lib.Script = &script

	if requested {
		tn := time.Now()
		d := tn.Sub(lib.Script.DateAccessed)

		if int(d.Minutes()) > 10 {
			// Get URL of New Paste.
			lib.Script.URL, err = pasteIt(lib.Script.Content, lib.Script.Name)
			if err != nil {
				return nil, err
			}
			err = lib.setAccessed()
			if err != nil {
				return nil, err
			}
		}
	}

	return lib.Script, nil
}

// Help prints help information for accessing script library.
func (lib *Library) Help() string {
	f := lib.Flags
	var s string

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

// Get gets a script from the database/library.
func (lib *Library) Get() (string, error) {

	s, err := lib.find(true)
	if err != nil {
		return "", err
	}

	return s.String(), nil

}

// List gets all scripts from library.
func (lib *Library) List() (string, error) {
	dbdat := DBdatCreate(lib.Database, CollectionScripts, Script{}, nil, nil)
	err := dbdat.dbGetAll(Script{})
	if err != nil {
		if err == mgo.ErrNotFound {
			return "", ErrScriptNotFound
		}
		return "", err
	}

	var docs []Script
	var doc Script
	for _, d := range dbdat.Documents {
		doc = d.(Script)
		docs = append(docs, doc)
	}

	var msg = "Current Scripts in Library:\n"
	for n, d := range docs {
		msg += fmt.Sprintf("  [%d] %s -> %s\n", n, d.Author.Username, d.Name)
	}
	msg += fmt.Sprintf("\nTo request a script, type:\n%s", scriptSyntaxGet)

	return msg, nil
}

// setArguments assists in initiating the library.
func (lib *Library) setArguments(io []string) error {
	lib.Arguments = make(map[string]string)
	var user, name, script string

	lib.Flags = flag.NewFlagSet("script", flag.ContinueOnError)
	lib.Flags.StringVar(&user, "user", "", "Username")
	lib.Flags.StringVar(&name, "name", "", "Script name")
	lib.Flags.StringVar(&script, "script", "", "Script text")
	lib.Flags.BoolVar(&lib.ArgAdd, "add", false, "Add a new script")
	lib.Flags.BoolVar(&lib.ArgEdit, "edit", false, "Edit an existing script")
	lib.Flags.BoolVar(&lib.ArgDel, "del", false, "Delete a script")
	lib.Flags.BoolVar(&lib.ArgList, "list", false, "List all scripts")
	if lib.ArgHelp == false {
		lib.Flags.BoolVar(&lib.ArgHelp, "help", true, "this message")
	}
	err := lib.Flags.Parse(io[1:])
	if err != nil {
		return err
	}

	lib.Arguments["user"] = user
	lib.Arguments["name"] = name
	lib.Arguments["script"] = script

	if user != "" {
		lib.Args |= argUser
	}

	if name != "" {
		lib.Args |= argName
	}

	if script != "" {
		lib.Args |= argScript
	}

	return nil
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
func (lib *Library) setAccessed() error {
	s := lib.Script
	var q = make(map[string]interface{})
	var c = make(map[string]interface{})

	q["$and"] = []bson.M{bson.M{"name": s.Name}, bson.M{"author.username": s.Author.Username}}

	c["$set"] = bson.M{
		"url":          s.URL,
		"dateaccessed": time.Now(),
	}

	dbdat := DBdatCreate(lib.Database, CollectionScripts, s, q, c)
	err := dbdat.dbEdit(Script{})
	if err != nil {
		return err
	}

	return nil
}
