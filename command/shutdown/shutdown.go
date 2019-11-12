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

func (op ShutdownOperation) DescribeShort() argparse.Description {
	return op.Parser().Describe("Request server shutdown")
}

func (op ShutdownOperation) HelpHeaderAndFooter() (string, string) {
	header := "Request server shutdown"
	return header, ""
}

func (op ShutdownOperation) ClientExec(cl *client.Client, cmd msg.Cmd) error {
	if cl.ServerIsRunning() {
		cl.SendReceivePrint(cmd)
	} else {
		cl.PrintMessage("Server appears to be down. Nothing to do")
	}
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

func init() {
	command.RegisterOperation(ShutdownOperation{})
}
