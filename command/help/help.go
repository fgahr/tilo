package help

import (
	"github.com/fgahr/tilo/argparse"
	"github.com/fgahr/tilo/client"
	"github.com/fgahr/tilo/command"
	"github.com/fgahr/tilo/msg"
	"github.com/fgahr/tilo/server"
	"github.com/pkg/errors"
)

type CommandHandler struct {
	specific bool
	command  string
}

func (h *CommandHandler) HandleArgs(_ *msg.Cmd, args []string) ([]string, error) {
	if len(args) == 0 {
		return args, nil
	}

	h.specific = true
	h.command = args[0]
	return args[1:], nil
}

func (h *CommandHandler) TakesParameters() bool {
	return true
}

type HelpOperation struct {
	ch *CommandHandler
}

func (op HelpOperation) Command() string {
	return "help"
}

func (op HelpOperation) Parser() *argparse.Parser {
	return argparse.CommandParser(op.Command()).WithoutTask().WithArgHandler(op.ch)
}

func (op HelpOperation) Describe() argparse.Description {
	return argparse.Description{
		Cmd:   op.Command(),
		First: "[command]",
		What:  "View detailed help for a command",
	}
}

func (op HelpOperation) ClientExec(cl *client.Client, cmd msg.Cmd) error {
	if op.ch.specific {
		if err := cl.PrintSingleOperationHelp(op.ch.command); err != nil {
			cl.PrintError(err)
			client.PrintAllOperationsHelp()
		}
	} else {
		client.PrintAllOperationsHelp()
	}
	return nil
}

func (op HelpOperation) ServerExec(srv *server.Server, req *server.Request) error {
	defer req.Close()
	resp := msg.Response{}
	resp.SetError(errors.New("Not a valid server operation:" + op.Command()))
	return srv.Answer(req, resp)
}

func init() {
	command.RegisterOperation(HelpOperation{&CommandHandler{}})
}
