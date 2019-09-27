package shutdown

import (
	"github.com/fgahr/tilo/argparse"
	"github.com/fgahr/tilo/client"
	"github.com/fgahr/tilo/command"
	"github.com/fgahr/tilo/msg"
	"github.com/fgahr/tilo/server"
	"github.com/pkg/errors"
)

type ShutdownOperation struct {
	// No state required
}

func (op ShutdownOperation) Command() string {
	return "shutdown"
}

func (op ShutdownOperation) Parser() *argparse.Parser {
	return argparse.CommandParser(op.Command()).WithoutTask().WithoutParams()
}

func (op ShutdownOperation) ClientExec(cl *client.Client, cmd msg.Cmd) error {
	cl.EstablishConnection()
	cl.SendToServer(cmd)
	resp := cl.ReceiveFromServer()
	cl.PrintResponse(resp)
	return errors.Wrapf(cl.Error(), "Failed to initiate server shutdown")
}

func (op ShutdownOperation) ServerExec(srv *server.Server, req *server.Request) error {
	defer srv.InitiateShutdown()
	defer req.Close()
	resp := msg.Response{}
	task, stopped := srv.StopCurrentTask()
	if stopped {
		if err := srv.SaveTask(task); err != nil {
			resp.SetError(err)
		}
		resp.AddStoppedTask(task)
	}
	resp.AddShutdownMessage()
	return srv.Answer(req, resp)
}

func (op ShutdownOperation) Help() command.Doc {
	return command.Doc{
		ShortDescription: "Request server shutdown",
		LongDescription:  "Request server shutdown",
	}
}

func init() {
	command.RegisterOperation(ShutdownOperation{})
}
