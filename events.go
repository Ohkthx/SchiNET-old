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
func (io *IOdat) CoreEvent() error {
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

	if err := fl.Getopt(io.io, nil); err != nil {
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
		io.output = Help(fl, prefix, suffix)
		return nil
	} else if list {
		ev := EventNew("", "", "", io.user, false)
		msg, err := ev.List()
		if err != nil {
			return err
		}
		io.output = msg
		return nil
	}

	if (add || edit || del) && day != "" {
		if ok := io.user.HasPermission(permAdmin); !ok {
			return ErrBadPermissions
		}
		ev := EventNew(comment, day, time, io.user, persist)

		var err error
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
			io.msgEmbed = embedCreator(msg, ColorGreen)
		}
	}

	ev := EventNew("", "", "", io.user, false)
	msg, err := ev.List()
	if err != nil {
		return err
	}
	io.output = msg
	return nil

}

// EventNew creates a new Event object that can be acted on.
func EventNew(desc, day, t string, u *User, persist bool) *Event {

	da, _ := evDayAdd(day)
	hhmm, _ := evHHMM(t)

	now := time.Now()
	ts := time.Date(now.Year(), now.Month(), now.Day()+da, hhmm[0], hhmm[1], 0, 0, now.Location())

	return &Event{
		Server:      u.Server,
		Description: desc,
		Day:         day,
		HHMM:        t,
		Time:        ts,
		Protected:   persist,
		AddedBy:     u.Basic(),
	}
}

// Add stores an Event in the Database.
func (ev *Event) Add() (string, error) {

	dbdat := DBdatCreate(ev.Server, CollectionEvents, ev, nil, nil)
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

	q["$and"] = []bson.M{bson.M{"day": ev.Day}, bson.M{"time": ev.Time}}
	var dbdat = DBdatCreate(ev.Server, CollectionEvents, Event{}, q, nil)
	if err := dbdat.dbGet(Event{}); err != nil {
		if err == mgo.ErrNotFound {
			return "", fmt.Errorf("event not found: %s -> %s", ev.Day, ev.HHMM)
		}
		return "", err
	}

	var e = dbdat.Document.(Event)
	if err := dbdat.dbDeleteID(e.ID); err != nil {
		return "", err
	}

	msg := fmt.Sprintf("%s removed the event on **%s**.", ev.AddedBy.StringPretty(), ev.Day)

	return msg, nil
}

// List events for the local server.
func (ev *Event) List() (string, error) {
	var events []string
	var msg string
	var err error
	var t = time.Now()
	var cnt int

	dbdat := DBdatCreate(ev.Server, CollectionEvents, nil, nil, nil)
	dbdat.dbGetAll(Event{})

	if len(dbdat.Documents) == 0 {
		return "", errors.New("no events scheduled for this server")
	}

	msg = "Upcoming Events:```C\n"
	for n, e := range dbdat.Documents {
		cnt++
		var ev = e.(Event)
		dur := ev.Time.Sub(t)
		hours := int(dur.Hours())

		if hours < -12 && ev.Protected {
			ev.Time, err = evTime(ev.Day, ev.HHMM, true)
			if err != nil {
				return "", err
			}
			dur = ev.Time.Sub(t)
			hours = int(dur.Hours())
			// Update Database here with new time.
			var q = make(map[string]interface{})
			var c = make(map[string]interface{})
			q["_id"] = ev.ID
			c["$set"] = bson.M{"time": ev.Time}
			var dbdat = DBdatCreate(ev.Server, CollectionEvents, ev, q, c)
			err = dbdat.dbEdit(Event{})
			if err != nil {
				return "", err
			}

		} else if hours < -12 {
			// Delete from Database here.
			err := dbdat.dbDeleteID(ev.ID)
			if err != nil {
				return "", err
			}
			continue
		}
		minutes := int(dur.Minutes()) % 60
		events = append(events, ev.Description)

		minT := "minutes"
		hourT := "hours"
		if minutes < 2 && minutes > -2 {
			minT = "minute"
		}
		if hours < 2 && hours > -2 {
			hourT = "hour"
		}

		event := fmt.Sprintf("[%d]  %3d %5s %2d %7s ->  %8s - %s CST\n", n, hours, hourT, minutes, minT, ev.Time.Weekday().String(), ev.Time.Format("15:04"))
		msg += event
	}
	for n, e := range events {
		msg += fmt.Sprintf("\n[%d] -> %s", n, e)
	}

	if cnt == 0 {
		msg += fmt.Sprintf("No events scheduled.\n")
	}
	msg += "```"

	return msg, nil
}

// Converts a weekday to a number.
func evDayAdd(wd string) (int, error) {
	var num, alt, dayAdd int
	now := time.Now()

	switch strings.ToLower(wd) {
	case "sunday":
		num = 0
		alt = 7
	case "monday":
		num = 1
	case "tuesday":
		num = 2
	case "wednesday":
		num = 3
	case "thursday":
		num = 4
	case "friday":
		num = 5
	case "saturday":
		num = 6
	default:
		return 0, ErrBadWeekday
	}

	if num < int(now.Weekday()) {
		dayAdd = alt - int(now.Weekday())
	} else if num > int(now.Weekday()) {
		dayAdd = num - int(now.Weekday())
	} else {
		dayAdd = 0
	}

	return dayAdd, nil
}

// Converts a time string.
func evHHMM(hhmms string) ([2]int, error) {
	var hm [2]int
	var err error

	hhmm := strings.Split(hhmms, ":")
	hm[0], err = strconv.Atoi(hhmm[0])
	if err != nil {
		return hm, ErrBadTime
	}
	hm[1], err = strconv.Atoi(hhmm[1])
	if err != nil {
		return hm, ErrBadTime
	}

	return hm, nil
}

func evTime(wd, hhmms string, rollover bool) (time.Time, error) {
	t := time.Now()
	var da int
	var err error

	if rollover {
		da = 7
	} else {
		da, err = evDayAdd(wd)
		if err != nil {
			return t, err
		}
	}
	hhmm, err := evHHMM(hhmms)
	if err != nil {
		return t, err
	}
	return time.Date(t.Year(), t.Month(), t.Day()+da, hhmm[0], hhmm[1], 0, 0, t.Location()), nil
}
