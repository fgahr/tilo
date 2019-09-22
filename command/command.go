// Package for commands to the server
package command

import (
	"github.com/fgahr/tilo/config"
	"github.com/fgahr/tilo/msg"
	"github.com/fgahr/tilo/server"
	"github.com/pkg/errors"
	"net"
)

var operations = make(map[string]Operation)

type Cmd struct {
	Op    string            `json:"operation"` // The operation to perform
	Flags map[string]bool   `json:"flags"`     // Possible flags
	Opts  map[string]string `json:"options"`   // Possible options
	Body  [][]string        `json:"body"`      // The body containing the command information
}

type Doc struct {
	ShortDescription string
	LongDescription  string
	Arguments        []string
	// TODO: Define proper structure
}

// TODO: Add help text information and create help message dynamically
type Operation interface {
	// The command identifier
	Command() string
	// Execute client-side behaviour based on args
	ClientExec(conf *config.Opts, args ...string) error
	// Execute server-side behaviour based on the command
	// NOTE: That's a lot of arguments. If it gets any more than that, set them as the operation's state.
	ServerExec(srv *server.Server, conn net.Conn, cmd Cmd, resp *msg.Response)
	// Documentation for this operation
	Help() Doc
}

func RegisterOperation(op Operation) {
	if operations[op.Command()] != nil {
		panic("Double registration of operations with identical command")
	}
	operations[op.Command()] = op
}

func ExecuteClient(conf *config.Opts, args []string) error {
	if len(args) == 0 {
		panic("Empty argument list")
	}
	command := args[0]
	op := operations[command]
	if op == nil {
		panic("No such command: " + command)
	}
	return op.ClientExec(conf, args[1:]...)
}

func ExecuteServer(srv *server.Server, conn net.Conn, cmd Cmd) (msg.Response, error) {
	resp := msg.Response{}
	command := cmd.Op
	op := operations[command]
	if op == nil {
		return resp, errors.New("No such operation: " + command)
	}
	op.ServerExec(srv, conn, cmd, &resp)
	return resp, nil
}

func PrintSingleOperationHelp(op Operation) {
	// TODO
}

func PrintAllOperationsHelp(op Operation) {
	// TODO
}
