package stop

import (
	"github.com/fgahr/tilo/client"
	"github.com/fgahr/tilo/command"
	"github.com/fgahr/tilo/config"
	"github.com/fgahr/tilo/msg"
	"github.com/fgahr/tilo/server"
	"github.com/pkg/errors"
)

type StopOperation struct {
	// No state required
}

func (op StopOperation) Command() string {
	return "stop"
}

func (op StopOperation) ClientExec(conf *config.Opts, args ...string) error {
	if len(args) != 0 {
		// TODO: Warn about ignored arguments? Crash? Print usage?
	}
	clientCmd := command.Cmd{
		Op: op.Command(),
	}

	if err := client.EnsureServerIsRunning(conf); err != nil {
		return errors.Wrap(err, "Failed to stop the current task")
	}

	resp, err := client.SendToServer(conf, clientCmd)
	if err != nil {
		return errors.Wrap(err, "Failed to stop the current task")
	}
	return client.PrintResponse(conf, resp)
}

func (op StopOperation) ServerExec(srv *server.Server, cmd command.Cmd, resp *msg.Response) {
	task, stopped := srv.StopCurrentTask()
	if stopped {
		if err := srv.SaveTask(task); err != nil {
			resp.SetError(err)
		}
		resp.AddStoppedTask(task)
	} else {
		resp.SetError(errors.New("No active task"))
	}
}

func (op StopOperation) Help() command.Doc {
	return command.Doc{
		ShortDescription: "Stop the current task",
		LongDescription:  "Stop the current task",
		Arguments:        []string{},
	}
}

func init() {
	command.RegisterOperation(StopOperation{})
}
