package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/d0x1p2/godbot"
	mgo "gopkg.in/mgo.v2"
)

// guildCreateHandler Handles newly added guilds that have invited the bot to the server.
func (conf *Config) guildCreateHandler(s *discordgo.Session, ng *discordgo.GuildCreate) {
	// If the gloabl is nil (not ready yet), return.
	if conf.Core == nil {
		return
	}

	var err error
	// Tell the bot core to update all connections.
	if err := conf.Core.UpdateConnections(); err != nil {
		fmt.Println(err)
		return
	}

	// Create a Guild Configuration
	var guildConfig = newGuildConfig(ng.ID, ng.Name)
	if err = guildConfig.Get(); err != nil {
		if err != mgo.ErrNotFound {
			fmt.Println("Initiating a new guild: " + err.Error())
		}
	}

	// If it is not initated- notify the chat and the owner.
	if guildConfig.Init == false {
		if guildConfig == nil {
			printDebug("Guild is nil?!")
		}

		// Create the guild roles.
		if err = conf.createGuildRoles(guildConfig, ng.ID); err != nil {
			fmt.Println("Creating roles: " + err.Error())
			return
		}

		// Create the internal channel.
		if err = conf.InternalCorrection(guildConfig.ID); err != nil {
			fmt.Println("Creating internal: " + err.Error())
			return
		}

		// Get the guild owner.
		var user *discordgo.Member
		user, err = s.GuildMember(ng.Guild.ID, ng.OwnerID)
		if err != nil {
			fmt.Println(err)
			return
		}

		// Pull Admin user, if it isn't found- continue with newly created user.
		var admin = UserNew(user.User)
		if err := admin.Get(admin.ID); err != nil {
			if err != mgo.ErrNotFound {
				fmt.Println(err)
				return
			}
		}

		// Get the Role from the running configuration.
		roleID := guildConfig.RoleIDGet(rolePermissionAdmin)
		if roleID == "" {
			fmt.Println("Error retrieving Role ID for Admin.")
			return
		}

		// Grant the role to the admin via discord.
		if err = s.GuildMemberRoleAdd(ng.ID, admin.ID, roleID); err != nil {
			fmt.Println(err)
			return
		}

		// Grant Admin permissions to server Owner.
		admin.RoleAdd(ng.ID, roleID)

		// Add the role to the admin in the database.
		if err = admin.Update(); err != nil {
			fmt.Println(err)
			return
		}

		guildConfig.Init = true

		// Update/Add guild to database and current running config.
		if err = conf.GuildConfigManager(guildConfig); err != nil {
			fmt.Println("Updating guild on new guild added: " + err.Error())
		}

		// Notify the main channel that the bot has been added and WHO has the ultimate admin privledges.
		embedMsg := embedCreator(fmt.Sprintf("Hello all, nice to meet you!\n<@%s> has been given the **Admin** privileges for <@%s> on this server.\nMy documentation is located at:\n%s\n", admin.ID, conf.Core.User.ID, helpDocs), ColorYellow)
		s.ChannelMessageSendEmbed(ng.ID, embedMsg)

		// Send a greeting to the Admin informing of the addition.
		if err := conf.dmAdmin(s, ng.OwnerID, ng.Name); err != nil {
			fmt.Println(err)
		}
	}
}

// guildRoleUpdateHandler processes updates to guild roles.
// Verifies our remain intact with correct permissions.
func (conf *Config) guildRoleUpdateHandler(s *discordgo.Session, ru *discordgo.GuildRoleUpdate) {
	// Check the current Admin/Mod roles for the guild.
	var guildConf *GuildConfig
	if guildConf = conf.GuildConfigByID(ru.GuildID); guildConf == nil {
		return
	}

	var exists bool
	var roleOld Role
	for _, role := range guildConf.Roles {
		if ru.Role.ID == role.ID {
			exists = true
			roleOld = role
			break
		}
	}

	// Not one of our roles.
	if !exists {
		return
	}

	// Validate name and minimum permissions.
	if ru.Role.Permissions&roleOld.Base != roleOld.Base || ru.Role.Name != roleOld.Name {
		// Permissions weren't included. Reupdate the role.
		roleNew, err := s.GuildRoleEdit(ru.GuildID, ru.Role.ID, roleOld.Name, ru.Role.Color, ru.Role.Hoist, ru.Role.Permissions|roleOld.Base, ru.Role.Mentionable)
		if err != nil {
			fmt.Println("Editing changed role: " + err.Error())
			return
		}

		// Update the running configuration.
		guildConf.RoleAdd(roleNew.ID, roleOld.ID, roleNew.Name, roleNew.Permissions, roleOld.Base)

		// Update the database's configuration.
		if err := guildConf.Update(); err != nil {
			fmt.Println("Updating an edited guild role: " + err.Error())
			return
		}
	}

	return
}

// guildRoleDeleteHandler processes the removal of roles.
// If ours is deleted, updates and reflects to the database.
func (conf *Config) guildRoleDeleteHandler(s *discordgo.Session, rd *discordgo.GuildRoleDelete) {
	// Check the current Admin/Mod roles for the guild.
	var guildConf *GuildConfig
	if guildConf = conf.GuildConfigByID(rd.GuildID); guildConf == nil {
		fmt.Print("Role updated, guild was nil.")
		return
	}

	var exists bool
	var roleOld Role
	for _, role := range guildConf.Roles {
		if rd.RoleID == role.ID {
			exists = true
			roleOld = role
			break
		}
	}

	// Not one of our roles.
	if !exists {
		return
	}

	// Readd the Role with proper permissions.
	roleNew, err := s.GuildRoleCreate(rd.GuildID)
	if err != nil {
		fmt.Println("Creating a deleted role: " + err.Error())
		return
	}

	// Update the newly created role with proper permissions.
	if _, err = s.GuildRoleEdit(rd.GuildID, roleNew.ID, roleOld.Name, roleNew.Color, roleNew.Hoist, roleOld.Base, roleNew.Mentionable); err != nil {
		fmt.Println("Editing newly recreated role: " + err.Error())
		return
	}

	// Update the current running configuration.
	guildConf.RoleAdd(roleNew.ID, roleOld.ID, roleOld.Name, roleOld.Value, roleOld.Base)

	// Update the configuration in the database.
	if err := guildConf.Update(); err != nil {
		fmt.Println("Updating a newly readded role: " + err.Error())
		return
	}

	return
}

// guildMemberAddHandler greets a new palyers to the channel.
func (conf *Config) guildMemberAddHandler(s *discordgo.Session, nu *discordgo.GuildMemberAdd) {
	c := conf.Core.GetMainChannel(nu.GuildID)
	msg := fmt.Sprintf("Welcome to the server, __**%s**#%s__!", nu.User.Username, nu.User.Discriminator)
	s.ChannelMessageSendEmbed(c.ID, embedCreator(msg, ColorBlue))

	tn := time.Now()
	// Add the new user to the database.
	if err := UserUpdateSimple(nu.User, 0, tn); err != nil {
		fmt.Println("Adding new user to database: " + err.Error())
	}

	for _, ch := range conf.Core.Channels {
		if ch.Name == "internal" && ch.GuildID == nu.GuildID {
			msg := fmt.Sprintf("__**%s**#%s__ [ID: %s] joined the server @ %s\n",
				nu.User.Username, nu.User.Discriminator, nu.User.ID, tn.Format(time.UnixDate))
			s.ChannelMessageSendEmbed(ch.ID, embedCreator(msg, ColorGreen))
			return
		}
	}
}

// guildMemberUpdateHandler handles newly updated user information and stores it into the database (such as additional roles.)
func (conf *Config) guildMemberUpdateHandler(s *discordgo.Session, uu *discordgo.GuildMemberUpdate) {
	// Get the user from the database.
	user := UserNew(uu.User)
	if err := user.Get(uu.User.ID); err != nil {
		if err != mgo.ErrNotFound {
			fmt.Println("Updated user, saving to database: " + err.Error())
			return
		}
	}

	var updated bool
	for n, g := range user.GuildRoles {
		if g.ID == uu.GuildID {
			updated = true
			user.GuildRoles[n].Roles = uu.Roles
		}
	}

	if !updated {
		user.GuildRoles = append(user.GuildRoles, GuildRole{ID: uu.GuildID, Name: "", Roles: uu.Roles})
	}

	if err := user.Update(); err != nil {
		fmt.Println(err)
		return
	}

	return
}

// guildMemberRemoveHandler notifies of a leaving user (NOT CURRENTLY WORKING)
func (conf *Config) guildMemberRemoveHandler(s *discordgo.Session, du *discordgo.GuildMemberRemove) {
	for _, c := range conf.Core.Channels {
		if c.Name == "internal" && c.GuildID == du.GuildID {
			tn := time.Now()
			msg := fmt.Sprintf("__**%s**#%s__ [ID: %s] left the server @ %s\n",
				du.User.Username, du.User.Discriminator, du.User.ID, tn.Format(time.UnixDate))
			s.ChannelMessageSendEmbed(c.ID, embedCreator(msg, ColorMaroon))
			return
		}
	}

	user := UserNew(du.User)

	for n, g := range user.GuildRoles {
		if g.ID == du.GuildID {
			if len(user.GuildRoles) == 1 {
				user.GuildRoles = user.GuildRoles[:0]
			} else {
				user.GuildRoles[n] = user.GuildRoles[len(user.GuildRoles)-1]
				user.GuildRoles = user.GuildRoles[:len(user.GuildRoles)-1]
			}
			break
		}
	}

	if err := user.Update(); err != nil {
		fmt.Println("User left, removing roles: " + err.Error())
		return
	}
}

// channelUpdateHandler will process existing channels that have been updated.
func (conf *Config) channelUpdateHandler(s *discordgo.Session, cu *discordgo.ChannelUpdate) {
	// If it's not the internal channel, we don't care.
	if cu.Name != "internal" {
		return
	}

	if err := conf.InternalCorrection(cu.GuildID); err != nil {
		fmt.Println("Internal Correction on updated channel: " + err.Error())
		return
	}

	return
}

// channelDeleteHandler will process the deletion of channels.
func (conf *Config) channelDeleteHandler(s *discordgo.Session, cd *discordgo.ChannelDelete) {
	core := conf.Core

	// Remove it from out channels.
	core.ChannelMemoryDelete(cd.Channel)

	// If it's not the internal channel, we don't care.
	if cd.Name != "internal" {
		return
	}

	if err := conf.InternalCorrection(cd.GuildID); err != nil {
		fmt.Println("Internal Correction on updated channel: " + err.Error())
		return
	}

	return
}

// watchLogHandler tracks if the message is under WatchLog.
func (conf *Config) watchLogHandler(guild *godbot.Guild, msg *discordgo.MessageCreate, channel string) {
	// TAG: TODO - account for watched servers that have same guild, but not same channel.
	// Verify we have a watchlogger on this guild.
	var guildID string
	var watched WatchLog
	// Check if private messages are being WatchLogged.
	if guild == nil && channel == "private" {
		guildID = "private"
		channel = ""
	} else if guild == nil {
		// Guild is nil, return to prevent accessing nil memory.
		return
	} else {
		guildID = guild.ID
	}

	// Cycle thru our WatchLogs.
	for _, w := range conf.watched {
		if w.guildID == guildID {
			watched = w
			break
		}
	}

	// Return since guild ID isn't being watched.
	if watched.guildID == "" {
		return
	}

	// If this isn't the correct channel, return.
	if strings.ToLower(watched.channelName) != strings.ToLower(channel) && !watched.channelAll {
		return
	}

	// Create output.
	output := watched.MessageCreate(msg.Author.Username, msg.Author.Discriminator, channel, msg.ContentWithMentionsReplaced())

	// Send the composed message.
	watched.Talk(output)
}
