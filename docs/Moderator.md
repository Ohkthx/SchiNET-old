# Moderator Commands

This section is dedicated to the use of moderator commands. If you have any further questions feel free to join the [Discord Server][discord_server]!
*Reminder, all commands must be prefixed with your server/guild's assigned prefix, default: ,

## Table of Contents

---
| Access Type | Guide  |
| ------ | ------ |
|------------- | [docs/Main.md][MainDoc] |
| Admin | [docs/Admin.md][AdminDoc] |
| Moderator | [docs/Moderator.md][ModeratorDoc] |
| User | [docs/User.md][UserDoc] |

## About

---

Moderator commands grant  users the ability to manipulate SchiNET in a way that is overall healthy for the community. Many new doors are opened up to those selected to become a Moderator. However, be careful- this is the **2nd** highest access for a user utilising SchiNET.

__Requirements for a Moderator command to be used:__

* Must be assigned the `SchiNET-Moderator` role in discord.
* Typed in a channel in the appropriate server/guild.

### Commands / How to Use

---

| Guides | Prefix | Argument 1 | Action |
|:------:| ------ | ------ | ------ |
| | abuse | *[@mention]* | Restricts the @mentioned user from being able to use the bot.
| [Events](#events) | event | - | Manage events for the server. |
| [Aliases](#aliases) | alias | - | Manage various aliases. |
| | clear | | **Fast** clear messages, leverages "bulk deletion" but has restrictions. |
| | clear-slow | | Slow clear messages, deletes each message individually. No restrictions. |
| [Ally](#ally) | ally | - | Allows the linking of 2 servers/guilds through a common channel. |

### Events

---

As a moderator, you can create events in which a timer will be set and the event will countdown. This is helpful for international servers interested in coordinating various events without the constant conversion of timezones.

Explaination of the various flags:

| Flag | Long Flag | Action |
| ------ | ------ | ------ |
|  | --add | Add an event for the Server/Guild |
|  | --remove | Remove an event. |
| -p | --persist | Used when adding a event to automatically reschedule the event to reoccur the following week. |
| -d | --day | Day of the week the event is to happen: Monday, Tuesday, Wednesday, Thursday, Friday, Saturday, or Sunday |
| -t | --time | Time the event will be occuring, **accepts 24hr** format: 8am = 0700, 1PM = 1300,  10:22PM = 2222 |
| -c | --comment | Add a commentto the event to explain exactly what it is. |
| -h | --help | Prints out a readily accessible "help" to describe what can be performed. |
| -l | --list | Lists all events that are scheduled.

Examples:

| Command | Explaination |
| ------ | ------ |
| event --add --persist --day Monday --time 1301 -c "Raid time!" | Creates an event that will reoccur every Monday at 1:01PM. |
| event --remove -d Monday -t 1301 | Removes the event that was created above. |
| event --add -d Thursday -t 0800 -c "Guild Meeting!" | Schedules an event to occur a single time on Thursday at 8am. |
| event --list | List all events for the Server/Guild |

### Aliases

---

Aliases are links to other commands. Using an alias just saves typing common commands and can be used to performed customized actions.

Explaination of the various flags:

| Flag | Long Flag | Action |
| ------ | ------ | ------ |
| -a | --add | Adds an alias |
| -r | --remove | Removes and alias |
| -i | | What will be inputted |
| -o | | What will be performed (outputted/original) |
| -h | --help | Prints out a help message, quick reference. |
| -l | --list | List all aliases current assigned.

Examples:

| Command | Explaination |
| ------ | ------ |
| alias --add -i hw -o "echo Hello World!" | **hw** will now print out **"Hello World!"** |
| alias -r -i hw | Removes the **hw** alias created above. |

### Ally

---

Ally forms a common link between **TWO** different discord servers/guilds. This common link is in the form of a channel. What happens is that once the alliace is formed, any message typed in the channel created will automatically be replicated to the allied server/guild. It allows for **two-way** communication without people having to be in both.

Alliances can be broken and the channel will automatically be removed from both servers/guilds.

Explaination of the various flags:

| Flag | Long Flag | Action |
| ------ | ------ | ------ |
| | --init | Initiate an alliance |
| -d | --delete | Break/Remove an alliance |
| -n | --name | The name of the alliance |
| -k | --key | Key for establishing an alliance |
| -l | --list | List all guilds available. |
|-h | --help | Displays a quick help on what all can be done. |

Example of Creating an alliance using 2 guilds, Guild1 and Guild2:

| Guild/Server |Command | Explaination |
| ------ | ------ | ------ |
| Guild1 | ally --init --name "Our_Alliance" | Initiates an alliance named "Our_Alliance" |
| Guild2 | ally --key *[key_here]*  --name "Our_Alliance" | The key will be created on Guild1 and needs to be used here. |

Example of Breaking an alliance using 2 guilds, requires only 1 guild:

| Guild/Server |Command | Explaination |
| ------ | ------ | ------ |
| Guild1 | ally --delete --name "Our_alliance" | Destroys the alliance on both ends regardless of which guild does it. |

SchiNET's source is available at the [Main][Home] page!

[//]: # (Guide Links:)
[Home]: <https://github.com/d0x1p2/SchiNET/>
[MainDoc]: <https://github.com/d0x1p2/SchiNET/blob/master/docs/Main.md>
[AdminDoc]: <https://github.com/d0x1p2/SchiNET/blob/master/docs/Admin.md>
[ModeratorDoc]: <https://github.com/d0x1p2/SchiNET/blob/master/docs/Moderator.md>
[UserDoc]: <https://github.com/d0x1p2/SchiNET/blob/master/docs/User.md>
[//]: # (Other Links:)
[discord_server]: <https://https://discord.gg/GpHDxx6>
