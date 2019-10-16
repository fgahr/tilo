package query

import (
	"github.com/fgahr/tilo/argparse"
	"github.com/fgahr/tilo/client"
	"github.com/fgahr/tilo/command"
	"github.com/fgahr/tilo/msg"
	"github.com/fgahr/tilo/server"
	"github.com/fgahr/tilo/server/backend"
	"github.com/pkg/errors"
	"io"
	"time"
)

type QueryOperation struct {
	// No state required
}

func (op QueryOperation) Command() string {
	return "query"
}

func (op QueryOperation) Parser() *argparse.Parser {
	return argparse.CommandParser(op.Command()).WithMultipleTasks().WithArgHandler(newQueryArgHandler())
}

func (op QueryOperation) ClientExec(cl *client.Client, cmd msg.Cmd) error {
	cl.SendReceivePrint(cmd)
	return errors.Wrap(cl.Error(), "Failed to query the server")
}

func (op QueryOperation) ServerExec(srv *server.Server, req *server.Request) error {
	defer req.Close()
	resp := msg.Response{}
	backend := srv.Backend
Outer:
	for _, task := range req.Cmd.Tasks {
		for _, param := range req.Cmd.QueryParams {
			if sum, err := queryBackend(backend, task, param); err != nil {
				resp.SetError(errors.Wrap(err, "A query failed"))
				break Outer
			} else {
				resp.AddQuerySummaries(sum)
			}
		}
	}
	return srv.Answer(req, resp)
}

func queryBackend(b backend.Backend, task string, param msg.QueryParam) ([]msg.Summary, error) {
	if len(param) < 2 {
		return nil, errors.Errorf("Invalid query parameter: %v", param)
	}

	var sum []msg.Summary
	if b == nil {
		return sum, errors.New("No backend present")
	}
	var err error
	switch param[0] {
	case QryDay:
		start, err := time.Parse("2006-01-02", param[1])
		if err != nil {
			return nil, errors.Wrap(err, "Unable to construct query")
		}
		end := start.AddDate(0, 0, 1)
		sum, err = b.GetTaskBetween(task, start, end)
	case QryBetween:
		if len(param) < 3 {
			return nil, errors.Errorf("Invalid query parameter: %v", param)
		}
		start, err := time.Parse("2006-01-02", param[1])
		if err != nil {
			return nil, err
		}
		end, err := time.Parse("2006-01-02", param[2])
		if err != nil {
			return nil, err
		}
		sum, err = b.GetTaskBetween(task, start, end)
	case QryMonth:
		start, err := time.Parse("2006-01", param[1])
		if err != nil {
			return nil, errors.Wrap(err, "Unable to construct query")
		}
		end := start.AddDate(0, 1, 0)
		sum, err = b.GetTaskBetween(task, start, end)
	case QryYear:
		start, err := time.Parse("2006", param[1])
		if err != nil {
			return nil, errors.Wrap(err, "Unable to construct query")
		}
		end := start.AddDate(1, 0, 0)
		sum, err = b.GetTaskBetween(task, start, end)
	}
	if err != nil {
		return nil, errors.Wrap(err, "Error in database query")
	}

	// Setting the details allows to give better output.
	for i, _ := range sum {
		sum[i].Details = param
	}
	return sum, nil
}

func (op QueryOperation) PrintUsage(w io.Writer) {
	command.PrintSingleOperationHelp(op, w)
}

func init() {
	command.RegisterOperation(QueryOperation{})
}
