package start

import (
	"github.com/fgahr/tilo/argparse"
	"github.com/fgahr/tilo/client"
	"github.com/fgahr/tilo/command"
	"github.com/fgahr/tilo/msg"
	"github.com/fgahr/tilo/server"
	"github.com/pkg/errors"
)

type operation struct {
	// No state required
}

func (op operation) Command() string {
	return "start"
}

func (op operation) Parser() *argparse.Parser {
	return argparse.CommandParser(op.Command()).WithSingleTask().WithoutParams()
}

func (op operation) DescribeShort() argparse.Description {
	return op.Parser().Describe("Start logging activity on a task")
}

func (op operation) HelpHeaderAndFooter() (string, string) {
	header := "Set the currently active task, i.e. start logging time. If a task is active, save it first"
	footer := "To avoid saving the previous task, use the `abort` command first\n\n" +
		"This command can also be used from time to time to avoid losing activity accidentally\n" +
		"In this case the `current` command will only show elapsed time since the last 'save'"
	return header, footer
}

func (op operation) ClientExec(cl *client.Client, cmd msg.Cmd) error {
	cl.SendReceivePrint(cmd)
	return errors.Wrapf(cl.Error(), "Failed to start task '%s'", cmd.Tasks[0])
}

func (op operation) ServerExec(srv *server.Server, req *server.Request) error {
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

func init() {
	command.RegisterOperation(operation{})
}
