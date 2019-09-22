package ping

import (
	"fmt"
	"github.com/fgahr/tilo/argparse"
	"github.com/fgahr/tilo/client"
	"github.com/fgahr/tilo/command"
	"github.com/fgahr/tilo/config"
	"github.com/fgahr/tilo/msg"
	"github.com/fgahr/tilo/server"
	"github.com/pkg/errors"
	"net"
	"os"
	"time"
)

type PingOperation struct {
	// No state required
}

func (op PingOperation) Command() string {
	return "ping"
}

func (op PingOperation) ClientExec(conf *config.Opts, args ...string) error {
	argparse.WarnUnused(args)
	pingCmd := command.Cmd{Op: op.Command()}
	before := time.Now()
	fmt.Fprintln(os.Stderr, "Sending ping to server")
	resp, err := client.SendToServer(conf, pingCmd)
	if err != nil {
		return errors.Wrap(err, "Error during ping roundtrip")
	} else if errStat := resp.Err(); errStat != nil {
		return resp.Err()
	}
	after := time.Now()
	_, err = fmt.Fprintf(os.Stderr, "Received pong from server after %v\n", after.Sub(before))
	return errors.Wrap(err, "Failed to write ping summary")
}

func (op PingOperation) ServerExec(srv *server.Server, conn net.Conn, cmd command.Cmd, resp *msg.Response) {
	resp.Status = msg.RespSuccess
	resp.AddPong()
}

func (op PingOperation) Help() command.Doc {
	return command.Doc{
		ShortDescription: "Check whether the server is running",
		LongDescription:  "Check whether the server is running",
	}
}

func init() {
	command.RegisterOperation(PingOperation{})
}
