package main

import (
	"errors"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/pborman/getopt/v2"

	mgo "gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

// Flags that can be parsed related to User commands.
type userFlags struct {
	flag       *getopt.Set // FlagSet object -> used for Help
	server     string      // Server for opperation.
	serverName string      // Name of the server.
	// Generics
	User   string // User to perform action on.
	Type   string // Type of Action.
	Help   bool   // Command Help
	List   bool   // List Command objects/items/etc
	Add    bool
	Remove bool

	// Ban related
	BotAbuse bool // Bot is being abused.

	// Credit related
	Xfer   bool // Transfer
	Gamble bool // Gamble
	Amount int  // Amount of Credits for action.
	All    bool // All credits.

	// Permission related
	Permission bool
}

// Error constants.
var (
	ErrBadUser        = errors.New("bad user supplied")
	ErrBadPermissions = errors.New("you do not have permissions to do that")
	ErrBanChanExists  = errors.New("user already has a ban for that channel")
	ErrBanNotFound    = errors.New("ban not found to clear")

	abuseSyntax    = ",user --abuse --user \"@Username\"\n"
	abuseSyntaxAll = abuseSyntax

	permSyntaxAdd    = ",permission  --add     --type \"Permission\"  --user \"@Username\"\n"
	permSyntaxRemove = ",permission  --remove  --type \"Permission\"  --user \"@Username\"\n"
	permSyntaxTypes  = "Permission Types:\nAdmin  Moderator  Base(default)"
	permSyntaxAll    = "\n\n" + permSyntaxAdd + permSyntaxRemove + permSyntaxTypes

	userSyntaxUser       = ",user  --user \"@Username\"\n"
	userSyntaxGamble     = ",user  --gamble  -n 100\n"
	userSyntaxBan        = ",user  --ban  --type soft  --user \"@Username\"\n"
	userSyntaxXfer       = ",user  -x  --user \"@Username\"  -n 100\n"
	userSyntaxPermission = ",user  --permission  --add  --type \"mod\"  --user \"@Username\"\n"
	userSyntaxAll        = "\n\n" + userSyntaxUser + userSyntaxBan + userSyntaxGamble + userSyntaxPermission + userSyntaxXfer
)

// Permission scheme constants.
const (
	permNormal = 1 << iota
	permModerator
	permAdmin
	permAscended
)

const permAll = permAdmin | permModerator | permAscended | permNormal

// CoreUser processes all user-related commands.
func (dat *IOdata) CoreUser() error {
	u := dat.user
	var uflags userFlags
	var all bool

	for n, s := range dat.io {
		if s == "-n" {
			if n+1 <= len(dat.io)-1 {
				if dat.io[n+1] == "all" {
					dat.io = append(dat.io[:n], dat.io[n+2:]...)
					all = true
					uflags.All = true
				} else {
					break
				}
			} else {
				dat.io = append(dat.io, "0")
			}
		}
	}

	fl := getopt.New()

	// Generics
	fl.FlagLong(&uflags.User, "user", 0, "Username")
	fl.FlagLong(&uflags.Type, "type", 't', "Type")
	fl.FlagLong(&uflags.Help, "help", 'h', "This message")
	fl.FlagLong(&uflags.List, "list", 0, "List all Abusers.")
	fl.FlagLong(&uflags.Add, "add", 0, "Add")
	fl.FlagLong(&uflags.Remove, "remove", 0, "Remove")

	// Ban related.
	fl.FlagLong(&uflags.BotAbuse, "abuse", 0, "Ban a user from the bot.")

	// Gambling related.
	fl.FlagLong(&uflags.Xfer, "xfer", 'x', "Xfer credits")
	fl.FlagLong(&uflags.Gamble, "gamble", 'g', "Gamble")
	if !all {
		fl.Flag(&uflags.Amount, 'n', "Amount (Number)")
		fl.FlagLong(&uflags.All, "all", 0, "Gamble all Credits")
	}

	if err := fl.Getopt(dat.io, nil); err != nil {
		return err
	}
	if fl.NArgs() > 0 {
		if err := fl.Getopt(fl.Args(), nil); err != nil {
			return err
		}
	}

	uflags.flag = fl
	uflags.User = userIDClean(uflags.User)
	uflags.server = dat.guild.ID

	var msg string
	var err error
	switch {
	case uflags.BotAbuse:
		err = u.BotAbuse(dat, dat.msg.ChannelID, uflags)
	case uflags.Xfer:
		msg, err = u.Transfer(uflags.Amount, uflags.User)
	case uflags.Gamble:
		if uflags.All {
			uflags.Amount = u.Credits
		} else if len(dat.io) < 4 {
			return ErrBadArgs
		}
		msg, err = u.Gamble(uflags.Amount)
	default:
		if uflags.Help {
			dat.output = Help(fl, "", userSyntaxAll)
			return nil
		} else if uflags.User != "" {
			// Get user information
			user := UserNew(nil)
			if err := user.Get(uflags.User); err != nil {
				return err
			}
			dat.msgEmbed = user.EmbedCreate()
			return nil
		} else {
			// Return current User
			dat.msgEmbed = u.EmbedCreate()
			return nil
		}
	}

	if err != nil {
		return err
	}

	dat.msgEmbed = embedCreator(msg, ColorBlue)
	return nil
}

// UserNew creates a new user instance based on accessing user.
func UserNew(u *discordgo.User) *User {
	if u == nil {
		u = &discordgo.User{
			ID:            "",
			Username:      "",
			Discriminator: "",
			Bot:           false,
		}
	}
	var user = User{
		ID:            u.ID,
		Username:      u.Username,
		Discriminator: u.Discriminator,
		Bot:           u.Bot,
		CreditsTotal:  1,
		Credits:       1,
	}
	return &user
}

// UserUpdateSimple stream-lines the process for incrementing credits.
func UserUpdateSimple(serverID string, user *discordgo.User, inc int, ts time.Time) error {
	u := UserNew(user)

	if err := u.Get(u.ID); err != nil {
		if err == mgo.ErrNotFound {
			u.Credits = 0
			u.CreditsTotal = 0
			u.LastSeen = ts
			if err := u.Update(); err != nil {
				return err
			}
			return nil
		}
		return err
	}

	if inc == 1 {
		u.CreditsTotal++
		u.Credits++
	} else {
		u.Credits += inc
	}

	// Make sure to only add most recent messages in time.
	if ts.After(u.LastSeen) {
		u.LastSeen = ts
	}

	if err := u.Update(); err != nil {
		return err
	}

	return nil

}

// Update pushes an update to the database.
func (u *User) Update() error {
	var err error
	var q = make(map[string]interface{})
	var c = make(map[string]interface{})

	q["id"] = u.ID
	c["$set"] = bson.M{
		"username":     u.Username,
		"creditstotal": u.CreditsTotal,
		"credits":      u.Credits,
		"roles":        u.Roles,
		"lastseen":     u.LastSeen,
	}

	dbdat := DBdataCreate(Database, CollectionUsers, u, q, c)
	err = dbdat.dbEdit(User{})
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

// Get a user from a database
func (u *User) Get(uID string) error {
	var q = make(map[string]interface{})

	q["id"] = uID

	dbdat := DBdataCreate(Database, CollectionUsers, User{}, q, nil)
	err := dbdat.dbGet(User{})
	if err != nil {
		return err
	}

	var user = User{}
	user = dbdat.Document.(User)
	*u = user

	return nil
}

// GetByName from database.
func (u *User) GetByName(username string) error {
	var q = make(map[string]interface{})

	q["username"] = username

	dbdat := DBdataCreate(Database, CollectionUsers, User{}, q, nil)
	err := dbdat.dbGet(User{})
	if err != nil {
		return err
	}

	var user = User{}
	user = dbdat.Document.(User)
	*u = user

	return nil
}

// EmbedCreate returns an embed object with User information.
func (u *User) EmbedCreate() *discordgo.MessageEmbed {
	tn := time.Now()
	dur := tn.Sub(u.LastSeen)

	ta := fmt.Sprintf("**%d** hours, **%d** minutes.", int(dur.Hours()), int(dur.Minutes())%60)

	description := fmt.Sprintf(
		"__ID: %s__\n"+
			"**Username**:   %-18s **%s**: %-10d\n"+
			"**Reputation**: %d\n\n"+
			"**Last Seen**: %s",
		u.ID,
		fmt.Sprintf("%s#%s", u.Username, u.Discriminator), strings.Title(GambleCredits), u.Credits,
		u.CreditsTotal,
		ta)

	return &discordgo.MessageEmbed{
		Author:      &discordgo.MessageEmbedAuthor{},
		Color:       ColorBlue,
		Description: description,
		Fields:      []*discordgo.MessageEmbedField{},
	}
}

// String produces a Username#Discriminator string.
func (u *User) String() string {
	return u.Username + "#" + u.Discriminator
}

// StringPretty produces elegant looking string.
func (u *User) StringPretty() string {
	return "**" + u.Username + "**#" + u.Discriminator
}

// Basic provides a basic User structure.
func (u *User) Basic() UserBasic {
	return UserBasic{
		ID:            u.ID,
		Name:          u.Username,
		Discriminator: u.Discriminator,
	}
}

// String returns a basic string of the object.
func (ub UserBasic) String() string {
	return ub.Name + "#" + ub.Discriminator
}

// StringPretty makes everything prettier.
func (ub UserBasic) StringPretty() string {
	return "**" + ub.Name + "**#" + ub.Discriminator
}

/*
	BAN RELATED
	ACTIONS
*/

// BotAbuse will remove a player from being able to use the bot.
func (u *User) BotAbuse(dat *IOdata, cID string, fl userFlags) error {
	var err error
	var uID string

	// Properly format ID string given by the person performing the ban to compare if they're banning themselves..
	if fl.User != "" {
		ids := strings.FieldsFunc(fl.User, idSplit)
		uID = ids[0]
	}

	// Self-ban attempt.
	if u.ID == uID {
		dat.output = fmt.Sprintf("Oh c'mon <@%s> that would just be silly...", u.ID)
		return nil
	}

	if err = u.Get(u.ID); err != nil {
		return err
	}

	if ok := u.HasRoleType(dat.guildConfig, rolePermissionAdmin); !ok {
		return ErrBadPermissions
	} else if fl.Help {
		prefix := "**Need** a __username__.\n\n"
		dat.output = Help(fl.flag, prefix, abuseSyntaxAll)
		return nil
	} else if fl.List {
		if err = abuseList(dat, fl.serverName, fl.server); err != nil {
			return err
		}
		return nil
	}

	if uID != "" {
		// Find user.
		criminal := UserNew(nil)
		if err = criminal.Get(uID); err != nil {
			return err
		}

		// Apply the role to the user on Discord.
		roleID := dat.guildConfig.RoleIDGet(0)
		if roleID == "" {
			return errors.New("Unable to find role to apply to newly banned used")
		}
		if err := Bot.Session.GuildMemberRoleAdd(dat.guildConfig.ID, criminal.ID, roleID); err != nil {
			return err
		}

		// Apply the banned role to the user in memory.
		criminal.RoleAdd(roleID)

		// Apply the role to the user in the database.
		if err := criminal.Update(); err != nil {
			return err
		}

		dat.output = fmt.Sprintf("Bot access has been __**revoked**__ for <@%s>.", criminal.ID)
		return nil
	}

	return errors.New("Need to supply a user")

}

func abuseList(dat *IOdata, database, id string) error {
	return nil
}

/*
	GAMBLE RELATED
	ACTIONS
*/

// Gamble User Credits.
func (u *User) Gamble(amount int) (string, error) {
	var twealth, spoils int
	var err error

	if amount < 50 {
		return "", ErrGambleNotMin(50)
	} else if amount < u.Credits/10 {
		return "", ErrGambleNotMin(u.Credits / 10)
	} else if amount > u.Credits {
		return "", ErrGambleNotEnough
	}
	twealth = u.Credits - amount
	spoils = gambleAlgorithm(59, 39, 2, amount)

	var msg string
	if spoils <= 0 {
		msg = fmt.Sprintf("%s gambled **%d** %s\n"+
			"Result: **loss**\n"+
			"%s remaining in bank: **%d**.",
			u.StringPretty(), amount, GambleCredits, strings.Title(GambleCredits), twealth)

		bu := UserNew(Bot.User)
		bu.Get(bu.ID)
		bu.Credits += amount
		err = bu.Update()
		if err != nil {
			return "", err
		}
	} else {
		twealth += spoils
		msg = fmt.Sprintf("%s gambled **%d** %s\n"+
			"Result: **Won**    spoils: **%d**\n"+
			"%s remaining in bank: **%d**.",
			u.StringPretty(), amount, GambleCredits, spoils, strings.Title(GambleCredits), twealth)
	}

	// twealth has new player bank amount.
	// Need to get difference and increment.
	u.Credits = twealth

	err = u.Update()
	if err != nil {
		return "", err
	}

	return msg, nil
}

// Transfer sends credits to another user.
func (u *User) Transfer(amount int, uID string) (string, error) {

	if amount < 0 {
		msg := "The authorities have been alerted with your attempt of theft!"
		return msg, nil
	} else if amount == 0 || uID == u.ID {
		msg := "What do you plan on accomplishing with this? Amount was 0. Use `-n Number`"
		return msg, nil
	}

	u2 := UserNew(nil)
	if err := u2.Get(uID); err != nil {
		return "", err
	}

	if u.Credits >= amount {
		u.Credits -= amount
		if err := u.Update(); err != nil {
			return "", err
		}

		u2.Credits += amount
		if err := u2.Update(); err != nil {
			return "", err
		}

		msg := fmt.Sprintf("%s has transferred __**%d**__ %s to %s.",
			u.StringPretty(), amount, GambleCredits, u2.StringPretty())

		return msg, nil
	}
	msg := fmt.Sprintf("You can't afford %d %s, you only have %d.",
		amount, GambleCredits, u.Credits)

	return msg, nil
}

// gambleAlgorithm for getting win/loss amount.
func gambleAlgorithm(l, d, t, credits int) int {
	s1 := rand.NewSource(time.Now().UnixNano())
	r1 := rand.New(s1)

	total := l + d + t
	if total != 100 {
		l = 60
		d = 38
		t = 2
	}

	num := r1.Intn(100)
	if num < l {
		// Lose all
		credits = 0
	} else if num >= l && num < l+d {
		// Win x2
		credits *= 2
	} else if num >= l+d {
		// Win x3
		credits *= 3
	}

	return credits
}

/*
	PERMISSION RELATED
	ACTIONS
*/

// HasRole will check if a particular user has a role assigned in discord.
func (u *User) HasRole(roleID string) bool {
	for _, r := range u.Roles {
		if r == roleID {
			return true
		}
	}
	return false
}

// HasRoleType checks if the user has a specific type of role.
func (u *User) HasRoleType(guildConfig *GuildConfig, base int) bool {
	// Get the role ID.
	var roleID string
	if roleID = guildConfig.RoleIDGet(base); roleID == "" {
		return false
	}

	// Return based on if the user has the role or not.
	return u.HasRole(roleID)
}

// RoleAddByType will grant a user a role based on it's type and not ID.
func (u *User) RoleAddByType(guildConfig *GuildConfig, base int) error {
	// Get the role ID.
	var roleID string
	if roleID = guildConfig.RoleIDGet(base); roleID == "" {
		return ErrBadPermissions
	}

	u.RoleAdd(roleID)
	return nil
}

// RoleAdd to a user.
func (u *User) RoleAdd(roleID string) {
	for _, r := range u.Roles {
		if r == roleID {
			return
		}
	}
	u.Roles = append(u.Roles, roleID)
	return
}

// RoleRemove from a user.
func (u *User) RoleRemove(roleID string) {
	for n, r := range u.Roles {
		if r == roleID {
			if len(u.Roles) == 1 {
				u.Roles = nil
				return
			}

			u.Roles[n] = u.Roles[len(u.Roles)-1]
			u.Roles = u.Roles[:len(u.Roles)-1]
			return
		}
	}
	return
}

/*
	Miscellanous Functions
*/

// Attempts to return an ID of a user despite <@ID> or Name#Discrim format.
func userIDClean(str string) string {
	if strings.ContainsRune(str, '@') {
		ids := strings.FieldsFunc(str, idSplit)
		id := ids[0]
		return id
	}
	return ""
}
