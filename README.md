### Ultima-Shards Discord Bot
+ Discord API: [bwmarrin/discordgo](https://github.com/bwmarrin/discordgo)
+ ~~MySQL Drivers: [my-sql-driver/mysql](https://github.com/go-sql-driver/mysql)~~
+ MongoDB Drivers: [go-mgo/mgo](https://github.com/go-mgo/mgo)
+ Pastebin API: [glaxx/go_pastebin](https://github.com/glaxx/go_pastebin)
+ Getopt: [pborman/getopt](https://github.com/pborman/getopt)
+ Vita-Nex: Core API: [d0x1p2/vncgo](https://github.com/d0x1p2/vncgo)
+ Discord Bot: Core: [d0x1p2/godbot](https://github.com/d0x1p2/godbot)
+ Original: [d0x1p2/DiscordBot-go](https://github.com/d0x1p2/DiscordBot-go)

Special thanks to the wonderful developers above and their projects that has given me the ability to write this.

This is the second Discord bot that I've written with Go and using the amazing Discord API provided at the link above. The first project quickly snowballed into something much larger that initially anticipated. Being my first project in Go, I've learned many practices that are cleaner- hence the birth of the Discord Bot: Core project.

### Features
+ MongoDB support.
+ Gambling support, based on message count.
+ Script library.
+ Pastebin posting via API.
+ User Soft/Hard bans.
+ Aliases to commands.
+ Server Message Histograms.
+ Permissions for bot manipulation.

### TODO/Reimplement List
+ Support for Vita-Nex: Core API
+ ~~Support for SQL~~
+ Public and Private Notifications
+ Event Timers and Tracking
+ ~~Message Count Gambling~~
+ Move ENV variables to CFG/INI file.
+ ~~Event Handlers for creating/deleting channels~~
+ ~~New player greeting.~~
+ ~~Softbans from channels.~~
+ ~~Hardbands from server/guild.~~
+ ~~Last seen to Users~~
+ ~~Check user existing in Database.~~
+ Parsing in-game messages and sending to discord.
+ ~~Message -> Database integrity check.~~
+ ~~Discord Guild -> Guild chat.~~
+ Script Library: Accessing via IDs

### Versions
See [changelog](https://github.com/d0x1p2/usmbot/blob/master/changelog)