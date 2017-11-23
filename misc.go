package main

import (
	"bytes"
	"errors"
	"fmt"
	"math/rand"
	"sort"
	"strings"
	"time"

	mgo "gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"

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

func (dat *IOdata) miscRoll() {
	var roll1, roll2 int
	s := rand.NewSource(time.Now().UnixNano())
	r := rand.New(s)
	roll1 = r.Intn(6) + 1
	roll2 = r.Intn(6) + 1

	msg := fmt.Sprintf("```*%s rolls %d, %d*```", dat.user.Username, roll1, roll2)

	dat.rm = true
	dat.output = msg
	return
}

func (dat *IOdata) miscTop10() {
	var roll int
	s := rand.NewSource(time.Now().UnixNano())
	r := rand.New(s)

	roll = r.Intn(100)
	if roll <= 25 {
		dat.output = fmt.Sprintf("**%s** is top 10!", dat.user.Username)
		return
	}

	dat.rm = true
	dat.output = fmt.Sprintf("**%s** is **NOT** top 10.", dat.user.Username)

	return
}

func creditsReset() (string, error) {
	var msg = "```\nUsers reset:\n\n"

	db := DBdataCreate(Database, CollectionUsers, User{}, nil, nil)
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
	} else {
		msg += "\n```"
	}
	return msg, nil
}

func pasteIt(msg, title string) (string, error) {
	p := go_pastebin.NewPastebin(envPBDK)
	pb, err := p.GenerateUserSession(envPB, envPBPW)
	if err != nil {
		return "", err
	}

	paste, err := pb.Paste(msg, title, "vim", "10M", "1")
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

// histograph creates a timeline of message activity within a year.
func (dat *IOdata) histograph(s *discordgo.Session) error {
	var snd string
	var mp = make(map[int]map[int]int)

	// Return if the user does not have the role
	if ok := dat.user.HasRoleType(dat.guildConfig, rolePermissionAdmin); !ok {
		return ErrBadPermissions
	}

	// Get ALL messages from Database
	data := DBdataCreate(dat.guild.ID, CollectionMessages, Message{}, nil, nil)
	if err := data.dbGetAll(Message{}); err != nil {
		return err
	}

	if len(data.Documents) == 0 {
		return fmt.Errorf("no documents found")
	}

	var msg Message
	for _, d := range data.Documents {
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
			s.ChannelMessageSend(dat.msg.ChannelID, "```"+snd+"```")
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
				s.ChannelMessageSend(dat.msg.ChannelID, fmt.Sprintf("```%s```", snd))
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
	s.ChannelMessageSend(dat.msg.ChannelID, "```"+res+"```")

	return nil
}

// roomGen generates a room for an RPG that will be integrated.
// TAG: TODO
func (dat *IOdata) roomGen() {
	d := generate.NewDungeon()
	dat.output = "```\n" + d.String() + "```"
}

// monToString converts a month int, to a 3 letter string.
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

// ChannelNew creates a new channel object.
func ChannelNew(cID, gID string) *ChannelInfo {
	return &ChannelInfo{
		ID:     cID,
		Server: gID,
	}
}

// ChannelCore handler for channel operations.
func (dat *IOdata) ChannelCore() error {
	var err error
	var msg string
	var ch = ChannelNew(dat.msg.ChannelID, dat.guild.ID)

	// Return if the user does not have the role
	if ok := dat.user.HasRoleType(dat.guildConfig, rolePermissionAdmin); !ok {
		return ErrBadPermissions
	}

	if len(dat.io) < 3 {
		return ErrBadArgs
	} else if dat.io[2] == "enable" {
		msg, err = ch.Enable()
	} else if dat.io[2] == "disable" {
		msg, err = ch.Disable()
	}

	if err != nil {
		return err
	} else if msg != "" {
		dat.msgEmbed = embedCreator(msg, ColorGray)
		return nil
	}
	return ErrBadArgs
}

// Enable a channel for bot commands.
func (ch *ChannelInfo) Enable() (string, error) {
	if err := ch.Get(); err != nil {
		if err == mgo.ErrNotFound {
			return "Bot commands are already enabled in this channel.", nil
		}
		return "", err
	}

	if ch.Enabled {
		return "Bot commands are already enabled in this channel.", nil
	}

	ch.Enabled = true
	if err := ch.Update(); err != nil {
		return "", err
	}
	return "Bot commands are now enabled for this channel.", nil
}

// Disable a channel for bot commands.
func (ch *ChannelInfo) Disable() (string, error) {
	if err := ch.Get(); err != nil {
		if err == mgo.ErrNotFound {
			// Not found, need to add.
			ch.Enabled = false
			if err := ch.Update(); err != nil {
				return "", err
			}
			return "Bot commands have been disabled for this channel.", nil
		}
		return "", err
	}

	if !ch.Enabled {
		return "Bot commands are already disabled for this channel.", nil
	}

	ch.Enabled = false
	if err := ch.Update(); err != nil {
		return "", err
	}
	return "Bot commands have beem disabled for this channel.", nil
}

// Check to see if a channel is eligible to do bot commands.
func (ch *ChannelInfo) Check() bool {
	if err := ch.Get(); err != nil {
		if err == mgo.ErrNotFound {
			return true
		}
		return false
	}

	if ch.Enabled {
		return true
	}

	return false
}

// Get a channel from database.
func (ch *ChannelInfo) Get() error {
	var q = make(map[string]interface{})

	q["id"] = ch.ID

	dbdat := DBdataCreate(ch.Server, CollectionChannels, ChannelInfo{}, q, nil)
	err := dbdat.dbGet(ChannelInfo{})
	if err != nil {
		return err
	}

	var channel = ChannelInfo{}
	channel = dbdat.Document.(ChannelInfo)
	*ch = channel

	return nil
}

// Update a channels representation in database.
func (ch *ChannelInfo) Update() error {
	var err error
	var q = make(map[string]interface{})
	var c = make(map[string]interface{})

	q["id"] = ch.ID
	c["$set"] = bson.M{
		"name":    ch.Name,
		"server":  ch.Server,
		"enabled": ch.Enabled,
	}

	dbdat := DBdataCreate(ch.Server, CollectionChannels, ch, q, c)
	err = dbdat.dbEdit(ChannelInfo{})
	if err != nil {
		if err == mgo.ErrNotFound {
			// Add to DB since it doesn't exist.
			if err := dbdat.dbInsert(); err != nil {
				return err
			}
			return nil
		}
		return err
	}

	return nil
}

func echoMsg(msg []string) string {
	var output string
	for _, w := range msg {
		switch w {
		case "@everyone":
			output += "everyone "
		default:
			output += w + " "
		}
	}
	return output
}
