package main

import (
	"errors"
	"strings"

	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

// Error constants for database issues.
var (
	ErrNilInterface = errors.New("nil interface provided")
	ErrNilQuery     = errors.New("nil interface provided")
	ErrNilChange    = errors.New("nil change provided")
	ErrUnknownType  = errors.New("unknown type")
)

// Collection constants
const (
	CollectionEvents    = "events"
	CollectionUsers     = "users"
	CollectionMessages  = "messages"
	CollectionBlacklist = "blacklist"
)

// DBdatCreate creates a database object used to get exchange information with mongodb
func DBdatCreate(db, coll string, doc interface{}, q bson.M, c bson.M) *DBdat {
	return &DBdat{Database: db, Collection: coll, Document: doc, Query: q, Change: c}
}

func (d *DBdat) dbInsert() error {

	if d.Document == nil {
		return ErrNilInterface
	}

	mdb, err := mgo.Dial(envDBUrl)
	if err != nil {
		return err
	}
	defer mdb.Close()

	//mdb.SetMode(mgo.Monotonic, true)

	c := mdb.DB(d.Database).C(d.Collection)
	err = c.Insert(d.Document)
	if err != nil {
		return err
	}
	return nil
}

func (d *DBdat) dbEdit(i interface{}) error {
	if d.Query == nil {
		return ErrNilQuery
	} else if d.Change == nil {
		return ErrNilChange
	}

	change := mgo.Change{
		Update:    d.Change,
		ReturnNew: true,
	}

	mdb, err := mgo.Dial(envDBUrl)
	if err != nil {
		return err
	}
	defer mdb.Close()

	//mdb.SetMode(mgo.Monotonic, true)

	c := mdb.DB(d.Database).C(d.Collection)
	_, err = c.Find(d.Query).Apply(change, &d.Document)
	if err != nil {
		return err
	}

	return nil
}

func (d *DBdat) dbDelete(id bson.ObjectId) error {
	mdb, err := mgo.Dial(envDBUrl)
	if err != nil {
		return err
	}
	defer mdb.Close()

	//mdb.SetMode(mgo.Monotonic, true)

	c := mdb.DB(d.Database).C(d.Collection)
	err = c.RemoveId(id)
	if err != nil {
		return err
	}

	return nil
}

func (d *DBdat) dbGet(i interface{}) error {
	var unk interface{}
	if d.Query == nil {
		return ErrNilInterface
	}

	mdb, err := mgo.Dial(envDBUrl)
	if err != nil {
		return err
	}
	defer mdb.Close()

	c := mdb.DB(d.Database).C(d.Collection)
	err = c.Find(d.Query).One(&unk)
	if err != nil {
		return err
	}

	d.Document = handlerForInterface(i, unk)

	return nil
}

func (d *DBdat) dbGetAll(i interface{}) error {
	var unk []interface{}
	mdb, err := mgo.Dial(envDBUrl)
	if err != nil {
		return err
	}
	defer mdb.Close()

	c := mdb.DB(d.Database).C(d.Collection)
	err = c.Find(nil).All(&unk)
	if err != nil {
		return err
	}

	for _, p := range unk {
		h := handlerForInterface(i, p)
		d.Documents = append(d.Documents, h)
	}

	return nil
}

func (io *IOdat) dbCore() (err error) {
	if Bot != nil {
		// Required for storing information in the correct database.
		io.guild = Bot.GetGuild(Bot.GetChannel(io.msg.ChannelID).GuildID)
	}

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
		}
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

func handlerForInterface(handler interface{}, i interface{}) interface{} {
	switch handler.(type) {
	case Event:
		var e Event
		byt, _ := bson.Marshal(i)
		bson.Unmarshal(byt, &e)
		return e
	}
	return nil
}
