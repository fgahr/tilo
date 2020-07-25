package listen

import (
	"io"
	"os"

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
	return "listen"
}

func (op operation) Parser() *argparse.Parser {
	return argparse.CommandParser(op.Command()).WithoutTask().WithoutParams()
}

func (op operation) DescribeShort() argparse.Description {
	return op.Parser().Describe("Listen for and print server notifications")
}

func (op operation) HelpHeaderAndFooter() (string, string) {
	header := "Connect to the server and listen for notifications. Print whatever is received"
	footer := "Use this mode for scripting purposes or as sample output when developing listeners in other languages"
	return header, footer
}

func (op operation) ClientExec(cl *client.Client, cmd msg.Cmd) error {
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

func (op operation) ServerExec(srv *server.Server, req *server.Request) error {
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
	command.RegisterOperation(operation{})
}
