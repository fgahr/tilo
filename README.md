# tilo
A simple time logging system written in Golang, backed by SQLite3, with
client/server operation. Communication happens by sending JSON-encoded
command and response messages back and forth.

Admittedly, this design already incurs considerable complexity cost. For a
simpler approach see [olit](https://github.com/fgahr/olit).

# Installation
Your best bet is running on Linux, Windows is very unlikely to work. Make sure
you have `go` installed. Then the following command will create the binary in
your `$GOPATH/bin`.
```
go get -u -v github.com/fgahr/tilo
```

# Purpose
For now, `tilo` is mainly meant as a personal learning project and is very much
incomplete. That being said, I intend to use it and fix/improve it as necessary.
Feel free to point out any mistakes I made.

# Usage
`tilo help` gives usage information, either as a summary of all commands or
detailed information about a particular command:
```
# tilo help

Usage: tilo [command] <task(s)> <parameters>

Available commands
    abort                                Abort the currently active task without saving
    current                              See which task is currently active
    help      <command>                  Describe program or detailed usage of a command
    listen                               Listen for and print server notifications
    ping                                 Ping the server
    query     [task,..]    [parameters]  Make enquiries about prior activity
    server    [start|run]                Start a server in the background/foreground
    shutdown                             Request server shutdown
    start     [task]                     Start logging activity on a task
    stop                                 Stop and save the currently active task
```

The `query` command is currently the most complex. Its usage is as follows:
```
# tilo help query
Usage: tilo query [task,..] [parameters]

Get information about recorded activity

Required task information
    [task,..]  One or more task names, separated by comma; :all to select all tasks

Possible parameters
    :between     YYYY-MM-DD:YYYY-MM-DD,...  Activity between two dates
    :day         YYYY-MM-DD,...             Activity on a given day
    :days-ago    N,...                      Activity N days ago
    :last-month                             Last month's activity
    :last-week                              Last week's activity
    :last-year                              Last year's activity
    :month       YYYY-MM,...                Activity in a given month
    :months-ago  N,...                      Activity N months ago
    :since       YYYY-MM-DD,...             Activity since a specific day
    :this-month                             This month's activity
    :this-week                              This week's activity
    :this-year                              This year's activity
    :today                                  Today's activity
    :weeks-ago   N,...                      Activity N weeks ago
    :year        YYYY,...                   Activity in a given year
    :years-ago   N,...                      Activity N years ago
    :yesterday                              Yesterday's activity

Where indicated, a list of quantifiers (or pairs thereof) can be given
Parameters can be freely combined and repeated in a single query

Examples
    tilo query :all :this-week                    # This week's activity across all tasks
    tilo query foo :between 2019-01-01:2019-06-30 # Logged on task foo in first half of 2019
    tilo query bar :month=2019-01,2019-02,2019-03 # Activity for bar in three different months
```

# Details
Server and client communicate through a Unix domain socket, so windows will
not work. Developed and tested on Linux but other unix-likes might work, too.
There is no reason communication can't be done through a tcp socket and it's
easy to change. Hopefully, this will be configurable at some point.

The socket is `/tmp/tilo$UID/server` when the server is running, where the
directory is accessible for the operating user only. For now this is the only
authentication method employed. This scheme is inspired by emacs. In case of
an unrecovered panic the server may fail to clean up the temporary directory.
Either remove it by hand or use the cleanup script at the repository root.

# Listeners
To be notified about task changes, server shutdown, etc. a program can send a
`listen` command. The connection is then kept open and the listener is fed with
information about task changes and server shutdown.

Sample output can be gathered with the `tilo listen` command. This way it can also
be used in e.g. shell scripts.

# Configuration
Configuration is possible, in ascending priority, via a configuration file,
environment variables, and command line arguments. The configuration file is
typically located under `~/.config/tilo/config` but another file can be chosen
via command line or environment variables.

When a server is started in a background process, all configuration is passed
via the process environment. For a foreground server process, all three ways are
available.

For now there are not a lot of options available. Documentation will follow when
things get more interesting.

# Bugs
There are a few that I'm aware of and many more yet unbeknownst to me. Feel
free to find them and let me know. There may already be a `FIXME` in the code.
If you want to help me squash them, feel free to.

# GUI
A system tray icon displaying the current server status can be found here:
https://github.com/fgahr/tilo-systray

# TODOs
## New commands:
- `recent`: Shows a given number of recently logged tasks
- `undo`: Delete one or several logged tasks, ideally with interactive choice
- `log`: Save a new log entry, in case you forgot to start the timer
- ...
## Other
- Bash/Zsh completion of task names and parameters
- Different output options (CSV, JSON, ...)
- Support for TCP connections, remote server
