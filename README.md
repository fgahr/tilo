# tilo
A simple time logging system written in Golang, backed by SQLite3, with
client/server operation. Communication happens by sending JSON-encoded
command and response messages back and forth.

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
As a major overhaul with a more dynamic help system is underway, the program
currently has no help message. As soon as I figure out how I want to implement
this and get around to actually doing that, this situation will improve. The
output will then be included here.

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
- `undo`: Undo a logged task, ideally with interactive choice
- `log`: Save a new log entry, in case you forgot to start the timer
- ...
## Other
- Dynamic help system
- Declarative argument parsing
- Configuration based on command-line parameters, environment variables, and a
  config file (currently: none)
