package listen

import (
	"github.com/fgahr/tilo/argparse"
	"github.com/fgahr/tilo/client"
	"github.com/fgahr/tilo/command"
	"github.com/fgahr/tilo/msg"
	"github.com/fgahr/tilo/server"
	"github.com/pkg/errors"
	"io"
	"os"
)

type ListenOperation struct {
	// No state required
}

func (op ListenOperation) Command() string {
	return "listen"
}

func (op ListenOperation) ClientExec(cl *client.Client, args ...string) error {
	argparse.WarnUnused(args)
	listenCmd := msg.Cmd{Op: "listen"}

	cl.EstablishConnection()
	cl.SendToServer(listenCmd)
	resp := cl.ReceiveFromServer()
	if resp.Err() != nil {
		return resp.Err()
	}
	if cl.Failed() {
		return errors.Wrap(cl.Error(), "Failed to establish listener connection")
	}
	_, err := io.Copy(os.Stdout, cl)
	return err
}

func (op ListenOperation) ServerExec(srv *server.Server, req *server.Request) error {
	// NOTE: Connection has to be kept open!
	resp := msg.Response{}
	if err := srv.RegisterListener(req); err != nil {
		resp.SetError(errors.Wrap(err, "Failed to add as listener"))
	} else {
		resp.SetListening()
	}
	return srv.Answer(req, resp)
}

func (op ListenOperation) Help() command.Doc {
	return command.Doc{
		ShortDescription: "Listen for notifications",
		LongDescription:  "Listen for notifications",
		Arguments:        []string{},
	}
}

func init() {
	command.RegisterOperation(ListenOperation{})
}
