// Package server describes all server-side operations.
package server

// NOTE: All operations do their own logging, any returned errors are wrapped
// with explanations.

import (
	"github.com/fgahr/tilo/config"
	"github.com/fgahr/tilo/msg"
	"github.com/pkg/errors"
	"log"
)

func (s *Server) logDebugSome(format string, v ...interface{}) {
	if s.conf.DebugLevel >= config.DebugSome {
		log.Printf(format, v...)
	}
}

func (s *Server) logDebugAll(format string, v ...interface{}) {
	if s.conf.DebugLevel >= config.DebugAll {
		log.Printf(format, v...)
	}
}

// Log a request at the appropriate debug level.
func (s *Server) logRequest(req msg.Request) {
	s.logDebugSome("Processing request: %v\n", req)
}

// Log a response at the appropriate debug level.
func (s *Server) logResponse(resp *msg.Response) {
	s.logDebugAll("Returning response: %v\n", *resp)
}

func (s *Server) SaveTask(task msg.Task) error {
	s.logDebugSome("Saving task: %v\n", task)
	if err := s.backend.Save(task); err != nil {
		s.logDebugSome("%v\n", err)
		return err
	}
	return nil
}

func (s *Server) SetActiveTask(taskName string) {
	if s.CurrentTask.IsRunning() {
		s.logDebugAll("Task was not stopped before being superseded: %v\n", s.CurrentTask)
		s.CurrentTask.Stop()
	}
	s.CurrentTask = msg.FreshTask(taskName)
	s.notifyListeners()
}

// Stop the current task and return it. Returns true if the task was actually
// halted and false if it had been stopped before this function was called.
func (s *Server) StopCurrentTask() (msg.Task, bool) {
	if s.CurrentTask.IsRunning() {
		s.CurrentTask.Stop()
		s.notifyListeners()
		return s.CurrentTask, true
	}
	return s.CurrentTask, false
}

// TODO: Delegate to individual operations
func (s *Server) OldStopCurrentTask(resp *msg.Response) error {
	if !s.CurrentTask.IsRunning() && resp != nil {
		resp.SetError(errors.New("No active task"))
		return nil
	}
	s.CurrentTask.Stop()
	// NOTE: Delegating to a goroutine might cause problems when shutting down,
	// therefore notify sequentially.
	s.notifyListeners()
	err := s.backend.Save(s.CurrentTask)
	if resp != nil {
		resp.AddStoppedTask(s.CurrentTask)
		s.logResponse(resp)
	}
	return err
}
