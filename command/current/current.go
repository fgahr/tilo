package current

import (
	"github.com/fgahr/tilo/argparse"
	"github.com/fgahr/tilo/client"
	"github.com/fgahr/tilo/command"
	"github.com/fgahr/tilo/msg"
	"github.com/fgahr/tilo/server"
	"github.com/pkg/errors"
	"io"
)

type CurrentOperation struct {
	// No starte required
}

func (op CurrentOperation) Command() string {
	return "current"
}

func (op CurrentOperation) Parser() *argparse.Parser {
	return argparse.CommandParser(op.Command()).WithoutTask().WithoutParams()
}

func (op CurrentOperation) ClientExec(cl *client.Client, cmd msg.Cmd) error {
	cl.EstablishConnection()
	cl.SendToServer(cmd)
	resp := cl.ReceiveFromServer()
	cl.PrintResponse(resp)
	return errors.Wrap(cl.Error(), "Failed to determine the current task")
}

func (op CurrentOperation) ServerExec(srv *server.Server, req *server.Request) error {
	defer req.Close()
	resp := msg.Response{}
	if srv.CurrentTask.IsRunning() {
		resp.AddCurrentTask(srv.CurrentTask)
	} else {
		resp.SetError(errors.New("No active task"))
	}
	return srv.Answer(req, resp)
}

func (op CurrentOperation) PrintUsage(w io.Writer) {
	command.PrintSingleOperationHelp(op, w)
}

func init() {
	command.RegisterOperation(CurrentOperation{})
}
