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

func (op ListenOperation) Parser() *argparse.Parser {
	return argparse.CommandParser(op.Command()).WithoutTask().WithoutParams()
}

func (op ListenOperation) DescribeShort() argparse.Description {
	return op.Parser().Describe("Listen for and print server notifications")
}

func (op ListenOperation) HelpHeaderAndFooter() (string, string) {
	header := "Connect to the server and listen for notifications. Print whatever is received"
	footer := "Use this mode for scripting purposes or as sample output when developing listeners in other languages"
	return header, footer
}

func (op ListenOperation) ClientExec(cl *client.Client, cmd msg.Cmd) error {
	cl.EstablishConnection()
	cl.SendToServer(cmd)
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
	if listener, err := srv.RegisterListener(req); err != nil {
		resp.SetError(errors.Wrap(err, "Failed to add as listener"))
	} else {
		resp.SetListening()
		defer listener.Notify(server.TaskNotification(srv.CurrentTask))
	}
	return srv.Answer(req, resp)
}

func init() {
	command.RegisterOperation(ListenOperation{})
}
