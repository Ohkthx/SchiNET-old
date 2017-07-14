package main

import (
	"bytes"
	"errors"
	"fmt"
	"math/rand"
	"sort"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/d0x1p2/generate"
	"github.com/d0x1p2/go_pastebin"
	"github.com/pborman/getopt/v2"
)

// Error constants for misc functions.
var (
	ErrGambleBadAmount = errors.New("bad amount of credits for action")
	ErrGambleNotMin    = func(a int) error { return fmt.Errorf("did not provide enough to reach the minimum of %d", a) }
	ErrGambleNotEnough = errors.New("not enough credits")
)

// Constants for Gambling.
const (
	GambleCredits = "gold"
	GambleMin     = 100
	GraphY        = 20
)

func (io *IOdat) miscRoll() {
	var roll1, roll2 int
	s := rand.NewSource(time.Now().UnixNano())
	r := rand.New(s)
	roll1 = r.Intn(6) + 1
	roll2 = r.Intn(6) + 1

	msg := fmt.Sprintf("```*%s rolls %d, %d*```", io.user.Username, roll1, roll2)

	io.rm = true
	io.output = msg
	return
}

func (io *IOdat) miscTop10() {
	var roll int
	s := rand.NewSource(time.Now().UnixNano())
	r := rand.New(s)

	roll = r.Intn(100)
	if roll <= 25 {
		io.output = fmt.Sprintf("**%s** is top 10!", io.user.Username)
		return
	}

	io.output = fmt.Sprintf("**%s** is **NOT** top 10.", io.user.Username)

	return
}

func creditsReset(database string, dgu *discordgo.User) (string, error) {
	var msg = "Users reset:\n\n"
	uID := dgu.ID
	user := UserNew(database, dgu)
	if err := user.Get(uID); err != nil {
		return "", err
	}

	if ok := user.HasPermission(permAll); !ok {
		return "", ErrBadPermissions
	}

	db := DBdatCreate(database, CollectionUsers, User{}, nil, nil)
	if err := db.dbGetAll(User{}); err != nil {
		return "", err
	}

	if len(db.Documents) == 0 {
		return "", nil
	}

	var found bool
	var doc User
	for _, u := range db.Documents {
		doc = u.(User)
		if doc.Credits != doc.CreditsTotal {
			found = true
			msg += fmt.Sprintf("\t__**%s**#%s__: %d -> %d\n",
				doc.Username, doc.Discriminator, doc.Credits, doc.CreditsTotal)
			doc.Credits = doc.CreditsTotal
			if err := doc.Update(); err != nil {
				return "", err
			}
		}
	}
	if !found {
		msg = "No users updated."
	}
	return msg, nil
}

func pasteIt(msg, title string) (string, error) {
	p := go_pastebin.NewPastebin(envPBDK)
	pb, err := p.GenerateUserSession(envPB, envPBPW)
	if err != nil {
		return "", err
	}

	paste, err := pb.Paste(msg, title, "c", "10M", "1")
	if err != nil {
		fmt.Println(err)
		return "", err
	}

	return paste.String(), nil
}

// Help prints various command assistance.
func Help(f *getopt.Set, prefix, suffix string) string {

	var buf = new(bytes.Buffer)
	f.PrintUsage(buf)

	return "```" + prefix + buf.String() + suffix + "```"
}

func channelsTemp() string {
	var msg string
	msg += "```C\n"
	for k, c := range Bot.Links {
		gg := Bot.GetGuild(k)
		if strings.Contains(gg.Name, "verse") {
			msg += fmt.Sprintf("Guild: [%s] [%s]\n", gg.Name, gg.ID)
			for n, cc := range c {
				msg += fmt.Sprintf("[%2d] Channel: [%16s] [%5s] [%s]\n", n, cc.Name, cc.Type, cc.ID)
			}
		}
	}
	return msg + "```"
}

func (io *IOdat) histograph(s *discordgo.Session, database string) error {
	var snd string
	var mp = make(map[int]map[int]int)

	if ok := io.user.HasPermission(permAdmin); !ok {
		return ErrBadPermissions
	}

	// Get ALL messages from Database
	dat := DBdatCreate(database, CollectionMessages, Message{}, nil, nil)
	if err := dat.dbGetAll(Message{}); err != nil {
		return err
	}

	if len(dat.Documents) == 0 {
		return fmt.Errorf("no documents found")
	}

	var msg Message
	for _, d := range dat.Documents {
		msg = d.(Message)
		t := msg.Timestamp
		if _, ok := mp[t.Year()]; !ok {
			mp[t.Year()] = make(map[int]int)
		}
		mp[t.Year()][int(t.Month())]++
	}

	var t [12]int
	var keys []int
	for k := range mp {
		keys = append(keys, k)
	}
	sort.Ints(keys)
	// Calculate and send the Histograms for each year.
	for _, k := range keys {
		snd += fmt.Sprintf("Year: %d\n", k)
		var m [12]int
		var starMax, star int
		for _, cnt := range mp[k] {
			if cnt > starMax {
				starMax = cnt
			}
		}

		star = starMax / GraphY
		// If there isn't enough messages... skip to the next year.
		if star == 0 {
			snd += "\tNot enough data.\n"
			s.ChannelMessageSend(io.msg.ChannelID, "```"+snd+"```")
			snd = ""
			continue
		}
		for mon, cnt := range mp[k] {
			m[mon-1] = cnt / star
			t[mon-1] += cnt
		}

		starMax = GraphY
		for starMax > 0 {
			var ln string
			for n, amt := range m {
				if starMax > 0 {
					if amt == starMax {
						ln += fmt.Sprintf("  █  ")
						m[n]--
					} else {
						ln += fmt.Sprintf("     ")
					}
				}
				// If N is len(m), print the newline
				if n+1 == len(m) {
					for i := len(ln); i > 0; i-- {
						if ln[i-1] != ' ' {
							break
						}
						ln = strings.TrimSuffix(ln, " ")
					}

					snd += ln + "\n"
					break
				}

			}
			// If loop n == len, print newline
			if starMax-1 == 0 {
				for i := 0; i < 12; i++ {
					snd += fmt.Sprintf(" %s ", monToString(i+1))
				}
				s.ChannelMessageSend(io.msg.ChannelID, fmt.Sprintf("```%s```", snd))
				snd = ""
			}
			starMax--
		}
	}

	// Calculate based on month
	var res string
	res += "Mon: Message Count\n"
	for n, c := range t {
		res += fmt.Sprintf("%s: %d\n", monToString(n+1), c)
	}
	s.ChannelMessageSend(io.msg.ChannelID, "```"+res+"```")

	return nil
}

func (io *IOdat) roomGen() {
	d := generate.NewDungeon()
	io.output = "```\n" + d.String() + "```"
}

func monToString(i int) string {
	switch i {
	case 1:
		return "jan"
	case 2:
		return "feb"
	case 3:
		return "mar"
	case 4:
		return "apr"
	case 5:
		return "may"
	case 6:
		return "jun"
	case 7:
		return "jul"
	case 8:
		return "aug"
	case 9:
		return "sep"
	case 10:
		return "oct"
	case 11:
		return "nov"
	case 12:
		return "dec"
	}
	return ""
}
