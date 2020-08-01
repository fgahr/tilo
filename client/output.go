package client

import (
	"github.com/fgahr/tilo/argparse"
	"github.com/fgahr/tilo/msg"
)

// Formatter describes an output formatter for the client.
type Formatter interface {
	// Name is the name by which configuration can refer to this formatter
	Name() string
	// Error formats an error
	Error(err error)
	// Response formats a server response
	Response(resp msg.Response)
	// HelpSingleOperation describes that operation
	HelpSingleOperation(op Operation)
	// HelpAllOperations describes a range of operations
	HelpAllOperations(descriptions []argparse.Description)
}

var formatters map[string]Formatter

// RegisterFormatter makes the formatter available under its name.
func RegisterFormatter(f Formatter) {
	formatters[f.Name()] = f
}

// Get a formatter by name
func GetFormatter(name string) Formatter {
	return formatters[name]
}

func init() {
	formatters = make(map[string]Formatter)
}
