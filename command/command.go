// Package for commands to the server
package command

import (
	"github.com/fgahr/tilo/client"
	"github.com/fgahr/tilo/server"
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
		panic("Double registration of operations with identical command")
	}
	opNames[op.Command()] = true
	client.RegisterOperation(op.Command(), op)
	server.RegisterOperation(op.Command(), op)
}

func PrintSingleOperationHelp(op Operation) {
	// TODO
}

func PrintAllOperationsHelp(op Operation) {
	// TODO
}
