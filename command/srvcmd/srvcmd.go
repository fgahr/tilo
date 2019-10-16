package srvcmd

import (
	"github.com/fgahr/tilo/argparse"
	"github.com/fgahr/tilo/client"
	"github.com/fgahr/tilo/command"
	"github.com/fgahr/tilo/msg"
	"github.com/fgahr/tilo/server"
	"github.com/pkg/errors"
	"io"
)

const (
	RUN   = "run"
	START = "start"
)

type CommandHandler struct {
	command string
}

func (h *CommandHandler) HandleArgs(_ *msg.Cmd, args []string) ([]string, error) {
	if len(args) == 0 {
		return args, errors.New("Require a command but none was given")
	}
	if isKnownCommand(args[0]) {
		h.command = args[0]
	} else {
		return args, errors.New("Not a known server command: " + args[0])
	}
	return args[1:], nil
}

func isKnownCommand(str string) bool {
	switch str {
	case RUN:
		return true
	case START:
		return true
	default:
		return false
	}
}

type ServerOperation struct {
	ch *CommandHandler
}

func (op ServerOperation) Command() string {
	return "server"
}

func (op ServerOperation) Parser() *argparse.Parser {
	return argparse.CommandParser(op.Command()).WithoutTask().WithArgHandler(op.ch)
}

func (op ServerOperation) ClientExec(cl *client.Client, _ msg.Cmd) error {
	switch op.ch.command {
	case START:
		cl.EnsureServerIsRunning()
	case RUN:
		cl.RunServer()
	}
	return cl.Error()
}

func (op ServerOperation) ServerExec(srv *server.Server, req *server.Request) error {
	defer req.Close()
	resp := msg.Response{}
	resp.SetError(errors.New("Not a valid server operation:" + op.Command()))
	return srv.Answer(req, resp)
}

func (op ServerOperation) PrintUsage(w io.Writer) {
	command.PrintSingleOperationHelp(op, w)
}

func init() {
	command.RegisterOperation(ServerOperation{new(CommandHandler)})
}
