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
	return "stop"
}

func (op AbortOperation) ClientExec(cl *client.Client, args ...string) error {
	argparse.WarnUnused(args)
	stopCmd := msg.Cmd{
		Op: op.Command(),
	}

	cl.EstablishConnection()
	cl.SendToServer(stopCmd)
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
		Arguments:        []string{},
	}
}

func init() {
	command.RegisterOperation(AbortOperation{})
}
