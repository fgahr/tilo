// A simple time logging program.
package main

import (
	"fmt"
	"os"

	"github.com/fgahr/tilo/client"
	_ "github.com/fgahr/tilo/command/abort"
	_ "github.com/fgahr/tilo/command/current"
	_ "github.com/fgahr/tilo/command/help"
	_ "github.com/fgahr/tilo/command/listen"
	_ "github.com/fgahr/tilo/command/ping"
	_ "github.com/fgahr/tilo/command/query"
	_ "github.com/fgahr/tilo/command/resume"
	_ "github.com/fgahr/tilo/command/shutdown"
	_ "github.com/fgahr/tilo/command/srvcmd"
	_ "github.com/fgahr/tilo/command/start"
	_ "github.com/fgahr/tilo/command/stop"
	"github.com/fgahr/tilo/config"
	_ "github.com/fgahr/tilo/server/backend/sqlite3"
)

// Initiate server or client operation based on given arguments.
func main() {
	args := os.Args[1:]

	conf, restArgs, err := config.GetConfig(args, os.Environ())
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	if client.Dispatch(conf, restArgs) {
		os.Exit(0)
	} else {
		os.Exit(1)
	}
}
