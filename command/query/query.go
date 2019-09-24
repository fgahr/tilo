package query

import (
	// "github.com/fgahr/tilo/argparse"
	"github.com/fgahr/tilo/client"
	"github.com/fgahr/tilo/command"
	"github.com/fgahr/tilo/msg"
	"github.com/fgahr/tilo/server"
	// "github.com/pkg/errors"
)

type QueryOperation struct {
	// No state required
}

func (op QueryOperation) Command() string {
	return "query"
}

func (op QueryOperation) ClientExec(cl *client.Client, args ...string) error {
	// TODO
	return nil
}

func (op QueryOperation) ServerExec(srv *server.Server, req *server.Request) error {
	resp := msg.Response{}
	// TODO
	return srv.Answer(req, resp)
}

func (op QueryOperation) Help() command.Doc {
	// TODO: Improve, figure out what's required
	return command.Doc{
		ShortDescription: "Query the backend",
		LongDescription:  "Query the backend",
		Arguments:        []string{"<task,...>"},
	}
}

func init() {
	command.RegisterOperation(QueryOperation{})
}
