package start

import (
	// "github.com/fgahr/tilo/client"
	"github.com/fgahr/tilo/cmd"
	"github.com/fgahr/tilo/config"
	"github.com/fgahr/tilo/msg"
	"github.com/fgahr/tilo/server"
	// "github.com/pkg/errors"
)

type StartOperation struct {
	// TODO
}

func (s StartOperation) Command() string {
	return "start"
}

func (s StartOperation) ClientExec(conf *config.Opts, args ...string) error {
	// TODO
	return nil
}

func (s StartOperation) ServerExec(srv *server.Server, cmd cmd.Cmd, resp *msg.Response) error {
	// TODO
	return nil
}

func (s StartOperation) Doc() string {
	// TODO
	return "TODO"
}

func init() {
	// TODO
	cmd.RegisterOperation(StartOperation{})
}
