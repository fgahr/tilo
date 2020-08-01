// Package command describes commands available to the user.
package command

import (
	"github.com/fgahr/tilo/client"
	"github.com/fgahr/tilo/server"
)

var opNames = make(map[string]bool)

// Operation is the common interface for the basic operations of the program.
type Operation interface {
	client.Operation
	server.Operation
	// Command describes the name by which this operation is recognized, i.e.
	// the command line identifier.
	Command() string
}

// RegisterOperation makes an operation available to be called from the command line.
func RegisterOperation(op Operation) {
	if opNames[op.Command()] {
		panic("Double registration of operations with identical command: " + op.Command())
	}
	opNames[op.Command()] = true
	client.RegisterOperation(op.Command(), op)
	server.RegisterOperation(op.Command(), op)
}
