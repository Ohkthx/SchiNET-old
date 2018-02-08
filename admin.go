package main

import (
	"errors"
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"

	mgo "gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

// Permissions and Errors.
var (
	rolePermissionAdmin = 0x00000008
	rolePermissionMod   = 0x00000001 | 0x00000002 | 0x00002000
	rolePermissionBan   = 0x0
	ErrRoleNotFound     = errors.New("Role was not found")
)

// CoreAdmin parses commands intended for the admin interface.
func (conf *Config) CoreAdmin(dat *IOdata) error {
	if len(dat.io) < 2 {
		return errors.New("Need more arguments than that")
	}

	// Return if the user does not have the role
	if ok := dat.user.HasRoleType(dat.guildConfig, rolePermissionAdmin); !ok {
		return ErrBadPermissions
	}

	var update bool
	arg := strings.ToLower(dat.io[1])
	if arg == "reset" {
		// Reset Guild Config here... update.
		update = true
		dat.guildConfig.Prefix = ConfigFile.Prefix
		if err := conf.Core.SetNickname(dat.guild.ID, "", false); err != nil {
			return err
		}
	} else if arg == "prefix" {
		if len(dat.io) < 3 {
			return ErrBadArgs
		}
		// Modify guild config... update
		update = true
		dat.guildConfig.Prefix = dat.io[2]
	} else if arg == "nick" {
		if len(dat.io) < 3 {
			return ErrBadArgs
		} else if len(dat.io[2]) > 32 {
			return errors.New("name supplied is too long")
		}
		if err := conf.Core.SetNickname(dat.guild.ID, dat.io[2], false); err != nil {
			return err
		}
	} else if arg == "grant" {
		if len(dat.io) < 4 {
			return ErrBadArgs
		}

		user := UserNew(nil)
		if err := user.Get(dat.io[3]); err != nil {
			return err
		}

		var roleID string
		if dat.io[2] == "admin" {
			roleID = dat.guildConfig.RoleIDGet(rolePermissionAdmin)
		} else {
			roleID = dat.guildConfig.RoleIDGet(rolePermissionMod)
		}

		// Grant the role to the admin via discord.
		if err := conf.Core.Session.GuildMemberRoleAdd(dat.guild.ID, user.ID, roleID); err != nil {
			return err
		}

		user.RoleAdd(dat.guild.ID, roleID)
		if err := user.Update(); err != nil {
			return err
		}

		dat.output = "Role granted."
	} else if arg == "channel" {
		return dat.ChannelCore()
	} else if arg == "help" {
		dat.output = fmt.Sprintf("Admin Help:\n"+
			"```%s\n\t - %s\n"+
			"%s\n\t - %s\n"+
			"%s\n\t - %s\n"+
			"%s\n\t - %s\n"+
			"%s\n\t - %s\n```",
			"admin reset", "Resets to the bot's defaults.",
			"admin prefix [prefix]", "Sets the bots command prefix to the desired.",
			"admin nick [new_nick]", "Assigns a new name to the bot.",
			"admin channel enable/disable", " Enable or disable bot commands in the channel.",
			"admin grant [role] [id]", "Grants either an Admin or Moderator role to a user.")
		return nil
	}

	if update {
		// Update the config here.
		if err := conf.GuildConfigManager(dat.guildConfig); err != nil {
			return err
		}
	}

	return nil
}

// createGuildRoles for a new guild.
func (conf *Config) createGuildRoles(guildConfig *GuildConfig, guildID string) error {
	session := conf.Core.Session
	if session == nil {
		return errors.New("Session is nil when creating roles")
	}
	// Get current existing roles for the guild.
	guildRoles, err := session.GuildRoles(guildID)
	if err != nil {
		return err
	}

	rolesNew := make(map[string]int)
	rolesNew["SchiNET-Administrator"] = rolePermissionAdmin
	rolesNew["SchiNET-Moderator"] = rolePermissionMod
	rolesNew["SchiNET-Restricted"] = 0

	// Iterate the guild names that need to be added.
	for roleName, roleValue := range rolesNew {
		var exists bool
		// Search the current guild for a pre-existing role.
		for _, r := range guildRoles {
			if r.Name == roleName {
				// Found, break to continue to search for next name.
				exists = true
				// Compare existing roles permissions and validate they are correct.
				printDebug("Checking permissions on prexisting guild.")
				if r.Permissions&roleValue != roleValue {
					printDebug("Permissions are incorrect... correcting.")
					if _, err := session.GuildRoleEdit(guildID, r.ID, r.Name, r.Color, r.Hoist, r.Permissions|roleValue, r.Mentionable); err != nil {
						return err
					}
				}
				// Assign the role to the guild's configuration.
				if guildConfig != nil {
					guildConfig.RoleAdd(r.ID, "", roleName, r.Permissions|roleValue, roleValue)
				}
				break
			}
		}

		// If the name already exists... skip it.
		if exists {
			continue
		}

		// Create the Guild Role here.
		newRole, err := session.GuildRoleCreate(guildID)
		if err != nil {
			return nil
		}

		// Edit the guild role: Sets defaults... also grants additional permissions to role.
		if _, err = session.GuildRoleEdit(guildID, newRole.ID, roleName, newRole.Color, newRole.Hoist, newRole.Permissions|roleValue, newRole.Mentionable); err != nil {
			return err
		}

		// Assign the role to the guild's configuration.
		if guildConfig != nil {
			guildConfig.RoleAdd(newRole.ID, "", roleName, roleValue, roleValue)
		}
	}

	// Save the guild to the database.
	if guildConfig != nil {
		if err = guildConfig.Update(); err != nil {
			return err
		}
	}

	return nil
}

// guildPermissionAdd  Adds a role to a user.
func (conf *Config) guildPermissionAdd(guildID, userID, roleID string) error {
	session := conf.Core.Session
	if session == nil {
		return errors.New("Nil session while adding permissions")
	}

	return session.GuildMemberRoleAdd(guildID, userID, roleID)

}

// guildPermissionRemove Removes a permission for a user.
func (conf *Config) guildPermissionRemove(guildID, userID, roleID string) error {
	session := conf.Core.Session
	if session == nil {
		return errors.New("Nil session while removing permissions")
	}

	return session.GuildMemberRoleRemove(guildID, userID, roleID)
}

// GuildConfigLoad loads guild configs into memory for quicker access.
func (conf *Config) GuildConfigLoad() error {
	// Scan current guilds
	for _, g := range conf.Core.Guilds {
		var gc = newGuildConfig(g.ID, g.Name)
		if err := gc.Get(); err != nil {
			if err == mgo.ErrNotFound {
				fmt.Printf("DEBUG: New Guild while while loading:\n [%s] %s\n", g.ID, g.Name)
				ng := &discordgo.GuildCreate{Guild: g}
				conf.guildCreateHandler(conf.Core.Session, ng)
				return nil
			}
			return err
		}

		// Add it to the current config structure
		// TAG: TODO - it's updating into DB in GuildConfigManager- potentially remove.
		if err := conf.GuildConfigManager(gc); err != nil {
			return err
		}
	}
	return nil
}

// GuildConfigManager will append guilds if they're not already in the running config.
func (conf *Config) GuildConfigManager(guild *GuildConfig) error {
	// Find guild and replace with updated version.
	for n, g := range conf.GuildConf {
		if g.ID == guild.ID {
			conf.GuildConf[n] = guild
			err := guild.Update()
			return err
		}
	}

	if err := guild.Update(); err != nil {
		return err
	}

	// Guild wasn't found, needs to be appended.
	conf.GuildConf = append(conf.GuildConf, guild)
	return nil
}

// GuildConfigByID will search the running guild configurations and return a matching instance.
// If it isn't found, return nil.
func (conf *Config) GuildConfigByID(gID string) *GuildConfig {
	// Scan the current configs.
	for n, g := range conf.GuildConf {
		if g.ID == gID {
			return conf.GuildConf[n]
		}
	}

	// Wasn't found- return nil. Caller should check nil value.
	return nil
}

// GuildRoleByID will find a type of role within a guild.
func (conf *Config) GuildRoleByID(guildID string, roleBase int) string {
	// Search all Guilds for a specific one.
	for _, g := range conf.GuildConf {
		if g.ID == guildID {
			// Guild found. Search for the type of role (based on it's base value.)
			for _, role := range g.Roles {
				if role.Base == roleBase {
					return role.ID
				}
			}
		}
	}
	return ""
}

// newGuildConfig creates a new GuildConf object.
func newGuildConfig(gID, gName string) *GuildConfig {
	return &GuildConfig{
		ID:     gID,
		Name:   gName,
		Init:   false,
		Prefix: ",",
	}
}

// Get a guild from DB
func (g *GuildConfig) Get() error {
	var q = make(map[string]interface{})

	q["id"] = g.ID

	var dbdat = DBdataCreate(g.ID, CollectionConfig, GuildConfig{}, q, nil)
	err := dbdat.dbGet(GuildConfig{})
	if err != nil {
		return err
	}

	var guild GuildConfig
	guild = dbdat.Document.(GuildConfig)

	if guild.Prefix == "" {
		guild.Prefix = ConfigFile.Prefix
	}

	*g = guild

	return nil
}

// Update a guild's config.
func (g *GuildConfig) Update() error {
	var err error
	var q = make(map[string]interface{})
	var c = make(map[string]interface{})

	if g.Prefix == "" {
		g.Prefix = ConfigFile.Prefix
	}

	q["id"] = g.ID
	c["$set"] = bson.M{
		"id":     g.ID,
		"name":   g.Name,
		"init":   g.Init,
		"roles":  g.Roles,
		"prefix": g.Prefix,
	}

	var dbdat = DBdataCreate(g.ID, CollectionConfig, g, q, c)
	err = dbdat.dbEdit(GuildConfig{})
	if err != nil {
		if err == mgo.ErrNotFound {
			// Add to DB since it doesn't exist.
			err := dbdat.dbInsert()
			return err
		}
		return err
	}

	return nil
}

// RoleCheck if a guild has a role.
func (g *GuildConfig) RoleCheck(roleID string) bool {
	for _, r := range g.Roles {
		if r.ID == roleID {
			return true
		}
	}
	return false
}

// RoleIDGet gets a roleID based on it's basic value
func (g *GuildConfig) RoleIDGet(base int) string {
	for _, r := range g.Roles {
		if r.Base == base {
			return r.ID
		}
	}
	return ""
}

// RoleAdd check if a role exists, if not- update.
func (g *GuildConfig) RoleAdd(roleID, roleOldID, roleName string, value, base int) {
	if roleOldID != "" {
		// It does exist... update it.
		for n, r := range g.Roles {
			if r.ID == roleOldID {
				// Found, just update it.
				g.Roles[n].ID = roleID
				g.Roles[n].Name = roleName
				g.Roles[n].Value = value
				g.Roles[n].Base = base
				return
			}
		}
	}

	// Check if the role exists.
	if ok := g.RoleCheck(roleID); !ok {
		g.Roles = append(g.Roles, Role{ID: roleID, Name: roleName, Value: value, Base: base})
		return
	}

	return
}

// RoleCorrection verifies roles are as they should be upon last save.
func (g *GuildConfig) RoleCorrection(s *discordgo.Session) error {
	// Make sure we're not accessing a nil session:
	if s == nil {
		return errors.New("Session is nil when performing Role Correction")
	}

	// Get our potentially updated roles for the guild from the server.
	dRoles, err := s.GuildRoles(g.ID)
	if err != nil {
		return err
	}

	// Iterate our known roles.
	for n, role := range g.Roles {
		var roleFound *discordgo.Role
		for _, r := range dRoles {
			if role.ID == r.ID {
				// Found our role, break inner loop to process.
				roleFound = r
				break
			}
		}

		// If it exists, check permissions. If not... recreate.
		if roleFound != nil {
			if roleFound.Permissions&role.Base != role.Base || roleFound.Name != role.Name {
				newValue := roleFound.Permissions | role.Base
				_, err = s.GuildRoleEdit(g.ID, role.ID, role.Name, roleFound.Color, roleFound.Hoist, newValue, roleFound.Mentionable)
				if err != nil {
					return err
				}
				// Update the running permission value if editting succeeded.
				g.Roles[n].Value = newValue
			} else if roleFound.Name == "SchiNET-Banned" {
				_, err = s.GuildRoleEdit(g.ID, role.ID, "SchiNET-Restricted", roleFound.Color, roleFound.Hoist, roleFound.Permissions, roleFound.Mentionable)
				if err != nil {
					return err
				}
				g.Roles[n].Name = "SchiNET-Restricted"
			}
			continue
		}

		// If it wasn't found... recreate it.
		newRole, err := s.GuildRoleCreate(g.ID)
		if err != nil {
			return err
		}

		// Modify to correct permissions.
		_, err = s.GuildRoleEdit(g.ID, newRole.ID, role.Name, newRole.Color, newRole.Hoist, role.Value, newRole.Mentionable)
		if err != nil {
			return err
		}

		// Add the new role.
		g.RoleAdd(newRole.ID, role.ID, role.Name, role.Value, role.Base)
	}

	return nil
}

// InternalCorrection verifies an internal channel exists. If not, it recreates it.
func (conf *Config) InternalCorrection(guildID string) error {
	// Attempt to find a pre-existing #internal
	var internal *discordgo.Channel
	for _, ch := range conf.Core.Links[guildID] {
		if ch.Name == "internal" {
			internal = ch
			break
		}
	}

	var err error
	// Channel doesn't exists... needs to be created.
	if internal == nil {
		if internal, err = conf.Core.Session.GuildChannelCreate(guildID, "internal", "text"); err != nil {
			return err
		}
		conf.Core.ChannelMemoryAdd(internal)
	}

	if len(internal.PermissionOverwrites) == 0 {
		err := conf.Core.Session.ChannelPermissionSet(internal.ID, guildID, "role", 0, 0x00000400)
		return err
	}

	// Check if the permissions are right... if not- update.
	for _, p := range internal.PermissionOverwrites {
		if p.ID == guildID {
			// This is @everyone (matches Guild ID) check the permissions.
			if p.Deny&0x00000400 != 0x00000400 {
				// Permission isn't set correctly, update.
				if p.Allow&0x00000400 == 0x00000400 {
					p.Allow ^= 0x00000400
				}
				err := conf.Core.Session.ChannelPermissionSet(internal.ID, p.ID, p.Type, p.Allow, p.Deny|0x00000400)
				return err
			}
			return nil
		}
	}
	return nil
}

// MemberCorrection checks all members and their roles and corrects them in the database.
func (conf *Config) MemberCorrection() error {
	core := conf.Core
	// Process each guild.
	for _, g := range core.Guilds {
		// Pull all users of the guild.
		users, err := core.GetGuildMembers(g.ID, g.MemberCount)
		if err != nil {
			fmt.Println("Pulling guild members: " + err.Error())
			// Continue because we can try to process other guilds.
			continue
		}

		// Process the users.
		for _, u := range users {
			// Pull the User from the database to update guild specific information.
			user := UserNew(u.User)
			if err := user.Get(u.User.ID); err != nil {
				if err != mgo.ErrNotFound {
					// Skip and try the next one.
					// TAG: TODO - Send this to error log for MongoDB.
					fmt.Println(err)
					continue
				}
			}

			// Replace or append the new roles.
			if len(user.GuildRoles) > 0 {
				var exists bool
				for n, gr := range user.GuildRoles {
					if gr.ID == g.ID {
						user.GuildRoles[n] = GuildRole{ID: g.ID, Name: g.Name, Roles: u.Roles}
						exists = true
						break
					}
				}
				if !exists {
					user.GuildRoles = append(user.GuildRoles, GuildRole{ID: g.ID, Name: g.Name, Roles: u.Roles})
				}
			} else {
				user.GuildRoles = append(user.GuildRoles, GuildRole{ID: g.ID, Name: g.Name, Roles: u.Roles})
			}

			// Update the user.
			if err := user.Update(); err != nil {
				// TAG: TODO - Log for MongoDB
				fmt.Println(err)
				continue
			}
		}
	}

	return nil
}
