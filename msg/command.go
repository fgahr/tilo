// Package msg provides means for client and server to communicate.
package msg

import (
	"github.com/pkg/errors"
	"io"
	"time"
	"os"
)

const (
	// Types of command for the requests
	CmdStart    = "start"
	CmdStop     = "stop"
	CmdCurrent  = "current"
	CmdAbort    = "abort"
	CmdQuery    = "query"
	CmdShutdown = "shutdown"
)

type cmdOnlyParser struct {
	fnName string
	cliCmd string
	reqCmd string
	errout io.Writer
}

func (p cmdOnlyParser) identifier() string {
	return p.cliCmd
}

func (p cmdOnlyParser) handleArgs(args []string, now time.Time) (string, Request, error) {
	warnIgnoredArguments(args, p.errout)
	return p.fnName, Request{Cmd: p.reqCmd}, nil
}

func stopParser() argParser {
	return cmdOnlyParser{"RequestHandler.StopCurrentTask", argStop, CmdStop, os.Stderr}
}

func currentTaskParser() argParser {
	return cmdOnlyParser{"RequestHandler.GetCurrentTask", argCurrent, CmdCurrent, os.Stderr}
}

func abortTaskParser() argParser {
	return cmdOnlyParser{"RequestHandler.AbortCurrentTask", argAbort, CmdAbort, os.Stderr}
}

func shutdownParser() argParser {
	return cmdOnlyParser{"RequestHandler.ShutdownServer", argShutdown, CmdShutdown, os.Stderr}
}

type startParser struct {
	errout io.Writer
}

func (p startParser) identifier() string {
	return argStart
}

func (p startParser) handleArgs(args []string, now time.Time) (string, Request, error) {
	if len(args) < 1 {
		return "", Request{},
			errors.New("Missing task name for 'start'.")
	}

	warnIgnoredArguments(args[1:], p.errout)

	tasks, err := getTaskNames(args[0])
	if err != nil {
		return "", Request{}, err
	} else if len(tasks) > 1 {
		return "", Request{},
			errors.Errorf("Can only start one task at a time. Given: %v", tasks)
	}
	return "RequestHandler.StartTask", Request{Cmd: CmdStart, Tasks: tasks}, nil
}
