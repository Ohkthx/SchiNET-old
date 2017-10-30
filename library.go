package main

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"strings"

	"bytes"

	"github.com/bwmarrin/discordgo"
	"github.com/pborman/getopt/v2"
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
	scriptSyntaxGet  = ",script  --get   --user \"Username\"   --title \"Name Here\"\n"
	scriptSyntaxAdd  = ",script  --add   --title \"Name Here\"   [Attach .txt File]\n"
	scriptSyntaxEdit = ",script  --edit   --title \"Name Here\"   [Attach .txt File]\n"
	scriptSyntaxDel  = ",script  --remove   --title \"Name Here\"\n"
	scriptSyntaxAll  = "\n\n" + scriptSyntaxAdd + scriptSyntaxEdit + scriptSyntaxDel + scriptSyntaxGet

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
	Database    string // Database to store information on.
	Attachments []*discordgo.MessageAttachment
	Flags       *getopt.Set
	Script      *Script
}

// Script contains information pretaining to a specific script from a database.
type Script struct {
	ID           bson.ObjectId `bson:"_id,omitempty"`
	Name         string
	Author       UserBasic
	Content      string
	Length       int
	URL          string
	Version      float32
	DateAdded    time.Time
	DateModified time.Time
	DateAccessed time.Time
}

// CoreLibrary handles all script/library requests.
func (dat *IOdata) CoreLibrary() error {
	var err error
	var msg string

	var add, edit, remove, get, list, help bool
	var version float32
	var user, name string

	lib := LibraryNew(dat.guild.Name, dat.msg.Attachments)

	fl := getopt.New()

	fl.FlagLong(&add, "add", 0, "Add a script")
	fl.FlagLong(&edit, "edit", 0, "Edit a script")
	fl.FlagLong(&remove, "remove", 0, "Remove a script")
	fl.FlagLong(&get, "get", 'g', "Get a script")
	fl.FlagLong(&user, "user", 0, "Script Owner")
	fl.FlagLong(&name, "title", 't', "Title of script")
	fl.FlagLong(&version, "version", 'v', "Versioning")
	fl.FlagLong(&list, "list", 'l', "List all script in Library")
	fl.FlagLong(&help, "help", 'h', "Help")

	if err := fl.Getopt(dat.io, nil); err != nil {
		return err
	}
	if fl.NArgs() > 0 {
		if err := fl.Getopt(fl.Args(), nil); err != nil {
			return err
		}
	}

	lib.Script = ScriptNew(name, "", version, dat.user.Basic())

	if (add || edit) && name != "" {
		msg, err = lib.Add()
	} else if remove && name != "" {
		// Ability to delete as a moderator with specifying the --user flag.
		if user != "" {
			if ok := dat.user.HasPermission(dat.guild.ID, permModerator); !ok {
				return ErrBadPermissions
			}
			lib.Script.Author.Name = user
		}
		msg, err = lib.Delete()
	} else if get && name != "" && user != "" {
		lib.Script.Author.Name = user
		msg, err = lib.Get()
	} else if list {
		// List scripts in Database.
		dat.output, err = lib.List()
		if err != nil {
			return err
		}
		return nil
	}

	if err != nil {
		return err
	} else if msg != "" {
		dat.msgEmbed = embedCreator(msg, ColorGreen)
		return nil
	}

	dat.output = Help(fl, "", scriptSyntaxAll)

	return nil
}

// LibraryNew creates a new instance of Script.
func LibraryNew(database string, attachs []*discordgo.MessageAttachment) *Library {
	var lib = &Library{
		Database:    database,
		Attachments: attachs,
	}

	return lib
}

// Add a script to the library.
func (lib *Library) Add() (string, error) {
	var err error

	// Check if attachment is good.
	if len(lib.Attachments) != 1 {
		return "", errors.New("need to provide ONE and only ONE attachment")
	} else if strings.HasSuffix(lib.Attachments[0].Filename, ".txt") == false {
		return "", errors.New("bad file extension for uploaded script, want: .txt")
	}

	attach := lib.Attachments[0]

	txt, err := getFile(attach.Filename, attach.URL)
	if err != nil {
		return "", err
	}
	lib.Script.Content = txt
	lib.Script.Length = len(txt)

	s, err := lib.find(false)
	if err != nil {
		if err == ErrScriptNotFound {
			// Get URL
			lib.Script.URL, err = pasteIt(lib.Script.Content, lib.Script.Name)
			if err != nil {
				return "", err
			}

			// Add here since doesn't exists
			dbdat := DBdataCreate(lib.Database, CollectionScripts, lib.Script, nil, nil)
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

	s.Content = txt
	s.Length = len(txt)
	s.Version = lib.Script.Version
	return lib.Edit(s)
}

// Edit a script in the library.
func (lib *Library) Edit(changes *Script) (string, error) {
	s := lib.Script
	var err error
	var q = make(map[string]interface{})
	var c = make(map[string]interface{})

	q["$and"] = []bson.M{bson.M{"name": s.Name}, bson.M{"author.name": s.Author.Name}}

	tn := time.Now()
	// Get URL of New Paste.
	s.URL, err = pasteIt(s.Content, s.Name)
	if err != nil {
		return "", err
	}

	c["$set"] = bson.M{
		"url":          s.URL,
		"content":      s.Content,
		"length":       s.Length,
		"version":      s.Version,
		"datemodified": tn,
		"dateaccessed": tn,
	}

	dbdat := DBdataCreate(lib.Database, CollectionScripts, s, q, c)
	err = dbdat.dbEdit(Script{})
	if err != nil {
		return "", err
	}

	msg := fmt.Sprintf(
		"__**%s** edited **%s**__\n"+
			"**Added by**: %s\n"+
			"**Version**: %.1f\n"+
			"**Date Added**: %s\n"+
			"**Date Modified**: %s\n\n"+
			"**URL**: [%s](%s) by %s\n"+
			"**(Script will only be avaible for __10 minutes__.)*",
		s.Author.String(), s.Name,
		s.Author.String(),
		s.Version,
		s.DateAdded.Format(time.UnixDate),
		tn.Format(time.UnixDate),
		s.Name, s.URL, s.Author.String(),
	)

	return msg, nil
}

// Delete a script from a library.
func (lib *Library) Delete() (string, error) {

	s := lib.Script
	var err error
	var q = make(map[string]interface{})

	q["$and"] = []bson.M{bson.M{"name": s.Name}, bson.M{"author.name": s.Author.Name}}
	dbdat := DBdataCreate(lib.Database, CollectionScripts, lib.Script, q, nil)
	err = dbdat.dbDelete()
	if err != nil {
		if err == mgo.ErrNotFound {
			return "", fmt.Errorf("script doesn't exist, or you need to specify '--user' flag")
		}
		return "", err
	}

	msg := fmt.Sprintf("**%s** deleted -> **%s**\n  It will be missed...", s.Author.Name, lib.Script.Name)
	return msg, nil
}

func (lib *Library) find(requested bool) (*Script, error) {
	s := lib.Script
	var q = make(map[string]interface{})

	q["$and"] = []bson.M{bson.M{"name": s.Name}, bson.M{"author.name": s.Author.Name}}

	dbdat := DBdataCreate(lib.Database, CollectionScripts, Script{}, q, nil)
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
	dbdat := DBdataCreate(lib.Database, CollectionScripts, Script{}, nil, nil)
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

	var found bool
	var msg = "Current Scripts in Library:\n\nFormat: [User]  [Version]  [Title]\n"
	for n, d := range docs {
		found = true
		msg += fmt.Sprintf("  [%d] %s -> [v%.1f] %s\n", n, d.Author.Name, d.Version, d.Name)
	}
	if !found {
		msg += "No scripts found in library.\n"
	}
	msg += fmt.Sprintf("\nTo request a script, type:\n%s", scriptSyntaxGet)

	return "```" + msg + "```", nil
}

// SetAccessed updates database with new timestamp.
func (lib *Library) setAccessed() error {
	s := lib.Script
	var q = make(map[string]interface{})
	var c = make(map[string]interface{})

	q["$and"] = []bson.M{bson.M{"name": s.Name}, bson.M{"author.name": s.Author.Name}}

	c["$set"] = bson.M{
		"url":          s.URL,
		"dateaccessed": time.Now(),
	}

	dbdat := DBdataCreate(lib.Database, CollectionScripts, s, q, c)
	err := dbdat.dbEdit(Script{})
	if err != nil {
		return err
	}

	return nil
}

// ScriptNew creates a new script object.
func ScriptNew(name, content string, version float32, author UserBasic) *Script {
	tn := time.Now()
	return &Script{
		Name:         name,
		Author:       author,
		Content:      content,
		Length:       len(content),
		Version:      version,
		DateAdded:    tn,
		DateModified: tn,
		DateAccessed: tn,
	}
}

// Print returns a string of information about a script.
func (s *Script) String() string {
	if s == nil {
		return "script information could not be obtained."
	}

	msg := fmt.Sprintf(
		"__Script Name: %s__\n\n"+
			"**Added by**: %s\n"+
			"**Version**: %.1f\n"+
			"**Date Added**: %s\n"+
			"**Date Modified**: %s\n\n"+
			"**URL**: [%s](%s) by %s\n"+
			"**(Script will only be avaible for __10 minutes__.)*",
		s.Name,
		s.Author.String(),
		s.Version,
		s.DateAdded.Format(time.UnixDate),
		s.DateModified.Format(time.UnixDate),
		s.Name, s.URL, s.Author.String(),
	)
	return msg
}

func getFile(filename, url string) (string, error) {

	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var buf = new(bytes.Buffer)
	if _, err := io.Copy(buf, resp.Body); err != nil {
		return "", err
	}

	return buf.String(), nil
}
