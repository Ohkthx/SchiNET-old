package main

import (
	"errors"

	"github.com/bwmarrin/discordgo"

	mgo "gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

// Error constants.
var (
	ErrBadUser = errors.New("bad user supplied")
)

// Permission scheme constants.
const (
	permAdmin = 1 << iota
	permModerator
	permAscended
	permNormal
)

const permAll = permAdmin | permModerator | permAscended | permNormal

/*
func getUser(database, uID string) (*User, error) {
	var u = User{}
	q := make(map[string]interface{})
	q["id"] = uID

	db := DBdatCreate(database, CollectionConfigs, User{}, q, nil)
	err := db.dbGet(PermissionUser{})
	if err != nil {
		return nil, ErrBadUser
	}

	u = db.Document.(User)
	return &u, nil
}
*/

// UserNew creates a new user instance based on accessing user.
func UserNew(u *discordgo.User) *User {
	var user = User{
		ID:            u.ID,
		Username:      u.Username,
		Discriminator: u.Discriminator,
		Bot:           u.Bot,
		CreditsTotal:  1,
		Credits:       1,
		Access:        permNormal,
	}
	return &user
}

// UserUpdateSimple stream-lines the process for incrementing credits.
func UserUpdateSimple(database string, user *discordgo.User, inc int) error {
	u := UserNew(user)

	if err := u.Get(database, u.ID); err != nil {
		if err == mgo.ErrNotFound {
			if err := u.Update(database); err != nil {
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

	if err := u.Update(database); err != nil {
		return err
	}

	return nil

}

// GetPermission returns True if have permissions for action, false if not.
func (u *User) GetPermission(access int) bool { return u.Access&access == access }

// PermissionAdd upgrades a user to new permissions
func (u *User) PermissionAdd(access int) { u.Access &= access }

// PermissionDelete strips a permission from an User
func (u *User) PermissionDelete(access int) { u.Access ^= access }

// Update pushes an update to the database.
func (u *User) Update(database string) error {
	var err error
	var q = make(map[string]interface{})
	var c = make(map[string]interface{})

	q["id"] = u.ID
	c["$set"] = bson.M{"creditstotal": u.CreditsTotal, "credits": u.Credits, "access": u.Access}

	var dbdat = DBdatCreate(database, CollectionUsers, u, q, c)
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
func (u *User) Get(database, uID string) error {
	var q = make(map[string]interface{})

	q["id"] = uID

	var dbdat = DBdatCreate(database, CollectionUsers, User{}, q, nil)
	err := dbdat.dbGet(User{})
	if err != nil {
		return err
	}

	var user = User{}
	user = dbdat.Document.(User)
	u.ID = user.ID
	u.Username = user.Username
	u.Discriminator = user.Discriminator
	u.Bot = user.Bot
	u.Credits = user.Credits
	u.CreditsTotal = user.CreditsTotal
	u.Access = user.Access

	return nil
}
