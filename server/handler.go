// Package server describes all server-side operations.
package server

import (
	"github.com/fgahr/tilo/config"
	"github.com/fgahr/tilo/msg"
	"github.com/fgahr/tilo/server/db"
	_ "github.com/mattn/go-sqlite3"
	"github.com/pkg/errors"
	"log"
)

// Handler for all client requests. Exported functions are intended for
// RPC calls, so they have to satisfy the criteria.
type RequestHandler struct {
	shutdownChan chan struct{}  // The server to which this handler is attached
	activeTask   *msg.Task      // The currently active task, if any
	backend      *db.Backend    // Database connection
	conf         *config.Params // Configuration parameters for this instance
}

// Close the request handler, shutting down the backend.
func (h *RequestHandler) close() error {
	return h.backend.Close()
}

// Log a request at the appropriate debug level.
func (h *RequestHandler) logRequest(req msg.Request) {
	// FIXME: Requests will be logged twice in several situations
	if h.conf.DebugLevel >= config.DebugSome {
		log.Printf("Processing request: %v\n", req)
	}
}

// Log a response at the appropriate debug level.
func (h *RequestHandler) logResponse(resp *msg.Response) {
	if h.conf.DebugLevel >= config.DebugAll {
		log.Printf("Returning response: %v\n", *resp)
	}
}

// Process the given request, producing a response accordingly.
func (h *RequestHandler) HandleRequest(req msg.Request, resp *msg.Response) error {
	h.logRequest(req)
	var err error = nil
	switch req.Cmd {
	case msg.CmdStart:
		err = h.StartTask(req, resp)
	case msg.CmdStop:
		err = h.StopCurrentTask(req, resp)
	case msg.CmdCurrent:
		err = h.GetCurrentTask(req, resp)
	case msg.CmdAbort:
		err = h.AbortCurrentTask(req, resp)
	case msg.CmdQuery:
		err = h.Query(req, resp)
	case msg.CmdShutdown:
		err = h.ShutdownServer(req, resp)
	default:
		err = errors.Errorf("Not implemented: %s", req.Cmd)
	}

	if err != nil {
		log.Println(err)
		*resp = msg.ErrorResponse(err)
	}
	h.logResponse(resp)
	return nil
}

// Start a timer for the given arguments, respond its details.
func (h *RequestHandler) StartTask(req msg.Request, resp *msg.Response) error {
	h.logRequest(req)
	taskName := req.Tasks[0]
	oldTask := h.activeTask
	if oldTask != nil {
		err := h.StopCurrentTask(req, resp)
		if err != nil {
			return errors.Wrap(err, "Stopping previous timer failed")
		}
	}
	h.activeTask = msg.NewTask(taskName)
	*resp = msg.StartTaskResponse(h.activeTask, oldTask)
	h.logResponse(resp)
	return nil
}

// Stop the current timer, respond its details.
func (h *RequestHandler) StopCurrentTask(req msg.Request, resp *msg.Response) error {
	if resp != nil {
		h.logRequest(req)
	}
	if h.activeTask == nil && resp != nil {
		*resp = msg.ErrorResponse(errors.New("No active task"))
		return nil
	}
	h.activeTask.Stop()
	err := h.backend.Save(h.activeTask)
	if resp != nil {
		*resp = msg.StoppedTaskResponse(h.activeTask)
	}
	h.activeTask = nil
	if resp != nil {
		h.logResponse(resp)
	}
	return err
}

// Respond about the currently active task.
func (h *RequestHandler) GetCurrentTask(req msg.Request, resp *msg.Response) error {
	h.logRequest(req)
	if h.activeTask == nil {
		*resp = msg.ErrorResponse(errors.New("No active task"))
	} else {
		*resp = msg.CurrentTaskResponse(h.activeTask)
	}
	h.logResponse(resp)
	return nil
}

// Abort the currently active task without saving it to the backend. Respond
// its details.
func (h *RequestHandler) AbortCurrentTask(req msg.Request, resp *msg.Response) error {
	h.logRequest(req)
	if h.activeTask == nil {
		*resp = msg.ErrorResponse(errors.New("No active task"))
	}
	h.activeTask.Stop()
	aborted := h.activeTask
	h.activeTask = nil
	*resp = msg.AbortedTaskResponse(aborted)
	h.logResponse(resp)
	return nil
}

// Shut down the server, saving any currently active task beforehand.
func (h *RequestHandler) ShutdownServer(req msg.Request, resp *msg.Response) error {
	h.logRequest(req)
	// This causes the server's main loop to exit at the next iteration.
	close(h.shutdownChan)
	lastActive := h.activeTask
	// TODO: When responding to a request, this should be returned somehow.
	err := h.StopCurrentTask(req, resp)
	*resp = msg.ShutdownResponse(lastActive, err)
	h.logResponse(resp)
	return nil
}

// Gather a query response from the database.
func (h *RequestHandler) Query(req msg.Request, resp *msg.Response) error {
	h.logRequest(req)
	var summaries []msg.Summary
	for _, detail := range req.QueryArgs {
		for _, task := range req.Tasks {
			newSummaries, err := h.backend.Query(task, detail)
			if err != nil {
				return errors.Wrapf(err, "backend.Query failed for task %s with detail %v",
					task, detail)
			}
			summaries = append(summaries, newSummaries...)
		}
	}
	*resp = msg.QueryResponse(summaries)
	h.logResponse(resp)
	return nil
}
