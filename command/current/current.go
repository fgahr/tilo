package current

import (
	"github.com/fgahr/tilo/client"
	"github.com/fgahr/tilo/command"
	"github.com/fgahr/tilo/msg"
	"github.com/fgahr/tilo/server"
	"github.com/pkg/errors"
)

type CurrentOperation struct {
	// No starte required
}

func (op CurrentOperation) Command() string {
	return "current"
}

func (op CurrentOperation) ClientExec(cl *client.Client, args ...string) error {
	currentCmd := msg.Cmd{
		Op: op.Command(),
	}

	cl.EstablishConnection()
	cl.SendToServer(currentCmd)
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

func (op CurrentOperation) Help() command.Doc {
	// TODO: Improve, figure out what's required
	return command.Doc{
		ShortDescription: "Print the currently running task",
		LongDescription:  "Print the currently running task",
		Arguments:        []string{},
	}
}

func init() {
	command.RegisterOperation(CurrentOperation{})
}
