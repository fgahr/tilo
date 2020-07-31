package query

import (
	"time"

	"github.com/fgahr/tilo/argparse"
	"github.com/fgahr/tilo/argparse/quantifier"
	"github.com/fgahr/tilo/client"
	"github.com/fgahr/tilo/command"
	"github.com/fgahr/tilo/msg"
	"github.com/fgahr/tilo/server"
	"github.com/fgahr/tilo/server/backend"
	"github.com/pkg/errors"
)

type operation struct {
	// No state required
}

func (op operation) Command() string {
	return "query"
}

func (op operation) Parser() *argparse.Parser {
	return argparse.CommandParser(op.Command()).WithMultipleTasks().WithArgHandler(newQueryArgHandler(time.Now()))
}

func (op operation) DescribeShort() argparse.Description {
	return op.Parser().Describe("Make enquiries about prior activity")
}

func (op operation) HelpHeaderAndFooter() (string, string) {
	header := "Get information about recorded activity"
	footer := "Where indicated, a list of quantifiers (or pairs thereof) can be given\n" +
		"Parameters can be freely combined and repeated in a single query\n\n" +
		"Examples\n" +
		"    tilo query :all :this-week                    # This week's activity across all tasks\n" +
		"    tilo query foo :between 2019-01-01:2019-06-30 # Logged on task foo in first half of 2019\n" +
		"    tilo query bar :month=2019-01,2019-02,2019-03 # Activity for bar in three different months"
	return header, footer
}

func (op operation) ClientExec(cl *client.Client, cmd msg.Cmd) error {
	cl.SendReceivePrint(cmd)
	return errors.Wrap(cl.Error(), "Failed to query the server")
}

func (op operation) ServerExec(srv *server.Server, req *server.Request) error {
	defer req.Close()
	resp := msg.Response{}
	backend := srv.Backend
Outer:
	for _, task := range req.Cmd.TaskNames {
		for _, quant := range req.Cmd.Quantities {
			if sum, err := queryBackend(backend, task, quant); err != nil {
				resp.SetError(errors.Wrap(err, "A query failed"))
				break Outer
			} else {
				resp.AddQuerySummaries(sum)
			}
		}
	}
	return srv.Answer(req, resp)
}

func queryBackend(b backend.Backend, task string, param msg.Quantity) ([]msg.Summary, error) {
	var sum []msg.Summary
	if b == nil {
		return sum, errors.New("No backend present")
	}
	var err error
	// TODO: Some more length checks required. Might be restructured beforehand.
	switch param.Type {
	case quantifier.TimeDay:
		start, err := time.Parse("2006-01-02", param.Elems[0])
		if err != nil {
			return nil, errors.Wrap(err, "Unable to construct query")
		}
		end := start.AddDate(0, 0, 1)
		sum, err = b.GetTaskBetween(task, start, end)
	case quantifier.TimeBetween:
		if len(param.Elems) < 2 {
			return nil, errors.Errorf("Invalid query parameter: %v", param)
		}
		start, err := time.Parse("2006-01-02", param.Elems[0])
		if err != nil {
			return nil, err
		}
		end, err := time.Parse("2006-01-02", param.Elems[1])
		if err != nil {
			return nil, err
		}
		sum, err = b.GetTaskBetween(task, start, end)
	case quantifier.TimeMonth:
		start, err := time.Parse("2006-01", param.Elems[0])
		if err != nil {
			return nil, errors.Wrap(err, "Unable to construct query")
		}
		end := start.AddDate(0, 1, 0)
		sum, err = b.GetTaskBetween(task, start, end)
	case quantifier.TimeYear:
		start, err := time.Parse("2006", param.Elems[0])
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

func init() {
	command.RegisterOperation(operation{})
}
