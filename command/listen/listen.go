package listen

import (
	"encoding/json"
	"github.com/fgahr/tilo/argparse"
	"github.com/fgahr/tilo/client"
	"github.com/fgahr/tilo/command"
	"github.com/fgahr/tilo/config"
	"github.com/fgahr/tilo/msg"
	"github.com/fgahr/tilo/server"
	"github.com/pkg/errors"
	"io"
	"os"
	"net"
)

type ListenOperation struct {
	// No state required
}

func (op ListenOperation) Command() string {
	return "listen"
}

func (op ListenOperation) ClientExec(conf *config.Opts, args ...string) error {
	argparse.WarnUnused(args...)
	conn, err := client.EstablishConnection(conf)
	if err != nil {
		return err
	}
	listenCmd := command.Cmd{Op: "listen"}
	enc := json.NewEncoder(conn)
	if err = enc.Encode(listenCmd); err != nil {
		return errors.Wrap(err, "Failed to send listening request")
	}
	_, err = io.Copy(os.Stdout, conn)
	return err
}

func (op ListenOperation) ServerExec(srv *server.Server, conn net.Conn, cmd command.Cmd, resp *msg.Response) {
	if err := srv.RegisterListenerConnection(conn); err != nil {
		resp.SetError(errors.Wrap(err, "Failed to add as listener"))
	} else {
		resp.SetListening()
	}
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
