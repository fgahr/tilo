package help

import (
	"github.com/fgahr/tilo/argparse"
	"github.com/fgahr/tilo/client"
	"github.com/fgahr/tilo/command"
	"github.com/fgahr/tilo/msg"
	"github.com/fgahr/tilo/server"
	"github.com/pkg/errors"
)

type cmdHandler struct {
	specific bool
	command  string
}

func (h *cmdHandler) HandleArgs(_ *msg.Cmd, args []string) ([]string, error) {
	if len(args) == 0 {
		return args, nil
	}

	h.specific = true
	h.command = args[0]
	return args[1:], nil
}

func (h *cmdHandler) TakesParameters() bool {
	return true
}

func (h *cmdHandler) DescribeParameters() []argparse.ParamDescription {
	return []argparse.ParamDescription{
		argparse.ParamDescription{
			ParamName:        "",
			ParamValues:      "<command>",
			ParamExplanation: "If given, the command to examine more closely",
		},
	}
}

type operation struct {
	ch *cmdHandler
}

func (op operation) Command() string {
	return "help"
}

func (op operation) Parser() *argparse.Parser {
	return argparse.CommandParser(op.Command()).WithoutTask().WithArgHandler(op.ch)
}

func (op operation) DescribeShort() argparse.Description {
	return argparse.Description{
		Cmd:   op.Command(),
		First: "<command>",
		What:  "Describe program or detailed usage of a command",
	}
}

func (op operation) HelpHeaderAndFooter() (string, string) {
	header := "Describe usage of a command"
	footer := "You already know how to use this command :-)"
	return header, footer
}

func (op operation) ClientExec(cl *client.Client, cmd msg.Cmd) error {
	if op.ch.specific {
		if cl.CommandExists(op.ch.command) {
			cl.PrintSingleOperationHelp(op.ch.command)
		} else {
			cl.PrintAllOperationsHelp()
			return errors.Errorf("\nNo such command: %s", op.ch.command)
		}
	} else {
		cl.PrintAllOperationsHelp()
	}
	return nil
}

func (op operation) ServerExec(srv *server.Server, req *server.Request) error {
	defer req.Close()
	resp := msg.Response{}
	resp.SetError(errors.New("Not a valid server operation:" + op.Command()))
	return srv.Answer(req, resp)
}

func init() {
	command.RegisterOperation(operation{&cmdHandler{}})
}
