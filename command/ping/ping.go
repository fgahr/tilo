package ping

import (
	"fmt"
	"os"
	"time"

	"github.com/fgahr/tilo/argparse"
	"github.com/fgahr/tilo/client"
	"github.com/fgahr/tilo/command"
	"github.com/fgahr/tilo/msg"
	"github.com/fgahr/tilo/server"
)

type operation struct {
	// No state required
}

func (op operation) Command() string {
	return "ping"
}

func (op operation) Parser() *argparse.Parser {
	return argparse.CommandParser(op.Command()).WithoutTask().WithoutParams()
}

func (op operation) DescribeShort() argparse.Description {
	return op.Parser().Describe("Ping the server")
}

func (op operation) HelpHeaderAndFooter() (string, string) {
	header := "Request a reply from the server, measure the time between sending and receiving"
	footer := "Use this command to test server responsiveness"
	return header, footer
}

func (op operation) ClientExec(cl *client.Client, cmd msg.Cmd) error {
	// TODO: Should ping start a server if none is running?
	cl.EstablishConnection()
	before := time.Now()
	if _, err := fmt.Fprintln(os.Stderr, "Sending ping to server"); err != nil {
		return err
	}
	cl.SendToServer(cmd)
	cl.ReceiveFromServer() // Ignoring response
	after := time.Now()
	if cl.Failed() {
		return cl.Error()
	}
	_, err := fmt.Fprintf(os.Stderr, "Received pong from server after %v\n", after.Sub(before))
	return err
}

func (op operation) ServerExec(srv *server.Server, req *server.Request) error {
	defer req.Close()
	resp := msg.Response{}
	resp.Status = msg.RespSuccess
	resp.AddPong()
	return srv.Answer(req, resp)
}

func init() {
	command.RegisterOperation(operation{})
}
