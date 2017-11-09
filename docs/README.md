# SchiNET

SchiNET is a discord bot written in Go (Golang) that provides a helpful interface for Administrators, Moderators, and the everyday User! The bot is in frequent development and depends on several others' hardwork and dedication to open-sourced software. What to see the projects that made this happen? Scroll down!

Curious how to perform some commands? Check out the guides linked in the **Table of Contents** or join the [Discord Server][discord_server]!

## Table of Contents

| Access Type | Guide  |
| ------ | ------ |
|------------- | [docs/README.md][MainDoc] |
| Admin | [docs/Admin.md][AdminDoc] |
| Moderator | [docs/Moderator.md][ModeratorDoc] |
| User | [docs/User.md][UserDoc] |

## Latest News and Additions

* Project renamed to SchiNET
* SchiNET can be renamed using:  `,admin nick [new_nick]`
* Commands Enabling/Disabling in local channel:  `,admin channel [enable/disable]`
* Automated Role Management: Recreates/modifies discord roles that are used by SchiNET
* Automated Channel Management: Recreates/modifies `#internal` channel
* Banning removed, instead `,abuse [@mention]` has taken it's place to ban abuses from bot commands.

### This where the Magic happens

If it wasn't for these many projects (and even more contributors)- none of this would be possible. Thanks if you're reading this!

* [discordgo] - Discord's API Bindings for Go.
* [mgo] - Helpful driver for using MongoDB in Go.
* [go_pastebin] - Allows for pasting Scripts to the interwebs!
* [getopt] - Provides helpful command parsing for our many commands.
* [godbot] - Core of the bot, handles a bunch of the behind-the-scenes.
* [original] - First (failed) version of the project, subject to termination!

SchiNET's source is available at the [Main][Home] page!

#### License

MIT

[//]: # (These are reference links used in the body of this note and get stripped out when the markdown processor does its job. There is no need to format nicely because it shouldn't be seen. Thanks SO - http://stackoverflow.com/questions/4823468/store-comments-in-markdown-syntax)
[//]: # (Guide Links:)
[Home]: <https://github.com/d0x1p2/SchiNET/>
[MainDoc]: <https://github.com/d0x1p2/SchiNET/blob/master/docs/README.md>
[AdminDoc]: <https://github.com/d0x1p2/SchiNET/blob/master/docs/Admin.md>
[ModeratorDoc]: <https://github.com/d0x1p2/SchiNET/blob/master/docs/Moderator.md>
[UserDoc]: <https://github.com/d0x1p2/SchiNET/blob/master/docs/User.md>
[//]: # (Projects:)
[discordgo]: <https://github.com/bwmarrin/discordgo>
[mgo]: <https://github.com/go-mgo/mgo>
[go_pastebin]: <https://github.com/glaxx/go_pastebin>
[getopt]: <https://github.com/pborman/getopt>
[godbot]: <https://github.com/d0x1p2/godbot>
[original]: <https://github.com/d0x1p2/DiscordBot-go>
[//]: # (Other Links:)
[discord_server]: <https://discord.gg/GpHDxx6>
