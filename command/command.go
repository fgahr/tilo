// Package for commands to the server
package command

import (
	"github.com/fgahr/tilo/client"
	"github.com/fgahr/tilo/server"
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
