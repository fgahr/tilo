package query

import (
	"github.com/fgahr/tilo/argparse"
	"github.com/fgahr/tilo/client"
	"github.com/fgahr/tilo/command"
	"github.com/fgahr/tilo/msg"
	"github.com/fgahr/tilo/server"
	"github.com/pkg/errors"
	"io"
)

type QueryOperation struct {
	// No state required
}

type QueryParamHandler struct {
	// No state required
}

func (h QueryParamHandler) HandleParams(cmd *msg.Cmd, params []string) ([]string, error) {
	// TODO: Move to this package
	msg.ParseQueryArgs(params, cmd)
	return nil, nil
}

func (op QueryOperation) Command() string {
	return "query"
}

func (op QueryOperation) Parser() *argparse.Parser {
	return argparse.CommandParser(op.Command()).WithMultipleTasks().WithParamHandler(QueryParamHandler{})
}

func (op QueryOperation) ClientExec(cl *client.Client, cmd msg.Cmd) error {
	cl.SendReceivePrint(cmd)
	return errors.Wrap(cl.Error(), "Failed to query the server")
}

func (op QueryOperation) ServerExec(srv *server.Server, req *server.Request) error {
	defer req.Close()
	resp := msg.Response{}
Outer:
	for _, task := range req.Cmd.Tasks {
		for _, param := range req.Cmd.QueryParams {
			if sum, err := srv.Query(task, param); err != nil {
				resp.SetError(errors.Wrap(err, "A query failed"))
				break Outer
			} else {
				resp.AddQuerySummaries(sum)
			}
		}
	}
	return srv.Answer(req, resp)
}

func (op QueryOperation) PrintUsage(w io.Writer) {
	command.PrintSingleOperationHelp(op, w)
}

func init() {
	command.RegisterOperation(QueryOperation{})
}
