package srvcmd

import (
	"github.com/fgahr/tilo/argparse"
	"github.com/fgahr/tilo/client"
	"github.com/fgahr/tilo/command"
	"github.com/fgahr/tilo/msg"
	"github.com/fgahr/tilo/server"
	"github.com/pkg/errors"
)

type ServerOperation struct {
	// No state required
}

func (op ServerOperation) Command() string {
	return "server"
}

func (op ServerOperation) ClientExec(cl *client.Client, args ...string) error {
	if len(args) == 0 {
		command.PrintSingleOperationHelp(op)
	}
	argparse.WarnUnused(args[1:])
	switch args[0] {
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
		Arguments:        []string{"start|run"},
	}
}

func init() {
	command.RegisterOperation(ServerOperation{})
}
