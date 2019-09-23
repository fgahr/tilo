// A simple time logging program.
package main

import (
	"github.com/fgahr/tilo/client"
	"github.com/fgahr/tilo/command"
	_ "github.com/fgahr/tilo/command/current"
	_ "github.com/fgahr/tilo/command/listen"
	_ "github.com/fgahr/tilo/command/ping"
	_ "github.com/fgahr/tilo/command/srvcmd"
	_ "github.com/fgahr/tilo/command/start"
	_ "github.com/fgahr/tilo/command/stop"
	"github.com/fgahr/tilo/config"
	"log"
	"os"
)

// Initiate server or client operation based on given arguments.
func main() {
	args := os.Args[1:]
	if len(args) == 0 {
		command.PrintAllOperationsHelp()
		os.Exit(1)
	}

	if args[0] == "-h" || args[0] == "--help" {
		command.PrintAllOperationsHelp()
		os.Exit(0)
	}

	// TODO: Parse config-related options, read environment/config file
	conf, err := config.DefaultConfig()
	if err != nil {
		// TODO: Consider printing without timestamp
		log.Fatal(err)
	}

	if err := client.Dispatch(conf, args); err != nil {
		// TODO: Consider printing without timestamp
		log.Fatal(err)
	}
}
