// Package for commands to the server
package command

import (
	"fmt"
	"github.com/fgahr/tilo/client"
	"github.com/fgahr/tilo/server"
	"os"
)

var opNames = make(map[string]bool)

// TODO: Define proper structure
type Doc struct {
	ShortDescription string
	LongDescription  string
	Arguments        []string
}

type Operation interface {
	client.ClientOperation
	server.ServerOperation
	// The command identifier
	Command() string
	// Documentation for this operation
	Help() Doc
}

func RegisterOperation(op Operation) {
	if opNames[op.Command()] {
		panic("Double registration of operations with identical command: " + op.Command())
	}
	opNames[op.Command()] = true
	client.RegisterOperation(op.Command(), op)
	server.RegisterOperation(op.Command(), op)
}

func PrintSingleOperationHelp(op Operation) {
	// TODO
	fmt.Fprintf(os.Stderr, "Currently no help message for operation '%s'\n", op.Command())
}

func PrintAllOperationsHelp() {
	// TODO
	fmt.Fprintln(os.Stderr, "Currently no help message exists.")
}
