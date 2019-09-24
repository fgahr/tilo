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

func (op StartOperation) ClientExec(cl *client.Client, args ...string) error {
	if len(args) == 0 {
		return errors.New("No task name given")
	}
	tasks, err := argparse.GetTaskNames(args[0])
	if err != nil {
		return err
	} else if len(tasks) > 1 || tasks[0] == argparse.AllTasks {
		return errors.New("Cannot start more than one task")
	}

	taskName := tasks[0]
	startCmd := msg.Cmd{
		Op:    op.Command(),
		Tasks: []string{taskName},
	}

	cl.EstablishConnection()
	cl.SendToServer(startCmd)
	resp := cl.ReceiveFromServer()
	cl.PrintResponse(resp)
	return errors.Wrapf(cl.Error(), "Failed to start task '%s'", taskName)
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
