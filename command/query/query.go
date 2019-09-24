package query

import (
	"github.com/fgahr/tilo/argparse"
	"github.com/fgahr/tilo/client"
	"github.com/fgahr/tilo/command"
	"github.com/fgahr/tilo/msg"
	"github.com/fgahr/tilo/server"
	"github.com/pkg/errors"
)

type QueryOperation struct {
	// No state required
}

func (op QueryOperation) Command() string {
	return "query"
}

func (op QueryOperation) ClientExec(cl *client.Client, args ...string) error {
	if len(args) < 2 {
		return errors.New("Need task name(s) and at least on parameter")
	}
	tasks, err := argparse.GetTaskNames(args[0])
	if err != nil {
		return err
	}
	cmd := msg.Cmd{
		Op:    op.Command(),
		Tasks: tasks,
	}
	msg.ParseQueryArgs(args[1:], &cmd)

	cl.EstablishConnection()
	cl.SendToServer(cmd)
	resp := cl.ReceiveFromServer()
	cl.PrintResponse(resp)
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
