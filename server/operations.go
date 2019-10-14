// Package server describes all server-side operations.
package server

// NOTE: All operations do their own logging, any returned errors are wrapped
// with explanations.

import (
	"github.com/fgahr/tilo/msg"
	"github.com/pkg/errors"
)

// Log a request at the appropriate debug level.
func (s *Server) logCommand(cmd msg.Cmd) {
	s.logFmtInfo("Processing command: %v\n", cmd)
}

// Log a response at the appropriate debug level.
func (s *Server) logResponse(resp msg.Response) {
	s.logFmtDebug("Returning response: %v\n", resp)
}

// Answer the request with the provided response.
func (s *Server) Answer(req *Request, resp msg.Response) error {
	return errors.Wrap(writeJsonLine(resp, req.Conn), "Failed to send response")
}

// Save a task to the backend database.
func (s *Server) SaveTask(task msg.Task) error {
	if task.IsRunning() {
		return errors.New("Cannot save an active task")
	}
	s.logFmtInfo("Saving task: %v\n", task)
	if err := s.Backend.Save(task); err != nil {
		s.logFmtInfo("%v\n", err)
		return err
	}
	return nil
}

// Change the server's current task.
func (s *Server) SetActiveTask(taskName string) {
	if s.CurrentTask.IsRunning() {
		s.logWarn("Task was not stopped before being superseded:", s.CurrentTask)
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

// Register the listener with the server. If it cannot be notified immediately,
// an error is returned.
func (s *Server) RegisterListener(req *Request) (NotificationListener, error) {
	lst := NotificationListener{req.Conn}
	// FIXME: Make thread-safe
	s.listeners = append(s.listeners, lst)
	return lst, nil
}

// Initiate the server to shut down, accepting no further connections.
func (s *Server) InitiateShutdown() {
	close(s.shutdownChan)
	if r := recover(); r != nil {
		s.logWarn(r)
	}
}
