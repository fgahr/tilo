// Package for commands to the server
package cmd

import (
	"github.com/fgahr/tilo/msg"
	"github.com/fgahr/tilo/server"
	"github.com/pkg/errors"
)

var operations map[string]operation

type Cmd struct {
	Op   string     `json:"operation"` // The operation to perform
	Body [][]string `json:"body"`      // The body containing the command information
}

// TODO: Add help text information and create help message dynamically
type operation interface {
	// Perform this operation on the server, based on the contents of cmd.
	perform(cmd Cmd, srv *server.Server, resp *msg.Response) error
	// Encapsulate command line arguments as a command for this operation.
	parseArguments(args []string) (Cmd, error)
}

func registerOperation(command string, op operation) {
	if operations[command] != nil {
		panic("Double registration of operations with identical command")
	}
	operations[command] = op
}

func Parse(args []string) (Cmd, error) {
	if len(args) == 0 {
		panic("Empty argument list")
	}
	command := args[0]
	op := operations[command]
	if op == nil {
		panic("No such command: " + command)
	}
	return op.parseArguments(args[1:])
}

func ExecuteCommand(cmd Cmd, srv *server.Server, resp *msg.Response) error {
	op := operations[cmd.Op]
	if op == nil {
		return errors.Errorf("No such operation: %s", cmd.Op)
	}
	return errors.Wrap(op.perform(cmd, srv, resp), "Command execution failed")
}
