package main

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/pborman/getopt/v2"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

// Error constants for errors related to events.
var (
	ErrBadWeekday = errors.New("bad weekday provided")
	ErrBadTime    = errors.New("bad time provided")
	ErrBadArgs    = errors.New("you did not specify enough arguments")
)

// Constants for producing helpful text for normal command operations.
const (
	eventSyntaxAdd  = ",event   --add   --day \"Weekday\"   -t \"Time\"   -c \"Comment\"\n"
	eventSyntaxDel  = ",event   --day \"Weekday\"   -t \"Time\"   --remove\n"
	eventSyntaxEdit = ",event   --edit   --day \"Weekday\"   -t \"Time\"   -c \"Comment\"\n"
	eventSyntaxAll  = eventSyntaxAdd + eventSyntaxEdit + eventSyntaxDel
)

// CoreEvent handles all event related commands from input.
func (dat *IOdata) CoreEvent() error {
	var add, del, edit, persist, help, list bool
	var comment, day, time string
	time = "12:00"

	fl := getopt.New()

	fl.FlagLong(&add, "add", 0, "Add an Event")
	fl.FlagLong(&edit, "edit", 0, "Edit an Event")
	fl.FlagLong(&del, "remove", 0, "Delete an Event")
	fl.FlagLong(&persist, "persist", 'p', "Reoccuring Event")
	fl.FlagLong(&help, "help", 'h', "Prints this")
	fl.FlagLong(&list, "list", 'l', "List all Events")
	fl.FlagLong(&day, "day", 'd', "Weekday of Event")
	fl.FlagLong(&time, "time", 't', "Time Occuring [12:00 default]")
	fl.FlagLong(&comment, "comment", 'c', "Event Information/Comment")

	if err := fl.Getopt(dat.io, nil); err != nil {
		return err
	}
	if fl.NArgs() > 0 {
		if err := fl.Getopt(fl.Args(), nil); err != nil {
			return err
		}
	}

	if help {
		prefix := "**Need** __command__,  __day__.\n\n"
		suffix := "\n\nExamples:\n" + eventSyntaxAll
		dat.output = Help(fl, prefix, suffix)
		return nil
	} else if list {
		var err error
		var ev *Event
		if ev, err = EventNew(dat.guild.Name, "", "", "", dat.user, false); err != nil {
			return err
		}

		msg, err := ev.List()
		if err != nil {
			return err
		}

		dat.output = msg
		return nil
	}

	if (add || edit || del) && day != "" {
		// Return if the user does not have the role
		if ok := dat.user.HasRoleType(dat.guildConfig, rolePermissionAdmin); !ok {
			return ErrBadPermissions
		}
		var err error
		var ev *Event
		if ev, err = EventNew(dat.guild.Name, comment, day, time, dat.user, persist); err != nil {
			return err
		}

		var msg string
		switch {
		case add:
			msg, err = ev.Add()
		case edit:
			msg, err = ev.Edit()
		case del:
			msg, err = ev.Delete()
		}
		if err != nil {
			return err
		} else if msg != "" {
			dat.msgEmbed = embedCreator(msg, ColorGreen)
			return nil
		}
	}

	var err error
	var ev *Event
	if ev, err = EventNew(dat.guild.Name, "", "", "", dat.user, false); err != nil {
		return err
	}

	msg, err := ev.List()
	if err != nil {
		return err
	}

	dat.output = msg
	return nil

}

// EventNew creates a new Event object that can be acted on.
func EventNew(database, desc, day, t string, u *User, persist bool) (*Event, error) {

	var err error
	var ts time.Time
	if ts, err = evTime(day, t); err != nil {
		if t != "" {
			return nil, err
		}
	}

	return &Event{
		Server:      database,
		Description: desc,
		Day:         day,
		HHMM:        t,
		Time:        ts,
		Protected:   persist,
		AddedBy:     u.Basic(),
	}, nil
}

// Add stores an Event in the Database.
func (ev *Event) Add() (string, error) {

	dbdat := DBdataCreate(ev.Server, CollectionEvents, ev, nil, nil)
	if err := dbdat.dbInsert(); err != nil {
		return "", err
	}

	msg := fmt.Sprintf("%s added an event: **%s**", ev.AddedBy.StringPretty(), ev.Description)

	return msg, nil
}

// Edit modifies an event inside the database.
func (ev *Event) Edit() (string, error) {
	return "Command not functional yet.", nil
}

// Delete removes an event from the database.
func (ev *Event) Delete() (string, error) {

	var q = make(map[string]interface{})

	// Create the query, get the Event.
	q["$and"] = []bson.M{bson.M{"day": ev.Day}, bson.M{"hhmm": ev.HHMM}}
	var dbdat = DBdataCreate(ev.Server, CollectionEvents, Event{}, q, nil)
	if err := dbdat.dbGet(Event{}); err != nil {
		if err == mgo.ErrNotFound {
			return "", fmt.Errorf("event not found: %s -> %s", ev.Day, ev.HHMM)
		}
		return "", err
	}

	// Convert and remove the Event.
	var e = dbdat.Document.(Event)
	if err := dbdat.dbDeleteID(e.ID); err != nil {
		return "", err
	}

	msg := fmt.Sprintf("%s removed the event on **%s**.", ev.AddedBy.StringPretty(), ev.Day)

	return msg, nil
}

// List events for the local server.
func (ev *Event) List() (string, error) {

	var msg string
	var err error
	var t = time.Now()
	var cnt int

	dbdat := DBdataCreate(ev.Server, CollectionEvents, nil, nil, nil)
	dbdat.dbGetAll(Event{})

	if len(dbdat.Documents) == 0 {
		return "", errors.New("no events scheduled for this server")
	}

	var events []EventSmall
	msg = "Upcoming Events:```C\n"
	for _, e := range dbdat.Documents {
		cnt++
		var ev = e.(Event)
		dur := ev.Time.Sub(t)
		hours := int(dur.Hours())

		// If the event is 23 hours old and protected-
		if hours < -23 && ev.Protected {
			ev.Time = evNextTime(ev.Time)
			dur = ev.Time.Sub(t)
			hours = int(dur.Hours())

			// Update Database here with new time.
			var q = make(map[string]interface{})
			var c = make(map[string]interface{})
			q["_id"] = ev.ID
			c["$set"] = bson.M{"time": ev.Time}
			var dbdat = DBdataCreate(ev.Server, CollectionEvents, ev, q, c)
			err = dbdat.dbEdit(Event{})
			if err != nil {
				return "", err
			}

		} else if hours < -23 {
			// Delete the event if it is not protected and near a full day old.
			// Delete from Database here.
			err := dbdat.dbDeleteID(ev.ID)
			if err != nil {
				return "", err
			}
			continue
		}

		minutes := int(dur.Minutes()) % 60
		// If hours is less than 0, it will display: -1 hour, 30 minutes instead of -1hour, -30minutes.
		if hours < 0 {
			if minutes < 0 {
				minutes = 60 + minutes
			}
		}

		// Add it to our list of events to process for sorting.
		events = append(events, EventSmall{Hours: hours, Minutes: minutes, Time: ev.Time, Description: ev.Description})
	}

	for i := 0; i < len(events); i++ {
		eventSort(events)
	}

	for n, e := range events {
		minT := "minutes"
		hourT := "hours"
		if e.Minutes < 2 && e.Minutes > -2 {
			minT = "minute"
		}
		if e.Hours < 2 && e.Hours > -2 {
			hourT = "hour"
		}

		event := fmt.Sprintf("[%d]  %4d %5s %3d %7s ->  %9s - %s CST\n", n, e.Hours, hourT, e.Minutes, minT, e.Time.Weekday().String(), e.Time.Format("15:04"))
		msg += event
	}

	for n, e := range events {
		msg += fmt.Sprintf("\n[%d] -> %s", n, e.Description)
	}

	if cnt == 0 {
		msg += fmt.Sprintf("No events scheduled.\n")
	}
	msg += "```"

	return msg, nil
}

// Converts a time string.
func evHHMM(hhmms string) ([2]int, error) {
	var hm [2]int
	var err error

	hhmm := strings.Split(hhmms, ":")
	hm[0], err = strconv.Atoi(hhmm[0])
	if err != nil {
		return hm, ErrBadTime
	} else if hm[0] > 24 || hm[0] < 0 {
		// Verify it is a good hour provided.
		return hm, ErrBadTime
	}

	hm[1], err = strconv.Atoi(hhmm[1])
	if err != nil {
		return hm, ErrBadTime
	} else if hm[1] > 60 || hm[1] < 0 {
		// Verify it is a good minute provided.
		return hm, ErrBadTime
	}

	return hm, nil
}

// evTime converts a potentially expiring time and updates it to the next.
func evTime(weekday, hhmms string) (time.Time, error) {
	now := time.Now()
	var err error

	hhmm, err := evHHMM(hhmms)
	if err != nil {
		return now, err
	}

	// Get the next occurence (weekday) this event will happen.
	var future = time.Date(now.Year(), now.Month(), now.Day(), hhmm[0], hhmm[1], 0, 0, now.Location())
	var date = now.Day()
	for strings.ToLower(future.Weekday().String()) != strings.ToLower(weekday) {
		date++
		future = time.Date(now.Year(), now.Month(), date, hhmm[0], hhmm[1], 0, 0, now.Location())
	}

	return future, nil
}

// evNextTime adds 7 days to the current day.
func evNextTime(ts time.Time) time.Time {
	now := time.Now()
	if ts.Before(now) {
		return ts.AddDate(0, 0, 7)
	}
	return ts
}

// eventSort just sorts events based on time.
func eventSort(events []EventSmall) {
	var firstIndex = 0
	var secondIndex = 1

	for secondIndex < len(events) {
		var firstEvent = events[firstIndex]
		var secondEvent = events[secondIndex]

		if firstEvent.Time.After(secondEvent.Time) {
			events[firstIndex] = secondEvent
			events[secondIndex] = firstEvent
		}

		firstIndex++
		secondIndex++
	}
}
