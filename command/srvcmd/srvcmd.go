package srvcmd

import (
	"github.com/fgahr/tilo/argparse"
	"github.com/fgahr/tilo/client"
	"github.com/fgahr/tilo/command"
	"github.com/fgahr/tilo/msg"
	"github.com/fgahr/tilo/server"
	"github.com/pkg/errors"
)

const (
	RUN   = "run"
	START = "start"
	STOP  = "stop"
)

type cmdHandler struct {
	command string
}

func (h *cmdHandler) HandleArgs(_ *msg.Cmd, args []string) ([]string, error) {
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

func (h *cmdHandler) TakesParameters() bool {
	return true
}

func (h *cmdHandler) DescribeParameters() []argparse.ParamDescription {
	return []argparse.ParamDescription{
		argparse.ParamDescription{
			ParamName:        "start",
			ParamExplanation: "Start a server in the background, suppressing output",
		},
		argparse.ParamDescription{
			ParamName:        "stop",
			ParamExplanation: "Stop a running server",
		},
		argparse.ParamDescription{
			ParamName:        "run",
			ParamExplanation: "Start a server in the foreground, printing log messages",
		},
	}
}

func isKnownCommand(str string) bool {
	switch str {
	case RUN:
		return true
	case START:
		return true
	case STOP:
		return true
	default:
		return false
	}
}

type operation struct {
	ch *cmdHandler
}

func (op operation) Command() string {
	return "server"
}

func (op operation) Parser() *argparse.Parser {
	return argparse.CommandParser(op.Command()).WithoutTask().WithArgHandler(op.ch)
}

func (op operation) DescribeShort() argparse.Description {
	return argparse.Description{
		Cmd:   op.Command(),
		First: "[start|stop|run]",
		What:  "Start or stop a server process or run in the foreground",
	}
}

func (op operation) HelpHeaderAndFooter() (string, string) {
	header := "Start or stop a server process"
	footer := "Several other commands may spawn a server process if it is not yet running"
	return header, footer
}

func (op operation) ClientExec(cl *client.Client, cmd msg.Cmd) error {
	switch op.ch.command {
	case START:
		cl.EnsureServerIsRunning()
	case STOP:
		op.requestShutdown(cl, cmd)
	case RUN:
		cl.RunServer()
	}
	return cl.Error()
}

func (op operation) requestShutdown(cl *client.Client, cmd msg.Cmd) error {
	// FIXME: This is a bit of a hack for now. With more server commands added
	// (such as `reload`, `restart`, etc.) it will make sense to enable
	// ServerExec for this operation.
	cmd.Op = "shutdown"
	if cl.ServerIsRunning() {
		cl.SendReceivePrint(cmd)
	} else {
		cl.PrintMessage("Server appears to be down. Nothing to do")
	}
	return errors.Wrapf(cl.Error(), "Failed to initiate server shutdown")
}

func (op operation) ServerExec(srv *server.Server, req *server.Request) error {
	defer req.Close()
	resp := msg.Response{}
	resp.SetError(errors.New("Not a valid server operation:" + op.Command()))
	return srv.Answer(req, resp)
}

func init() {
	command.RegisterOperation(operation{new(cmdHandler)})
}
