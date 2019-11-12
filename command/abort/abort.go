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

func (op AbortOperation) DescribeShort() argparse.Description {
	return op.Parser().Describe("Abort the currently active task without saving")
}

func (op AbortOperation) HelpHeaderAndFooter() (string, string) {
	header := "Abort the currently active task without logging the time"
	footer := "Use the `stop` command to log the time of a task"
	return header, footer
}

func (op AbortOperation) ClientExec(cl *client.Client, cmd msg.Cmd) error {
	cl.SendReceivePrint(cmd)
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

func init() {
	command.RegisterOperation(AbortOperation{})
}
