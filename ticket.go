package main

import (
	"errors"
	"fmt"
	"time"

	mgo "gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"

	"github.com/pborman/getopt/v2"
)

// Ticket object
type Ticket struct {
	ID         string `bson:"_id,omitempty"`
	TicketID   int
	Server     string   // Server/Database of the ticket.
	Title      string   // General idea what ticket is about.
	Comment    string   // Larger description.
	Notes      []string // Note from admin/developer regarding ticket.
	Open       bool     // Status of the ticket.
	Removed    bool     // If the ticket has been flagged to be removed.
	AddedBy    UserBasic
	ClosedBy   UserBasic
	DateAdded  time.Time
	DateClosed time.Time
}

func ticketNew(server, title, comment string, status bool, tID int, addedBy *User) Ticket {
	return Ticket{
		TicketID:  tID,
		Server:    server,
		Title:     title,
		Comment:   comment,
		Open:      status,
		AddedBy:   addedBy.Basic(),
		DateAdded: time.Now(),
	}
}

// CoreTickets handles the ticketing system.
func (dat *IOdata) CoreTickets() error {

	var help, list, add, remove, close, update, get bool
	var title, comment, note string
	var tID int

	close = false
	tID = -1

	fl := getopt.New()

	// Generics
	fl.FlagLong(&title, "title", 't', "Title of the ticket")
	fl.FlagLong(&comment, "comment", 'c', "Comment for issue")
	fl.FlagLong(&note, "note", 'n', "Note from Admin/Developer")
	fl.FlagLong(&help, "help", 'h', "This message")
	fl.FlagLong(&list, "list", 0, "List all open tickets")
	fl.FlagLong(&add, "add", 0, "Add a new ticket")
	fl.FlagLong(&update, "update", 0, "Update a tickets title, comment or note")
	fl.FlagLong(&remove, "remove", 0, "Remove an existing ticket (Used for spam)")
	fl.FlagLong(&close, "close", 0, "Close a resolved ticket")
	fl.FlagLong(&get, "get", 0, "Get a Ticket based on ID")
	fl.FlagLong(&tID, "id", 0, "Ticket ID to modify")

	if err := fl.Getopt(dat.io, nil); err != nil {
		return err
	}
	if fl.NArgs() > 0 {
		if err := fl.Getopt(fl.Args(), nil); err != nil {
			return err
		}
	}

	t := ticketNew(dat.guild.Name, title, comment, close, tID, dat.user)

	switch {
	case get:
		if tID < 0 {
			return errors.New("bad Ticket ID (--id) supplied")
		}
		if err := t.Get(tID); err != nil {
			return err
		}
		dat.msgEmbed = embedCreator(t.String(), ColorYellow)
	case add:
		if t.Title == "" || t.Comment == "" {
			return errors.New("need a title and/or a comment")
		}
		t.Open = true
		if err := t.Update(); err != nil {
			return err
		}

		dat.msgEmbed = embedCreator("Ticket created.", ColorGreen)

	case update:
		if tID < 0 {
			return errors.New("bad Ticket ID (--id) supplied")
		}
		if err := t.Get(tID); err != nil {
			return err
		}

		if t.AddedBy.ID != dat.user.ID || dat.user.HasPermissionGTE(t.Server, permAdmin) {
			return ErrBadPermissions
		}

		if title != "" {
			t.Title = title
		}
		if comment != "" {
			t.Comment = comment
		}
		if note != "" {
			t.Notes = append(t.Notes, note)
		}

		if err := t.Update(); err != nil {
			return err
		}

		dat.msgEmbed = embedCreator("Ticket updated.", ColorGreen)

	case remove || close:
		if tID < 0 {
			return errors.New("bad Ticket ID (--id) supplied")
		}
		if err := t.Get(tID); err != nil {
			return err
		}
		if t.AddedBy.ID != dat.user.ID || dat.user.HasPermissionGTE(t.Server, permAdmin) {
			return ErrBadPermissions
		}
		if note == "" {
			return errors.New("need to specify a note (-n) for closing or removing")
		}
		var text = "Ticket successfully closed."
		t.Notes = append(t.Notes, note)
		t.Open = false
		t.ClosedBy = dat.user.Basic()
		t.DateClosed = time.Now()
		if remove {
			t.Removed = true
			text = "ticket successfully removed."
		}
		if err := t.Update(); err != nil {
			return err
		}
		dat.msgEmbed = embedCreator(text, ColorGreen)
	case list:
		var err error
		dat.output, err = ticketList(t.Server)
		if err != nil {
			return err
		}
	case help:
		fallthrough
	default:
		dat.output = Help(fl, "", "")
	}

	return nil
}

// Get a ticket from the database.
func (t *Ticket) Get(tID int) error {
	var q = make(map[string]interface{})

	q["ticketid"] = tID

	dbdat := DBdataCreate(t.Server, CollectionTickets, Ticket{}, q, nil)
	err := dbdat.dbGet(Ticket{})
	if err != nil {
		return err
	}

	var ticket = Ticket{}
	ticket = dbdat.Document.(Ticket)
	*t = ticket

	return nil
}

// Update a tickets object in the database.
func (t *Ticket) Update() error {
	var err error

	// Check if TicketID was supplied
	if t.TicketID < 0 {
		var n int
		var err error
		dbdat := DBdataCreate(t.Server, CollectionTickets, t, nil, nil)
		if n, err = dbdat.dbCount(); err != nil {
			return err
		}
		t.TicketID = n
	}

	var q = make(map[string]interface{})
	var c = make(map[string]interface{})

	q["ticketid"] = t.TicketID
	c["$set"] = bson.M{
		"server":     t.Server,
		"title":      t.Title,
		"comment":    t.Comment,
		"notes":      t.Notes,
		"open":       t.Open,
		"removed":    t.Removed,
		"addedby":    t.AddedBy,
		"closedby":   t.ClosedBy,
		"dateadded":  t.DateAdded,
		"dateclosed": t.DateClosed,
	}

	dbdat := DBdataCreate(t.Server, CollectionTickets, t, q, c)
	err = dbdat.dbEdit(Ticket{})
	if err != nil {
		if err == mgo.ErrNotFound {
			// Add to DB since it doesn't exist.
			var n int
			var err error
			if n, err = dbdat.dbCount(); err != nil {
				return err
			}
			t.TicketID = n
			if err := dbdat.dbInsert(); err != nil {
				return err
			}
			return nil
		}
		return err
	}

	return nil
}

// String displays ticket information
func (t Ticket) String() string {
	var notes, status string
	for n, s := range t.Notes {
		notes += fmt.Sprintf(" %d) %s\n", n+1, s)
	}

	status = "Closed"
	if t.Open {
		status = "Open"
	} else if t.Removed {
		status = "Removed"
	}

	text := fmt.Sprintf(
		"__**Ticket ID**: %d__\n"+
			"**Status**: %s\n"+
			"**Title**: %s\n"+
			"**Comment**: %s\n"+
			"**Notes**:\n%s\n\n"+
			"**Added By**: %s\n"+
			"**Date Added**: %s\n\n",
		t.TicketID,
		status,
		t.Title,
		t.Comment,
		notes,
		t.AddedBy,
		t.DateAdded,
	)

	if !t.Open {
		text += fmt.Sprintf(
			"**Closed By**: %s\n"+
				"**Date Closed**: %s",
			t.ClosedBy,
			t.DateClosed,
		)
	}

	return text
}

// List all of the tickets in the database.
func ticketList(server string) (string, error) {
	db := DBdataCreate(server, CollectionTickets, Ticket{}, nil, nil)
	if err := db.dbGetAll(Ticket{}); err != nil {
		return "", err
	}

	var msg = "```List of Tickets:\n\nFormat: [ID]:  [Status]  [Title]\n"
	if len(db.Documents) == 0 {
		return "There are no tickets.", nil
	}

	var t Ticket
	for _, ticket := range db.Documents {
		t = ticket.(Ticket)
		if t.Removed {
			msg += fmt.Sprintf("  %d: [%s] %s\n", t.TicketID, "Closed", "Removed")
		} else {
			var status, title string
			if t.Open {
				status = "Open"
			} else {
				status = "Closed"
			}
			title = t.Title
			if len(t.Title) > 37 {
				title = t.Title[0:37]
				title += "..."
			}
			msg += fmt.Sprintf("  %d: [%s] %s\n", t.TicketID, status, title)
		}
	}
	msg += fmt.Sprintf("```For more information on a ticket, use:\n `,ticket  --get  --id [id here]`")
	return msg, nil
}
