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
	conf         *config.Params          // Configuration parameters for this instance
	shutdownChan chan struct{}           // Channel to broadcast server shutdown
	currentTask  *msg.Task               // The currently active task, if any
	backend      *db.Backend             // Database connection
	listeners    []*notificationListener // Listeners for task change notifications
}

// Create a fresh request handler with the given configuration and connections.
func newRequestHandler(conf *config.Params, shutdownChan chan struct{}, backend *db.Backend) *RequestHandler {
	return &RequestHandler{
		conf:         conf,
		shutdownChan: shutdownChan,
		currentTask:  msg.NewIdleTask(),
		backend:      backend,
		listeners:    []*notificationListener{},
	}
}

// Close the request handler, shutting down the backend.
// NOTE: Exporting this method trips up the rpc server and we don't need to
// satisfy the Closer interface.
func (h *RequestHandler) close() error {
	if len(h.listeners) > 0 {
		log.Println("Disconnecting listeners")
	}
	for _, lst := range h.listeners {
		if err := lst.disconnect(); err != nil {
			log.Println("Error closing listener connection:", err)
		}
	}
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

// Register a listener waiting for notifications.
func (h *RequestHandler) registerListener(lst *notificationListener) {
	log.Println("Registering notification listener")
	// TODO: Make thread-safe: connections can be server concurrently!
	h.listeners = append(h.listeners, lst)
	lst.notify(taskNotification(h.currentTask))
}

// Send a notification to all registered listeners.
func (h *RequestHandler) notifyListeners(ntf Notification) {
	if h.conf.DebugLevel == config.DebugAll {
		log.Println("Notifying listeners:", ntf)
	}
	for i, lst := range h.listeners {
		if lst == nil {
			continue
		}
		if err := lst.notify(ntf); err != nil {
			log.Println("Could not notify listener, disconnecting:", err)
			lst.disconnect()
			h.listeners[i] = nil
		}
	}
}

// Start a timer for the given arguments, respond its details.
func (h *RequestHandler) StartTask(req msg.Request, resp *msg.Response) error {
	h.logRequest(req)
	taskName := req.Tasks[0]
	if h.currentTask.IsRunning() {
		err := h.StopCurrentTask(req, resp)
		if err != nil {
			return errors.Wrap(err, "Stopping previous timer failed")
		}
	}
	h.currentTask = msg.NewTask(taskName)
	h.notifyListeners(taskNotification(h.currentTask))
	resp.AddStartedTask(h.currentTask)
	h.logResponse(resp)
	return nil
}

// Stop the current timer, respond its details.
func (h *RequestHandler) StopCurrentTask(req msg.Request, resp *msg.Response) error {
	if resp != nil {
		h.logRequest(req)
	}
	if !h.currentTask.IsRunning() && resp != nil {
		resp.SetError(errors.New("No active task"))
		return nil
	}
	endTime := h.currentTask.Stop()
	// NOTE: Delegating to a goroutine might cause problems when shutting down,
	// therefore notify sequentially.
	h.notifyListeners(Notification{"", endTime})
	err := h.backend.Save(h.currentTask)
	if resp != nil {
		resp.AddStoppedTask(h.currentTask)
		h.logResponse(resp)
	}
	return err
}

// Respond about the currently active task.
func (h *RequestHandler) GetCurrentTask(req msg.Request, resp *msg.Response) error {
	h.logRequest(req)
	if h.currentTask.IsRunning() {
		resp.AddCurrentTask(h.currentTask)
	} else {
		resp.SetError(errors.New("No active task"))
	}
	h.logResponse(resp)
	return nil
}

// Abort the currently active task without saving it to the backend. Respond
// its details.
func (h *RequestHandler) AbortCurrentTask(req msg.Request, resp *msg.Response) error {
	h.logRequest(req)
	if !h.currentTask.IsRunning() {
		resp.SetError(errors.New("No active task"))
		return nil
	}
	endTime := h.currentTask.Stop()
	h.notifyListeners(Notification{"", endTime})
	aborted := h.currentTask
	resp.AddAbortedTask(aborted)
	h.logResponse(resp)
	return nil
}

// Shut down the server, saving any currently active task beforehand.
func (h *RequestHandler) ShutdownServer(req msg.Request, resp *msg.Response) error {
	h.logRequest(req)
	// This causes the server's main loop to exit at the next iteration.
	close(h.shutdownChan)
	// If the channel has already been closed due to other events, proceed.
	if r := recover(); r != nil {
		log.Println(r)
	}
	err := h.StopCurrentTask(req, resp)
	// The above call may undeservedly set the error status
	resp.Status = msg.RespSuccess
	resp.SetError(err)
	h.logResponse(resp)
	h.notifyListeners(shutdownNotification())
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
