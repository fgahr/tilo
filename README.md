# tilo
A simple time logging system written in Golang, backed by SQLite3, with
client/server operation.

This is mainly meant as a personal learning project and is very much incomplete.
That being said, I intend to use it and fix/improve it as necessary. Feel free
to point out any mistakes I made.

# Bugs
There are a few that I'm aware and many more yet unbeknownst to me. Feel free
to find them and let me know. There may already be a `FIXME` in the code.
If you want to help me squash them, feel free to.

# Installation
Developed and tested only on Linux.
Server and client communicate through a
Unix domain socket, so windows will not work. Other unix-likes might. Make sure
you have `go` installed. Then
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

# Future Features
Since this is a project for personal learning, I may add more features than
are necessary.

## GUI
There is already a branch which features a system tray widget to display the
server's current status (logging|not logging). But since qt is somewhat
unwieldy in terms of installation, compilation, and resulting binary size, this
is not on the master branch.
