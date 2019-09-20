// Package server describes all server-side operations.
package server

import (
	"github.com/fgahr/tilo/msg"
	"github.com/fgahr/tilo/config"
	"github.com/pkg/errors"
	"log"
)

// Log a request at the appropriate debug level.
func (s *Server) logRequest(req msg.Request) {
	// FIXME: Requests will be logged twice in several situations
	if s.conf.DebugLevel >= config.DebugSome {
		log.Printf("Processing request: %v\n", req)
	}
}

// Log a response at the appropriate debug level.
func (s *Server) logResponse(resp *msg.Response) {
	if s.conf.DebugLevel >= config.DebugAll {
		log.Printf("Returning response: %v\n", *resp)
	}
}

func (s *Server) LogTask(task msg.Task) {
	s.backend.Save(task)
}

func (s *Server) SetActiveTask(taskName string) {
	if s.CurrentTask.IsRunning() {
		s.CurrentTask.Stop()
	}
	s.CurrentTask = msg.FreshTask(taskName)
}

func (s *Server) StopCurrentTask(resp *msg.Response) error {
	if !s.CurrentTask.IsRunning() && resp != nil {
		resp.SetError(errors.New("No active task"))
		return nil
	}
	endTime := s.CurrentTask.Stop()
	// NOTE: Delegating to a goroutine might cause problems when shutting down,
	// therefore notify sequentially.
	s.notifyListeners(Notification{"", endTime})
	err := s.backend.Save(s.CurrentTask)
	if resp != nil {
		resp.AddStoppedTask(s.CurrentTask)
		s.logResponse(resp)
	}
	return err
}
