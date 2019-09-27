// Package for commands to the server
package command

import (
	"fmt"
	"github.com/fgahr/tilo/client"
	"github.com/fgahr/tilo/server"
	"io"
	"os"
)

var opNames = make(map[string]bool)

type Operation interface {
	client.ClientOperation
	server.ServerOperation
	// The command identifier
	Command() string
}

func RegisterOperation(op Operation) {
	if opNames[op.Command()] {
		panic("Double registration of operations with identical command: " + op.Command())
	}
	opNames[op.Command()] = true
	client.RegisterOperation(op.Command(), op)
	server.RegisterOperation(op.Command(), op)
}

func PrintSingleOperationHelp(op Operation, w io.Writer) {
	// TODO: Add actual help message
	fmt.Fprintf(w, "Currently no help message for operation '%s'\n", op.Command())
}

func PrintAllOperationsHelp() {
	// TODO: Use parser description
	fmt.Fprintln(os.Stderr, "Currently no help message exists.")
}
