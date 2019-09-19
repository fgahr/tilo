// A simple time logging program.
package main

import (
	"github.com/fgahr/tilo/command"
	_ "github.com/fgahr/tilo/command/start"
	"github.com/fgahr/tilo/config"
	"log"
	"os"
)

// Print usage information for this program.
func printUsage() {
	// TODO
}

// Initiate server or client operation based on given arguments.
func main() {
	args := os.Args[1:]
	if len(args) == 0 {
		printUsage()
		os.Exit(1)
	}

	// TODO: Parse config-related options?
	conf, err := config.DefaultConfig()
	if err != nil {
		log.Fatal(err)
	}

	command.ExecuteClient(conf, args)
}
