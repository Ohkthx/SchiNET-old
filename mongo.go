package main

import (
	"errors"
	"strings"

	mgo "gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

// Error constants for database issues.
var (
	ErrNilInterface = errors.New("nil interface provided")
	ErrNilQuery     = errors.New("nil interface provided")
	ErrNilChange    = errors.New("nil change provided")
	ErrUnknownType  = errors.New("unknown type")
	ErrBadInterface = errors.New("bad interface, unknown in switch")
	ErrNoDocument   = errors.New("document not found")
	//CollectionMessages = func(c string) string { return "messages." + c }
)

// Collection and Database constants
const (
	Database            = "discord"
	CollectionEvents    = "events"
	CollectionUsers     = "users"
	CollectionBlacklist = "blacklist"
	CollectionGamble    = "gamble"
	CollectionConfigs   = "config"
	CollectionScripts   = "library"
	CollectionMessages  = "messages"
	CollectionAlias     = "aliases"
	CollectionAlliances = "alliances"
	CollectionChannels  = "channels"
	CollectionTickets   = "tickets"
	CollectionConfig    = "config"
)

// DBdata passes information as to what to store into a database.
type DBdata struct {
	Handler    *mgo.Session
	Database   string
	Collection string
	Document   interface{}
	Documents  []interface{}
	Query      bson.M
	Change     bson.M
}

//DBHandler Stores a MongoDB connection.
type DBHandler struct {
	*mgo.Session
}

// DBdataCreate creates a database object used to get exchange information with mongodb
func DBdataCreate(db, coll string, doc interface{}, q, c bson.M) *DBdata {
	return &DBdata{Handler: Mgo, Database: dbSafe(db), Collection: coll, Document: doc, Query: q, Change: c}
}

func (dat *DBdata) dbInsert() error {
	var err error
	if dat.Document == nil {
		return ErrNilInterface
	}

	mdb := dat.Handler

	//mdb.SetMode(mgo.Monotonic, true)

	c := mdb.DB(dat.Database).C(dat.Collection)
	err = c.Insert(dat.Document)
	if err != nil {
		return err
	}
	return nil
}

func (dat *DBdata) dbEdit(i interface{}) error {
	var err error
	if dat.Query == nil {
		return ErrNilQuery
	} else if dat.Change == nil {
		return ErrNilChange
	}

	change := mgo.Change{
		Update:    dat.Change,
		ReturnNew: true,
	}

	mdb := dat.Handler
	//mdb.SetMode(mgo.Monotonic, true)

	c := mdb.DB(dat.Database).C(dat.Collection)
	_, err = c.Find(dat.Query).Apply(change, &dat.Document)
	if err != nil {
		return err
	}

	return nil
}

func (dat *DBdata) dbDeleteID(id bson.ObjectId) error {
	var err error

	mdb := dat.Handler

	//mdb.SetMode(mgo.Monotonic, true)

	c := mdb.DB(dat.Database).C(dat.Collection)
	err = c.RemoveId(id)
	if err != nil {
		return err
	}

	return nil
}

func (dat *DBdata) dbDelete() error {
	var err error

	if dat.Query == nil {
		return ErrNilQuery
	}

	mdb := dat.Handler

	//mdb.SetMode(mgo.Monotonic, true)

	c := mdb.DB(dat.Database).C(dat.Collection)
	err = c.Remove(dat.Query)
	if err != nil {
		return err
	}

	return nil
}

func (dat *DBdata) dbGet(i interface{}) error {
	var unk interface{}
	var err error
	if dat.Query == nil {
		return ErrNilInterface
	}

	mdb := dat.Handler

	c := mdb.DB(dat.Database).C(dat.Collection)
	err = c.Find(dat.Query).One(&unk)
	if err != nil {
		return err
	}

	if unk == nil {
		return mgo.ErrNotFound
	}

	dat.Document, err = handlerForInterface(i, unk)
	if err != nil {
		return err
	}

	return nil
}

func (dat *DBdata) dbGetWithSkip(i interface{}, amount int) error {
	var unk interface{}
	var err error

	mdb := dat.Handler

	c := mdb.DB(dat.Database).C(dat.Collection)
	err = c.Find(dat.Query).Skip(amount).One(&unk)
	if err != nil {
		return err
	}

	if unk == nil {
		return mgo.ErrNotFound
	}

	dat.Document, err = handlerForInterface(i, unk)
	if err != nil {
		return err
	}

	return nil

}

func (dat *DBdata) dbGetWithLimit(i interface{}, sort []string, amount int) error {
	var unk []interface{}
	var err error

	mdb := dat.Handler

	c := mdb.DB(dat.Database).C(dat.Collection)
	err = c.Find(dat.Query).Sort(sort...).Limit(amount).All(&unk)
	if err != nil {
		return err
	}

	for _, p := range unk {
		h, err := handlerForInterface(i, p)
		if err != nil {
			return err
		}
		dat.Documents = append(dat.Documents, h)
	}

	return nil
}

func (dat *DBdata) dbGetAll(i interface{}) error {
	var unk []interface{}
	var err error

	mdb := dat.Handler

	c := mdb.DB(dat.Database).C(dat.Collection)
	err = c.Find(nil).All(&unk)
	if err != nil {
		return err
	}

	for _, p := range unk {
		h, err := handlerForInterface(i, p)
		if err != nil {
			return err
		}
		dat.Documents = append(dat.Documents, h)
	}

	return nil
}

/*
// Potentially will replace the current find.One() method of getting documents.
// Awaiting additional benchmarking.
func (d *DBdat) dbGetLimit(i interface{}, count int) error {
	var unk interface{}
	var err error
	if d.Query == nil {
		return ErrNilInterface
	}

	mdb := d.Handler

	c := mdb.DB(d.Database).C(d.Collection)
	err = c.Find(d.Query).Limit(count)
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
*/

func (dat *DBdata) dbCount() (int, error) {
	var err error
	var n int
	mdb := dat.Handler

	c := mdb.DB(dat.Database).C(dat.Collection)
	if n, err = c.Count(); err != nil {
		return -1, err
	}

	return n, nil
}

func (dat *DBdata) dbExists() error {
	var err error
	var count int

	mdb := dat.Handler
	c := mdb.DB(dat.Database).C(dat.Collection)

	count, err = c.Find(dat.Query).Count()
	if err != nil {
		return err
	}

	if count == 0 {
		return ErrNoDocument
	}

	return nil

}

// CoreDatabase will control adding and removing user defined commands.
func (dat *IOdata) CoreDatabase() (err error) {

	switch dat.io[0] {
	case "add":
	case "edit":
	case "del":
	}

	return nil
}

func handlerForInterface(handler interface{}, i interface{}) (interface{}, error) {
	byt, _ := bson.Marshal(i)
	switch handler.(type) {
	case Alias:
		var a Alias
		bson.Unmarshal(byt, &a)
		return a, nil
	case GuildConfig:
		var g GuildConfig
		bson.Unmarshal(byt, &g)
		return g, nil
	case Event:
		var e Event
		bson.Unmarshal(byt, &e)
		return e, nil
	case User:
		var u User
		bson.Unmarshal(byt, &u)
		return u, nil
	case Script:
		var s Script
		bson.Unmarshal(byt, &s)
		return s, nil
	case Message:
		var m Message
		bson.Unmarshal(byt, &m)
		return m, nil
	case Alliance:
		var a Alliance
		bson.Unmarshal(byt, &a)
		return a, nil
	case ChannelInfo:
		var c ChannelInfo
		bson.Unmarshal(byt, &c)
		return c, nil
	case Ticket:
		var t Ticket
		bson.Unmarshal(byt, &t)
		return t, nil
	default:
		return nil, ErrBadInterface
	}
}

func dbSafe(name string) string {
	t := strings.FieldsFunc(name, idSplit)
	return strings.Join(t, "_")
}
