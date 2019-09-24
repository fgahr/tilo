// Package server describes all server-side operations.
package server

// NOTE: All operations do their own logging, any returned errors are wrapped
// with explanations.

import (
	"encoding/json"
	"github.com/fgahr/tilo/config"
	"github.com/fgahr/tilo/msg"
	"github.com/pkg/errors"
	"log"
)

// TODO: Rename to something sane
func (s *Server) logDebugSome(format string, v ...interface{}) {
	if s.conf.DebugLevel >= config.DebugSome {
		log.Printf(format, v...)
	}
}

// TODO: Rename to something sane
func (s *Server) logDebugAll(format string, v ...interface{}) {
	if s.conf.DebugLevel >= config.DebugAll {
		log.Printf(format, v...)
	}
}

// Log a request at the appropriate debug level.
func (s *Server) logCommand(cmd msg.Cmd) {
	s.logDebugSome("Processing command: %v\n", cmd)
}

// Log a response at the appropriate debug level.
func (s *Server) logResponse(resp msg.Response) {
	s.logDebugAll("Returning response: %v\n", resp)
}

// Answer the request with the provided response.
func (s *Server) Answer(req *Request, resp msg.Response) error {
	enc := json.NewEncoder(req.Conn)
	defer s.logResponse(resp)
	return errors.Wrap(enc.Encode(resp), "Failed to send response")
}

// Save a task to the backend database.
func (s *Server) SaveTask(task msg.Task) error {
	if task.IsRunning() {
		return errors.New("Cannot save an active task")
	}
	s.logDebugSome("Saving task: %v\n", task)
	if err := s.backend.Save(task); err != nil {
		s.logDebugSome("%v\n", err)
		return err
	}
	return nil
}

// Change the server's current task.
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

// Register the listener with the server. If it cannot be notified immediately,
// an error is returned.
func (s *Server) RegisterListener(req *Request) error {
	lst := NotificationListener{req.Conn}
	if err := lst.notify(taskNotification(s.CurrentTask)); err != nil {
		lst.disconnect()
		return errors.Wrap(err, "Could not notify listener, disconnecting")
	}
	// FIXME: Make thread-safe
	s.listeners = append(s.listeners, lst)
	return nil
}

// Query the server's backend for a task's within the given parameters.
func (s *Server) Query(taskName string, param msg.QueryParam) ([]msg.Summary, error) {
	return s.backend.Query(taskName, param)
}

// Initiate the server to shut down, accepting no further connections.
func (s *Server) InitiateShutdown() {
	close(s.shutdownChan)
	if r := recover(); r != nil {
		log.Println(r)
	}
}
