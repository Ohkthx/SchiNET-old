# User Commands

This section is dedicated to the use of user commands. If you have any further questions feel free to join the [Discord Server][discord_server]!
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

User commands are accessible by everyone that has **NOT** been reported for `abuse` and not in the `SchiNET-Banned`. User commands can do anything from displaying data such as events to gambling!

__Requirements for a User commands:__

* Must **NOT** be assigned the `SchiNET-Banned` role in discord.
* Typed in a channel in the appropriate server/guild.

### Commands / How to Use

---

| Guides | Prefix | Argument 1 | Action |
|:------:| ------ | ------ | ------ |
| [User](#user) | user | - |  Performs various user related commands. |
| [Script](#script) | script | - | Various interactions with the Script Library. |
| [Ticket](#ticket) | ticket | - | Actions to generate a trouble ticket. |
| | invite | | Displays information needed to invite the bot to another server. |
| [Event](#event) | events | | Displays all currently scheduled events for your server. |
| | xfer | *[@mention]* | Transfers credits from yourself to the user. |
| | gamble | *[amount]* | Gamble the amount of credits, accepts "all" |
| | top10 | | Checks if you're elite enough to be in the top 10! |
| | roll | | Rolls 2 6D, highest roll wins. |
| | echo | *[insert text here]* | Repeats (echos) the text from SchiNET! |

### User

---

User commands are designed to manipulate users. Not all of them will be accessible to regular users, the ones that can will be labelled with an asterisk(*).
Some of these commands are aliased to be easier to access, feel free to check out [Aliases](#aliases) for some additional information.

Explaination of the various flags:

| Flag | Long Flag | Action |
| ------ | ------ | ------ |
|  | --user | User to perform the action on. |
| -x | --xfer | Transfer credits to another user. |
| -g | --gamble | Gamble your credits! |
| -n | | Amount of credits you wish to transfer/gamble. |
| | --all | Gamble all of your credits. |
| -h | --help | Displays the quick-access help menu. |
|  | --list | Like all Abusers |

Examples:

| Command | Explaination |
| ------ | ------ |
| user | Check your profile/statistics- will display your credits.
| user --gamble -n 200 | Gambles 2o0 of your credits for a chance to double the amount gambled (2% to triple). |
| user --xfer --user *[@mention]* -n 15134 | Transfers 15134 credits to the user you mentioned.|

### Script

---

Scripts are documents that you can upload and retrieve from SchiNET. They are managed by the user who uploads them and Moderators and Admins can remove them at any given time. Requesting a script from SchiNET will have it posted straight to [Pastebin](https://pastebin.com/) and provide a link to the paste to access. Scripts will last for 10minutes on Pastebin and then be removed. Once they're removed- they have to be requested again.

If you're looking for advance script managing, be sure to check out the documentation for [Administrator - Scripts][AdminScripts].

Explaination of the various flags:

| Flag | Long Flag | Action |
| ------ | ------ | ------ |
|| --add |  Add a script to the library. |
|| --edit | Edit an existing script you've created. |
|| --remove | Remove an existing script you've created. |
| -g | --get | Requests to retrieve a script from the library. |
| -u | --user | Who the script's owner is, just use their username. |
| -t | --title | Title of the Script to Add/Edit/Remove/Request. |
| -v | --version | Add a version to a script while adding/editing.
| -l | --list | List all scripts that currently exist in the database. |
| -h | --help | Displays help information similar to this.

Examples:

| Command | Explaination |
| ------ | ------ |
| script --add --title "My Script" *[attached .txt file]* | Creates a script in the library called "My Script" with the information from the attached file. |
| script --edit --title "My Script" *[attached .txt file]* | Edits "My Script" with new information provided in the text file. |
| script --remove --title "My Script" | Removes the script you created. |
| script --add -v 2.1 -t "Tester" *[attached .txt file]* | Creates a script named "Tester" with the version of 2.1 |
| script --list | List all available scripts in the library. |
| script --get -t "Tester" --user d0x1p2 | Gets a script called "Tester" that was uploaded by "d0x1p2"

### Ticket

---

Tickets can be placed into the system for Administrators and Moderators to view and resolve issues. This is helpful for communities based on developing features and expanding their platforms. Often times things start snowballing out of control and issues are forgotten when simply just stated in a text message. This service allows them to be added, edited, and closed to keep track of things!

If you're looking for advance ticket managing, be sure to check out the documentation for [Administrator - Tickets][AdminTickets].

Explaination of the various flags:

| Flag | Long Flag | Action |
| ------ | ------ | ------ |
|| --add | Add a new ticket to the system. |
|| --update | Update a tickets title, comment, or note. |
|| --close | Close a resolved ticket. |
|| --remove | Remove a ticket (use this for spam) |
|| --get | Get a trouble ticket |
|| --id | Provide the ID of the Ticket. |
| -t | --title | Provide a title of/for a ticket. |
| -c | --comment | Provide a comment regarding the ticket. |
| - n | --note | Allows Administrators to place notes on a ticket. |
|| --list | List all open tickets.|
| -h | --help | Displays a help message similar to this. |

Examples of adding a ticket:

 |Command | Explaination |
| ------ | ------ |
| ticket --add -t "Auto-Logout" -c "When not performing an action for 5min, it is auto-logging me out" |Creates a ticket named "Auto-Logout" with the description provided by '-c' |
| ticket --list | Lists all tickets. |
| ticket --get --id 0 | Gets the ticket with the ID of 0. |
| ticket --remove --id 0 -n "Bad information." | You can remove tickets that **YOU** have created, otherwise requires an administrator. |

### Event

---

Events are scheduled timers that are created by Moderators. It can be anything from birthday parties to guild meetings! Without escalated permissions, you can only view events.
If you're looking for advance event managing, be sure to check out the documentation for [Moderators - Events][ModEvents].

| Command | Explaination |
| ------ | ------|
| events | List all events current scheduled. |

### Alias

---

Aliases are shortened commands that are most likely a commonly used command. SchiNET has several built-in commands that are listed below.

 |Command | Executes | Explaination |
| ------ | ------ | ------ |
| me | user | Prints data regarding you. |
| gamble | user --gamble -n | Shorthand and allows for easier gambling. |
| xfer | user --xfer | Helps with transferring of credits. |
| abuse | user --abuse --user | Easy access to restricting a users ability abuse the bot. |

SchiNET's source is available at the [Main][Home] page!

[//]: # (Guide Links:)
[Home]: <https://github.com/d0x1p2/SchiNET/>
[MainDoc]: <https://github.com/d0x1p2/SchiNET/blob/master/docs/Main.md>
[AdminDoc]: <https://github.com/d0x1p2/SchiNET/blob/master/docs/Admin.md>
[ModeratorDoc]: <https://github.com/d0x1p2/SchiNET/blob/master/docs/Moderator.md>
[UserDoc]: <https://github.com/d0x1p2/SchiNET/blob/master/docs/User.md>
[ModEvents]: <https://github.com/d0x1p2/SchiNET/blob/master/docs/Moderator.md#Event>
[AdminTickets]: <https://github.com/d0x1p2/SchiNET/blob/master/docs/Admin.md#Ticket>
[AdminScripts]: <https://github.com/d0x1p2/SchiNET/blob/master/docs/Admin.md#Script>
[//]: # (Other Links:)
[discord_server]: <https://https://discord.gg/GpHDxx6>
