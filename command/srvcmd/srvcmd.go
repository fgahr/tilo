package srvcmd

import (
	"github.com/fgahr/tilo/argparse"
	"github.com/fgahr/tilo/client"
	"github.com/fgahr/tilo/command"
	"github.com/fgahr/tilo/msg"
	"github.com/fgahr/tilo/server"
	"github.com/pkg/errors"
)

type CommandHandler struct {
	command string
}

func (h *CommandHandler) HandleParams(_ *msg.Cmd, params []string) ([]string, error) {
	if len(params) == 0 {
		return params, errors.New("Require a command but none was given")
	}
	h.command = params[0]
	return params[1:], nil
}

type ServerOperation struct {
	ch *CommandHandler
}

func (op ServerOperation) Command() string {
	return "server"
}

func (op ServerOperation) Parser() *argparse.Parser {
	return argparse.CommandParser(op.Command()).WithoutTask().WithParamHandler(op.ch)
}

func (op ServerOperation) ClientExec(cl *client.Client, _ msg.Cmd) error {
	switch op.ch.command {
	case "start":
		cl.EnsureServerIsRunning()
	case "run":
		cl.RunServer()
	default:
		command.PrintSingleOperationHelp(op)
	}
	return cl.Error()
}

func (op ServerOperation) ServerExec(srv *server.Server, req *server.Request) error {
	defer req.Close()
	resp := msg.Response{}
	resp.SetError(errors.New("Not a valid server operation:" + op.Command()))
	return srv.Answer(req, resp)
}

func (op ServerOperation) Help() command.Doc {
	return command.Doc{
		ShortDescription: "Run in server mode",
		LongDescription:  "Run in server mode",
	}
}

func init() {
	command.RegisterOperation(ServerOperation{new(CommandHandler)})
}
