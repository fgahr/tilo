// A simple time logging program.
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
	executable := os.Args[0]
	// TODO: Rename variables/functions in command.go and query.go to match.
	fmt.Printf("Usage: %s <command> [task-names] [parameters]\n", executable)
	fmt.Print(`
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
	listen              Register as a notification listener, print notifications

Query command: query <tasks> <conf> Query the database
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
`)
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
		os.Exit(1)
	}

	if os.Args[1] == "-h" || os.Args[1] == "--help" {
		printUsage()
		os.Exit(0)
	}

	conf, err := config.DefaultConfig()
	if err != nil {
		log.Fatal(err)
	}

	// NOTE: This is mostly for debugging purposes.
	if os.Args[1] == "listen" {
		printServerNotifications(conf)
	}

	// "server run" and "server start" do not involve requests
	if len(args) > 1 && args[0] == "server" && args[1] == "run" {
		signal.Ignore(syscall.SIGHUP)
		err = server.Run(conf)
	} else if len(args) > 1 && args[0] == "server" && args[1] == "start" {
		err = server.StartInBackground(conf)
	} else {
		err = handleClientArgs(args, conf)
	}
	if err != nil {
		log.Fatal(err)
	}
}

// Print server notifications to stdout.
func printServerNotifications(conf *config.Params) {
	c, err := client.NewClient(conf)
	if err != nil {
		log.Fatal(err)
	}
	if err := c.PrintNotifications(os.Stdout); err != nil {
		log.Fatal(err)
	}
	os.Exit(0)
}

// Handle client functionality, parsing the relevant arguments.
func handleClientArgs(clientArgs []string, conf *config.Params) error {
	requireArgs(clientArgs, 1)
	c, err := client.NewClient(conf)
	if err != nil {
		return err
	}
	// Note: Ignoring possible error.
	defer c.Close()

	return c.HandleArgs(clientArgs)
}
