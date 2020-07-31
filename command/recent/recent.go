package recent

import (
	"github.com/fgahr/tilo/argparse"
	"github.com/fgahr/tilo/client"
	"github.com/fgahr/tilo/command"
	"github.com/fgahr/tilo/msg"
	"github.com/fgahr/tilo/server"
	"github.com/pkg/errors"
)

type operation struct {
	// No state required
}

func (op operation) Command() string {
	return "recent"
}

func (op operation) Parser() *argparse.Parser {
	return argparse.CommandParser(op.Command()).WithoutTask().WithoutParams()
}

func (op operation) DescribeShort() argparse.Description {
	return op.Parser().Describe("Display recent activity")
}

func (op operation) HelpHeaderAndFooter() (string, string) {
	header := "Display recent activity"
	footer := "For more detailed inquiries, use the `query` command"
	return header, footer
}

func (op operation) ClientExec(cl *client.Client, cmd msg.Cmd) error {
	cl.SendReceivePrint(cmd)
	return errors.Wrap(cl.Error(), "Failed to determine recent activity")
}

func (op operation) ServerExec(srv *server.Server, req *server.Request) error {
	defer req.Close()
	resp := msg.Response{}

	fetchNum := 5
	if srv.CurrentTask.IsRunning() {
		fetchNum--
		resp.AddCurrentTask(srv.CurrentTask)
	}

	if summary, err := srv.Backend.RecentTasks(fetchNum); err != nil {
		resp.SetError(errors.Wrap(err, "failed to fetch recent task data"))
	} else {
		resp.AddQuerySummaries(summary)
	}

	return srv.Answer(req, resp)
}

func init() {
	command.RegisterOperation(operation{})
}
