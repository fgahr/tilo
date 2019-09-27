package abort

import (
	"github.com/fgahr/tilo/argparse"
	"github.com/fgahr/tilo/client"
	"github.com/fgahr/tilo/command"
	"github.com/fgahr/tilo/msg"
	"github.com/fgahr/tilo/server"
	"github.com/pkg/errors"
)

type AbortOperation struct {
	// No state required
}

func (op AbortOperation) Command() string {
	return "abort"
}

func (op AbortOperation) Parser() *argparse.Parser {
	return argparse.CommandParser(op.Command()).WithoutTask().WithoutParams()
}

func (op AbortOperation) ClientExec(cl *client.Client, cmd msg.Cmd) error {
	cl.EstablishConnection()
	cl.SendToServer(cmd)
	resp := cl.ReceiveFromServer()
	cl.PrintResponse(resp)
	return errors.Wrap(cl.Error(), "Failed to stop the current task")
}

func (op AbortOperation) ServerExec(srv *server.Server, req *server.Request) error {
	defer req.Close()
	resp := msg.Response{}
	task, stopped := srv.StopCurrentTask()
	if stopped {
		resp.AddStoppedTask(task)
	} else {
		resp.SetError(errors.New("No active task"))
	}
	return srv.Answer(req, resp)
}

func (op AbortOperation) Help() command.Doc {
	return command.Doc{
		ShortDescription: "Abort the current task",
		LongDescription:  "Abort the current task",
	}
}

func init() {
	command.RegisterOperation(AbortOperation{})
}
