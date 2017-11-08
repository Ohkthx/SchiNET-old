# Administrator Commands

This section is dedicated to the use of administrator commands. If you have any further questions feel free to join the [Discord Server][discord_server]!
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

Administrator commands allow dedicated and **trusted** users to interact with SchiNET. Users assigned to the `SchiNET-Administrator` role have **Administrator** access to your ENTIRE guild/server on discord- so be careful!
This is the highest avaiable access for users.

__Requirements for a Administrator command to be used:__

* Must be assigned the `SchiNET-Administrator` role in discord.
* Typed in a channel in the appropriate server/guild.

### Commands / How to Use

---

| Guides | Prefix | Argument 1 | Action |
|:------:| ------ | ------ | ------ |
| [Admin](#admin) | admin | - | Performs various admin related commands, see the guide. |
| [Script](#script) | script | - | Advanced script features. |
| [Ticket](#ticket) | ticket | - | Advanced ticket features. |

### Admin

---

Admin (administrator) commands opens up unique features to manipulate the bot for your server/guild.

Explaination of the various flags:

| Command | Argument 1 | Argument 2 | Argument 3 | Action |
| ------ | ------ | ------ | ------ | ------ |
| admin | reset | | | Reset SchiNET to the defaults. |
| admin | prefix | *[new_prefix]* | | Assign a new prefix for commands. |
| admin | nick | *[new_nick]* | | Give SchiNET a differnt NickName . |
| admin | grant | *[role type]* | *[user ID]* | Give the the user a new role. Role types: "admin" and "mod" |
| admin | channel | *[enable/disable]* | | Enable/disable SchiNET for the local channel. |

### Script

---

Scripts are text documents stored into a database that can be retrieved by users. All users can add, modify, and remove their own scripts. Administrators have the ability to remove the scripts of others in the event it is used for advertising or spam.

Normal accessible features are located at [User - Scripts][UserScripts].

Example of advanced features:

| Command | Explaination |
| ------ | ------ |
| script --remove --user d0x1p2 --title "My Script" | Removes the script uploaded by d0x1p2 titled "My Script"

### Ticket

---

Tickets are issues that can be placed in by anyone who does not have restricted SchiNET access. Administrators have several advanced features that allows the closing, noting, and removing of tickets.

Normal accessible features are located at [User - Tickets][UserTickets].

Example of advanced features:

| Command | Explaination |
| ------ | ------ |
| ticket --remove --id 0 -n "Ticket is spam." | Removes a ticket and makes note that it is spam. |
| ticket --close --id 0 -n "Ticket is resolved by rebooting." | Closes an issue and assigns a note from the administrator |
| ticket --update --id 1 --title "New Title" | Edits the title of the specified title. |

SchiNET's source is available at the [Main][Home] page!

[//]: # (Guide Links:)
[Home]: <https://github.com/d0x1p2/SchiNET/>
[MainDoc]: <https://github.com/d0x1p2/SchiNET/Main.md>
[AdminDoc]: <https://github.com/d0x1p2/SchiNET/docs/Admin.md>
[ModeratorDoc]: <https://github.com/d0x1p2/SchiNET/docs/Moderator.md>
[UserDoc]: <https://github.com/d0x1p2/SchiNET/docs/User.md>
[UserScripts]: <https://github.com/d0x1p2/SchiNET/docs/User.md#Script>
[UserTickets]: <https://github.com/d0x1p2/SchiNET/docs/User.md#Ticket>
[//]: # (Other Links:)
[discord_server]: <https://https://discord.gg/GpHDxx6>
