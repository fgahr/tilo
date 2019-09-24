package shutdown

import (
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

func (op ShutdownOperation) ClientExec(cl *client.Client, args ...string) error {
	shutdownCmd := msg.Cmd{
		Op: op.Command(),
	}

	cl.EstablishConnection()
	cl.SendToServer(shutdownCmd)
	resp := cl.ReceiveFromServer()
	cl.PrintResponse(resp)
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

func (op ShutdownOperation) Help() command.Doc {
	// TODO: Improve, figure out what's required
	return command.Doc{
		ShortDescription: "Request server shutdown",
		LongDescription:  "Request server shutdown",
		Arguments:        []string{""},
	}
}

func init() {
	command.RegisterOperation(ShutdownOperation{})
}
