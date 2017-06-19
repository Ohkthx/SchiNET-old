package main

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"gopkg.in/mgo.v2/bson"
)

// Error constants for errors related to events.
var (
	ErrBadWeekday = errors.New("bad weekday provided")
	ErrBadTime    = errors.New("bad time provided")
	ErrBadArgs    = errors.New("you did not specify enough arguments")
)

func (io *IOdat) miscAddEvent() error {
	var future time.Time
	var now = time.Now()
	var protect bool

	if len(io.io) < 5 {
		return ErrBadArgs
	} else if len(io.io) == 6 {
		if strings.ToLower(io.io[5]) == "true" {
			protect = true
		}
	}

	// Fields:
	// 0 - command, 1 - event, 2 - weekday, 3 - time, 4 - message, 5 - protected.
	da, err := evDayAdd(io.io[2])
	if err != nil {
		return err
	}
	hhmm, err := evHHMM(io.io[3])
	if err != nil {
		return err
	}

	future = time.Date(now.Year(), now.Month(), now.Day()+da, hhmm[0], hhmm[1], 0, 0, now.Location())

	var e = Event{
		Description: io.io[4],
		Day:         io.io[2],
		Time:        future,
		Protected:   protect,
		AddedBy:     io.user,
	}

	dbdat := DBdatCreate(io.guild.Name, CollectionEvents, e, nil, nil)
	err = dbdat.dbInsert()
	if err != nil {
		return err
	}

	msg := fmt.Sprintf("<@%s> added an event: **%s**", io.msg.Author.ID, e.Description)
	io.msgEmbed = embedCreator(msg, ColorGreen)

	return nil
}

func (io *IOdat) miscEditEvent() error {
	return nil
}

func (io *IOdat) miscDelEvent() error {
	if len(io.io) < 4 {
		return ErrBadArgs
	}

	_, err := evDayAdd(io.io[2])
	if err != nil {
		return err
	}

	t, err := io.evTime(io.io[3], false)
	if err != nil {
		return err
	}
	var q = make(map[string]interface{})
	q["$and"] = []bson.M{bson.M{"day": io.io[2]}, bson.M{"time": t}}
	var dbdat = DBdatCreate(io.guild.Name, CollectionEvents, Event{}, q, nil)
	err = dbdat.dbGet(Event{})
	if err != nil {
		return err
	}

	var ev = dbdat.Document.(Event)
	err = dbdat.dbDeleteID(ev.ID)
	if err != nil {
		return err
	}

	msg := fmt.Sprintf("<@%s> removed the event on **%s** at **%s**.", io.user.ID, io.io[2], io.io[3])
	io.msgEmbed = embedCreator(msg, ColorMaroon)

	return nil
}

func (io *IOdat) miscGetEvents() error {
	var events []string
	var msg string
	var err error
	var t = time.Now()

	dbdat := DBdatCreate(io.guild.Name, CollectionEvents, nil, nil, nil)
	dbdat.dbGetAll(Event{})

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
