package resume

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
	return "resume"
}

func (op operation) Parser() *argparse.Parser {
	return argparse.CommandParser(op.Command()).WithoutTask().WithoutParams()
}

func (op operation) DescribeShort() argparse.Description {
	return op.Parser().Describe("Resume the last active task")
}

func (op operation) HelpHeaderAndFooter() (string, string) {
	header := "Resume the last active task"
	footer := "Exits with non-zero status if a task is currently active or if no prior task exists"
	return header, footer
}

func (op operation) ClientExec(cl *client.Client, cmd msg.Cmd) error {
	cl.SendReceivePrint(cmd)
	return errors.Wrap(cl.Error(), "failed to resume the last active task")
}

func (op operation) ServerExec(srv *server.Server, req *server.Request) error {
	resp := msg.Response{}
	if srv.CurrentTask.IsRunning() {
		resp.SetError(errors.New("a task is already active"))
	} else {
		if summary, err := srv.Backend.RecentTasks(1); err != nil {
			resp.SetError(errors.Wrap(err, "failed to determine latest task"))
		} else if len(summary) == 0 {
			resp.SetError(errors.New("no recent activity to continue"))
		} else {
			tName := summary[0].Task
			srv.SetActiveTask(tName)
			resp.AddCurrentTask(srv.CurrentTask)
		}
	}
	return srv.Answer(req, resp)
}

func init() {
	command.RegisterOperation(operation{})
}
