// Package msg provides means for client and server to communicate.
package msg

import (
	"fmt"
	"github.com/pkg/errors"
	"io"
	"os"
	"time"
)

const (
	// Possible command arguments.
	// NOTE: They are conceivably different from request commands and therefore
	// defined separately, although identical.
	ArgStart    = "start"
	ArgStop     = "stop"
	ArgCurrent  = "current"
	ArgAbort    = "abort"
	ArgQuery    = "query"
	ArgShutdown = "shutdown"
)

type Cmd struct {
	Op    string            `json:"operation"` // The operation to perform
	Flags map[string]bool   `json:"flags"`     // Possible flags
	Opts  map[string]string `json:"options"`   // Possible options
	Body  [][]string        `json:"body"`      // The body containing the command information
}

type QueryDetails []string

// Request, to be sent to the server.
// NOTE: Renaming pending as soon as the old struct is removed.
type Request struct {
	Cmd       string
	Tasks     []string
	QueryArgs []QueryDetails
	Combine   bool
}

type argParser interface {
	identifier() string
	handleArgs(args []string, now time.Time) (string, Request, error)
}

// Create a request based on command line parameters and the current time.
// This function contains the main command language logic.
// Note that passing the time here is necessary to avoid inconsistencies when
// encountering a date change around midnight. As a side note, it also
// simplifies testing.
func ParseRequest(args []string, now time.Time) (string, Request, error) {
	if len(args) == 0 {
		panic("Empty argument list passed from main.")
	}
	parsers := []argParser{
		stopParser(),
		currentTaskParser(),
		abortTaskParser(),
		shutdownParser(),
		startParser{os.Stderr},
		queryParser{os.Stderr},
	}

	cliCmd := args[0]
	for _, p := range parsers {
		if cliCmd == p.identifier() {
			return p.handleArgs(args[1:], now)
		}
	}
	return "", Request{}, errors.Errorf("Unknown command: %s", args[0])
}

// If args are given, a warning is emitted that they will be ignored.
func warnIgnoredArguments(args []string, out io.Writer) {
	if len(args) > 0 {
		fmt.Fprintf(out, "Extra arguments ignored: %v", args)
	}
}
