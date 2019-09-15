# tilo
A simple time logging system written in Golang, backed by SQLite3, with
client/server operation. Server and client communicate through a
Unix domain socket, so windows will not work. Developed and tested on Linux
but other unix-likes might work, too. There is no reason communication
can't be done through a tcp socket and it's easy to change. Hopefully, this
will be configurable at some point.

The socket is `/tmp/tilo$UID/server` when the server is running, where the
directory is accessible for the operating user only. For now this is the only
authentication method employed. This scheme is inspired by emacs. In case of
an unrecovered panic the server may fail to clean up the temporary directory.
Either remove it by hand or use the cleanup script at the repository root.

For now, `tilo` is mainly meant as a personal learning project and is very much
incomplete. That being said, I intend to use it and fix/improve it as necessary.
Feel free to point out any mistakes I made.

# Listeners
To be notified about task changes, server shutdown, etc. a program can establish
a connection to the `notify` socket (next to the server socket). This is intended
to be used by other programs that depend on the current task. The format is not
yet set in stone.

Sample output can be gathered with the `tilo listen` command. This way it can also
be used in e.g. shell scripts.

# Bugs
There are a few that I'm aware of and many more yet unbeknownst to me. Feel
free to find them and let me know. There may already be a `FIXME` in the code.
If you want to help me squash them, feel free to.

# Installation
As mentioned above, your best bet is running on Linux. Make sure you have
`go` installed. Then
```
go get -u -v github.com/fgahr/tilo
```

# Usage
The program's help message:
```
Usage: tilo <command> [task-names] [parameters]

Available commands:
Help:
    -h|--help      Print this message

Server commands:
    server run          Start the server in the foreground
    server start        Start the server in the background

Simple commands (may start server in background):
    start <task>        Start logging time for the given task
    stop                Stop the current task, log the time
    abort               Stop the current task without logging it
    shutdown            Shut down the server. The current task will be logged
    listen              Register as a notification listener, print notifications.

Query command: query <tasks> <params> Query the database
    tasks: A comma-separated list of task names (no spaces!), --all to get all tasks

Unquantified parameters:
    --today             Today's activity
    --yesterday         Yesterday's activity
    --ever              All recorded activity
    --(this|last)-week  This|Last week's activity
    --(this|last)-month This|Last month's activity
    --(this|last)-year  This|Last year's activity

Quantified parameters (can take several, comma-separated quantifiers):
    --day=YYYY-MM-DD    Activity on the given day
    --month=YYYY-MM     Activity in the given month
    --year=YYYY         Activity in the given year
    --weeks-ago=N       Activity in the Nth past week (0 => current week)
    --months-ago=N      Activity in the Nth past month (0 => current month)
    --years-ago=N       Activity in the Nth past year (0 => current year)
    --since=YYYY-MM-DD  Activity since the given day
    --between=d1,d2     Activity between two days, each given as YYYY-MM-DD
```

# GUI
A system tray icon displaying the current server status can be found here:
https://github.com/fgahr/tilo-systray
