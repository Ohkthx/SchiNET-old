package main

import (
	"errors"
	"flag"
	"fmt"
	"strconv"
	"strings"
	"time"

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
	eventSyntaxAdd  = ",event   -add   -d \"Weekday\"   -t \"Time\"   -c \"Comment\"\n"
	eventSyntaxDel  = ",event   -d \"Weekday\"   -t \"Time\"   -remove\n"
	eventSyntaxEdit = ",event   -edit   -d \"Weekday\"   -t \"Time\"   -c \"Comment\"\n"
	eventSyntaxAll  = eventSyntaxAdd + eventSyntaxEdit + eventSyntaxDel
)

// Events handles all event related commands from input.
func (io *IOdat) Events(database, id string, command []string) (string, error) {
	var add, del, edit, persist, help, list bool
	var comment, day, time string
	fl := flag.NewFlagSet("event", flag.ContinueOnError)
	fl.BoolVar(&add, "add", false, "Add an Event")
	fl.BoolVar(&edit, "edit", false, "Edit an Event")
	fl.BoolVar(&del, "remove", false, "Delete an Event")
	fl.BoolVar(&persist, "persist", false, "Reoccuring Event")
	fl.BoolVar(&help, "help", false, "Prints this")
	fl.BoolVar(&list, "list", false, "List all Events")
	fl.StringVar(&day, "d", "", "Weekday of Event")
	fl.StringVar(&time, "t", "12:00", "Time Occuring [12:00 default]")
	fl.StringVar(&comment, "c", "", "Event Information/Comment")
	if err := fl.Parse(command[1:]); err != nil {
		return "", err
	}

	if help {
		prefix := "**Need** __command__,  __day__.\n\n"
		suffix := "\n\nExamples:\n" + eventSyntaxAll
		return Help(fl, prefix, suffix), nil
	} else if list {
		return "", io.GetEvents()
	}

	if (add || edit || del) && day != "" {
		if ok := io.user.HasPermission(permAdmin); !ok {
			return "", ErrBadPermissions
		}
		ev, err := EventNew(comment, day, time, io.user, persist)
		if err != nil {
			return "", err
		}

		switch {
		case add:
			return ev.Add(database)
		case edit:
			return ev.Edit(database)
		case del:
			return ev.Delete(database)
		}
	}

	return "", io.GetEvents()
}

// EventNew creates a new Event object that can be acted on.
func EventNew(desc, day, t string, u *User, persist bool) (*Event, error) {

	da, err := evDayAdd(day)
	if err != nil {
		return nil, err
	}
	hhmm, err := evHHMM(t)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	ts := time.Date(now.Year(), now.Month(), now.Day()+da, hhmm[0], hhmm[1], 0, 0, now.Location())

	return &Event{
		Description: desc,
		Day:         day,
		HHMM:        t,
		Time:        ts,
		Protected:   persist,
		AddedBy:     u,
	}, nil
}

// Add stores an Event in the Database.
func (ev *Event) Add(database string) (string, error) {

	dbdat := DBdatCreate(database, CollectionEvents, ev, nil, nil)
	if err := dbdat.dbInsert(); err != nil {
		return "", err
	}

	msg := fmt.Sprintf("%s added an event: **%s**", ev.AddedBy.StringPretty(), ev.Description)

	return msg, nil
}

// Edit modifies an event inside the database.
func (ev *Event) Edit(database string) (string, error) {
	return "Command not functional yet.", nil
}

// Delete removes an event from the database.
func (ev *Event) Delete(database string) (string, error) {

	var q = make(map[string]interface{})

	q["$and"] = []bson.M{bson.M{"day": ev.Day}, bson.M{"time": ev.Time}}
	var dbdat = DBdatCreate(database, CollectionEvents, Event{}, q, nil)
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

// GetEvents for the local server.
func (io *IOdat) GetEvents() error {
	var events []string
	var msg string
	var err error
	var t = time.Now()

	dbdat := DBdatCreate(io.guild.Name, CollectionEvents, nil, nil, nil)
	dbdat.dbGetAll(Event{})

	if len(dbdat.Documents) == 0 {
		io.msgEmbed = embedCreator("No events found", ColorMaroon)
		return nil
	}

	msg = "Upcoming Events:```C\n"
	for n, e := range dbdat.Documents {
		var ev = e.(Event)
		dur := ev.Time.Sub(t)
		hours := int(dur.Hours())

		if hours < -12 && ev.Protected {
			ev.Time, err = io.evTime(io.io[3], true)
			if err != nil {
				return err
			}
			dur = ev.Time.Sub(t)
			hours = int(dur.Hours())
			// Update Database here with new time.
			var q = make(map[string]interface{})
			var c = q
			q["_id"] = ev.ID
			c["$set"] = bson.M{"time": ev.Time}
			var dbdat = DBdatCreate(io.guild.Name, CollectionEvents, ev, q, c)
			err = dbdat.dbEdit(Event{})
			if err != nil {
				return err
			}

		} else if hours < -12 {
			// Delete from Database here.
			err := dbdat.dbDeleteID(ev.ID)
			if err != nil {
				return err
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
	msg += "```"
	//io.output = msg
	io.msgEmbed = embedCreator(msg, ColorBlue)

	return nil
}

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

func (io *IOdat) evTime(hhmms string, rollover bool) (time.Time, error) {
	t := time.Now()
	var da int
	var err error

	if rollover {
		da = 7
	} else {
		da, err = evDayAdd(io.io[2])
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
