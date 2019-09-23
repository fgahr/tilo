package start

import (
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

func (op StartOperation) ClientExec(cl *client.Client, args ...string) error {
	// TODO: Parse arguments, extract task name
	taskName := "foo"
	clientCmd := msg.Cmd{
		Op:   op.Command(),
		Body: [][]string{[]string{taskName}},
	}

	if err := cl.EnsureServerIsRunning(); err != nil {
		return errors.Wrapf(err, "Cannot start task '%s'", taskName)
	}

	cl.SendToServer(clientCmd)
	resp := cl.ReceiveFromServer()
	cl.PrintResponse(resp)
	return cl.Error()
}

func (op StartOperation) ServerExec(srv *server.Server, req *server.Request) error {
	defer req.Close()
	resp := msg.Response{}
	taskName := req.Cmd.Body[0][0]
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
	// TODO: Improve, figure out what's required
	return command.Doc{
		ShortDescription: "Start a task",
		LongDescription:  "Start a task",
		Arguments:        []string{"<task>"},
	}
}

func init() {
	command.RegisterOperation(StartOperation{})
}
