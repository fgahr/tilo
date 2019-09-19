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
	// Execute client-side behaviour based on args
	clientExec(args ...string) error
	// Execute server-side behaviour based on the command
	serverExec(srv *server.Server, cmd Cmd, resp *msg.Response) error
	// TODO: require more structure?
	// Documentation for this operation
	doc() string
}

func registerOperation(command string, op operation) {
	if operations[command] != nil {
		panic("Double registration of operations with identical command")
	}
	operations[command] = op
}

func ExecuteClientOperation(args []string) error {
	if len(args) == 0 {
		panic("Empty argument list")
	}
	command := args[0]
	op := operations[command]
	if op == nil {
		panic("No such command: " + command)
	}
	return op.clientExec(args[1:]...)
}

func ExecuteServerOperation(srv *server.Server, cmd Cmd) (msg.Response, error) {
	resp := msg.Response{}
	command := cmd.Op
	op := operations[command]
	if op == nil {
		return resp, errors.New("No such operation: " + command)
	}
	err := op.serverExec(srv, cmd, &resp)
	return resp, err
}

func DescribeOperations() string {
	// TODO: Define operation groups?
	return "TODO"
}
