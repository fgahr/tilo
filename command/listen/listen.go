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
	cl.EstablishConnection()
	if cl.Failed() {
		return cl.Error()
	}
	listenCmd := msg.Cmd{Op: "listen"}
	cl.SendToServer(listenCmd)
	if cl.Failed() {
		return cl.Error()
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
