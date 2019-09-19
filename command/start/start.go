package start

import (
	"github.com/fgahr/tilo/client"
	"github.com/fgahr/tilo/command"
	"github.com/fgahr/tilo/config"
	"github.com/fgahr/tilo/msg"
	"github.com/fgahr/tilo/server"
	"github.com/pkg/errors"
)

type StartOperation struct {
	// No state required
}

func (s StartOperation) Command() string {
	return "start"
}

func (s StartOperation) ClientExec(conf *config.Opts, args ...string) error {
	// TODO: Parse arguments, extract task name
	taskName := "foo"
	clientCmd := command.Cmd{
		Op:   s.Command(),
		Body: [][]string{[]string{taskName}},
	}
	// TODO: Print response
	_, err := client.SendToServer(conf, clientCmd)
	if err != nil {
		// TODO: Include task name
		return errors.Wrap(err, "Failed to start task: "+taskName)
	}
	// TODO
	return nil
}

func (s StartOperation) ServerExec(srv *server.Server, command command.Command, resp *msg.Response) error {
	// TODO
	return nil
}

func (s StartOperation) Doc() string {
	// TODO
	return "TODO"
}

func init() {
	// TODO
	command.RegisterOperation(StartOperation{})
}
