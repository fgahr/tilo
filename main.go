// Package determines whether to drive the application in client or server mode.
package main

import (
	"fmt"
	"github.com/fgahr/tilo/client"
	"github.com/fgahr/tilo/config"
	"github.com/fgahr/tilo/server"
	"log"
	"os"
	"os/signal"
	"syscall"
)

// Print usage information for this program.
func printUsage() {
	fmt.Println("You are doing it wrong.")
}

// Make sure there are enough at least num arguments.
func requireArgs(args []string, num int) {
	if len(args) < num {
		printUsage()
		os.Exit(1)
	}
}

// Initiate server or client operation based on given arguments.
func main() {
	args := os.Args[1:]
	if len(args) == 0 {
		printUsage()
		os.Exit(0)
	}

	params, err := config.DefaultParams()
	if err != nil {
		log.Fatal(err)
	}
	// "server run" and "server start" do not involve requests
	if len(args) > 1 && args[0] == "server" && args[1] == "run" {
		signal.Ignore(syscall.SIGHUP)
		err = server.Run(params)
	} else if len(args) > 1 && args[0] == "server" && args[1] == "start" {
		err = server.StartInBackground(params)
	} else {
		err = handleClientArgs(args, params)
	}
	if err != nil {
		log.Fatal(err)
	}
}

// Handle client functionality, parsing the relevant arguments.
func handleClientArgs(clientArgs []string, params *config.Params) error {
	requireArgs(clientArgs, 1)
	c, err := client.NewClient(params)
	if err != nil {
		return err
	}
	// Note: Ignoring possible error.
	defer c.Close()

	return c.HandleArgs(clientArgs)
}
