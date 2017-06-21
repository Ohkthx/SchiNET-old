package main

import (
	"errors"
	"strings"

	"github.com/bwmarrin/discordgo"

	mgo "gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

// Error constants for database issues.
var (
	ErrNilInterface    = errors.New("nil interface provided")
	ErrNilQuery        = errors.New("nil interface provided")
	ErrNilChange       = errors.New("nil change provided")
	ErrUnknownType     = errors.New("unknown type")
	ErrBadInterface    = errors.New("bad interface, unknown in switch")
	CollectionMessages = func(c string) string { return "messages." + c }
)

// Collection constants
const (
	CollectionEvents    = "events"
	CollectionUsers     = "users"
	CollectionBlacklist = "blacklist"
	CollectionGamble    = "gamble"
	CollectionConfigs   = "config"
	CollectionScripts   = "library"
)

// DBdat passes information as to what to store into a database.
type DBdat struct {
	Handler    *mgo.Session
	Database   string
	Collection string
	Document   interface{}
	Documents  []interface{}
	Query      bson.M
	Change     bson.M
}

// DBMsg stores information on messages last processed.
type DBMsg struct {
	ID      string
	MTotal  int
	MIDr    string // Message ID of most recent.
	MIDf    string // Message ID of first message.
	Content string
}

//DBHandler Stores a MongoDB connection.
type DBHandler struct {
	*mgo.Session
}

// DBdatCreate creates a database object used to get exchange information with mongodb
func DBdatCreate(db, coll string, doc interface{}, q bson.M, c bson.M) *DBdat {
	return &DBdat{Handler: Mgo, Database: dbSafe(db), Collection: coll, Document: doc, Query: q, Change: c}
}

func (d *DBdat) dbInsert() error {
	var err error
	if d.Document == nil {
		return ErrNilInterface
	}

	mdb := d.Handler

	//mdb.SetMode(mgo.Monotonic, true)

	c := mdb.DB(d.Database).C(d.Collection)
	err = c.Insert(d.Document)
	if err != nil {
		return err
	}
	return nil
}

func (d *DBdat) dbEdit(i interface{}) error {
	var err error
	if d.Query == nil {
		return ErrNilQuery
	} else if d.Change == nil {
		return ErrNilChange
	}

	change := mgo.Change{
		Update:    d.Change,
		ReturnNew: true,
	}

	mdb := d.Handler
	//mdb.SetMode(mgo.Monotonic, true)

	c := mdb.DB(d.Database).C(d.Collection)
	_, err = c.Find(d.Query).Apply(change, &d.Document)
	if err != nil {
		return err
	}

	return nil
}

func (d *DBdat) dbDeleteID(id bson.ObjectId) error {
	var err error

	mdb := d.Handler

	//mdb.SetMode(mgo.Monotonic, true)

	c := mdb.DB(d.Database).C(d.Collection)
	err = c.RemoveId(id)
	if err != nil {
		return err
	}

	return nil
}

func (d *DBdat) dbDelete() error {
	var err error

	if d.Query == nil {
		return ErrNilQuery
	}

	mdb := d.Handler

	//mdb.SetMode(mgo.Monotonic, true)

	c := mdb.DB(d.Database).C(d.Collection)
	err = c.Remove(d.Query)
	if err != nil {
		return err
	}

	return nil
}

func (d *DBdat) dbGet(i interface{}) error {
	var unk interface{}
	var err error
	if d.Query == nil {
		return ErrNilInterface
	}

	mdb := d.Handler

	c := mdb.DB(d.Database).C(d.Collection)
	err = c.Find(d.Query).One(&unk)
	if err != nil {
		return err
	}

	if unk == nil {
		return mgo.ErrNotFound
	}
	d.Document, err = handlerForInterface(i, unk)
	if err != nil {
		return err
	}

	return nil
}

func (d *DBdat) dbGetAll(i interface{}) error {
	var unk []interface{}
	var err error

	mdb := d.Handler

	c := mdb.DB(d.Database).C(d.Collection)
	err = c.Find(nil).All(&unk)
	if err != nil {
		return err
	}

	for _, p := range unk {
		h, err := handlerForInterface(i, p)
		if err != nil {
			return err
		}
		d.Documents = append(d.Documents, h)
	}

	return nil
}

func (io *IOdat) dbCore() (err error) {
	var s string
	if len(io.io) > 1 {
		switch strings.ToLower(io.io[1]) {
		case "event", "events":
			switch strings.ToLower(io.io[0]) {
			case "add":
				err = io.miscAddEvent()
			case "edit":
				err = io.miscEditEvent()
			case "del":
				err = io.miscDelEvent()
			}
			return
		case "script", "scripts":
			s, err = scriptCore(io.guild.Name, io.msg.Author, io.io, io.help)
			io.msgEmbed = embedCreator(s, ColorGreen)
		}
	}

	if err != nil {
		return err
	} else if io.msgEmbed != nil {
		return nil
	}

	switch io.io[0] {
	case "event":
		err = io.miscGetEvents()
	case "add":
		dbdat := DBdatCreate(io.guild.Name, "commands", Command{}, nil, nil)
		err = dbdat.dbInsert()
	case "edit":
	case "del":
	}

	return nil
}

func handlerForInterface(handler interface{}, i interface{}) (interface{}, error) {
	byt, _ := bson.Marshal(i)
	switch handler.(type) {
	case Event:
		var e Event
		bson.Unmarshal(byt, &e)
		return e, nil
	case DBMsg:
		var d DBMsg
		bson.Unmarshal(byt, &d)
		return d, nil
	case User:
		var u User
		bson.Unmarshal(byt, &u)
		return u, nil
	case Script:
		var s Script
		bson.Unmarshal(byt, &s)
		return s, nil
	case Ban:
		var b Ban
		bson.Unmarshal(byt, &b)
		return b, nil
	case discordgo.Message:
		var m discordgo.Message
		bson.Unmarshal(byt, &m)
		return m, nil
	default:
		return nil, ErrBadInterface
	}
}

func dbSafe(name string) string {
	t := strings.FieldsFunc(name, idSplit)
	return strings.Join(t, "_")
}
