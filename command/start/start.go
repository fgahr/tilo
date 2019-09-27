package start

import (
	"github.com/fgahr/tilo/argparse"
	"github.com/fgahr/tilo/client"
	"github.com/fgahr/tilo/command"
	"github.com/fgahr/tilo/msg"
	"github.com/fgahr/tilo/server"
	"github.com/pkg/errors"
)

type StartOperation struct {
	// No state required
}

func (op StartOperation) Command() string {
	return "start"
}

func (op StartOperation) Parser() *argparse.Parser {
	return argparse.CommandParser(op.Command()).WithSingleTask().WithoutParams()
}

func (op StartOperation) ClientExec(cl *client.Client, cmd msg.Cmd) error {
	cl.EstablishConnection()
	cl.SendToServer(cmd)
	resp := cl.ReceiveFromServer()
	cl.PrintResponse(resp)
	return errors.Wrapf(cl.Error(), "Failed to start task '%s'", cmd.Tasks[0])
}

func (op StartOperation) ServerExec(srv *server.Server, req *server.Request) error {
	defer req.Close()
	resp := msg.Response{}
	taskName := req.Cmd.Tasks[0]
	task, stopped := srv.StopCurrentTask()
	if stopped {
		if err := srv.SaveTask(task); err != nil {
			resp.SetError(err)
		}
		resp.AddStoppedTask(task)
	}
	srv.SetActiveTask(taskName)
	resp.AddCurrentTask(srv.CurrentTask)
	return srv.Answer(req, resp)
}

func (op StartOperation) Help() command.Doc {
	return command.Doc{
		ShortDescription: "Start a task",
		LongDescription:  "Start a task",
	}
}

func init() {
	command.RegisterOperation(StartOperation{})
}
