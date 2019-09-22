package srvcmd

import (
	"github.com/fgahr/tilo/argparse"
	"github.com/fgahr/tilo/client"
	"github.com/fgahr/tilo/command"
	"github.com/fgahr/tilo/config"
	"github.com/fgahr/tilo/msg"
	"github.com/fgahr/tilo/server"
	"github.com/pkg/errors"
	"net"
)

type ServerOperation struct {
	// No state required
}

func (op ServerOperation) Command() string {
	return "server"
}

func (op ServerOperation) ClientExec(conf *config.Opts, args ...string) error {
	if len(args) == 0 {
		command.PrintSingleOperationHelp(op)
	}
	argparse.WarnUnused(args[1:])
	switch args[0] {
	case "start":
		return client.EnsureServerIsRunning(conf)
	case "run":
		return server.Run(conf)
	default:
		command.PrintSingleOperationHelp(op)
	}
	return nil
}

func (op ServerOperation) ServerExec(srv *server.Server, conn net.Conn, cmd command.Cmd, resp *msg.Response) {
	resp.SetError(errors.New("Not a valid server operation:" + op.Command()))
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
