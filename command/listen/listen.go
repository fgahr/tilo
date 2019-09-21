package listen

import (
	// "github.com/fgahr/tilo/client"
	"github.com/fgahr/tilo/command"
	"github.com/fgahr/tilo/config"
	"github.com/fgahr/tilo/msg"
	"github.com/fgahr/tilo/server"
	// "github.com/pkg/errors"
)

type ListenOperation struct {
	// No state required
}

func (op ListenOperation) Command() string {
	return "listen"
}

func (op ListenOperation) ClientExec(conf *config.Opts, args ...string) error {
	// TODO
	return nil
}

func (op ListenOperation) ServerExec(srv *server.Server, cmd command.Cmd, resp *msg.Response) {
	// TODO
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
