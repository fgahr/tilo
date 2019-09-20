// Package server describes all server-side operations.
package server

import (
	"github.com/fgahr/tilo/config"
	"github.com/fgahr/tilo/msg"
	"log"
)

// Handler for all client requests. Exported functions are intended for
// RPC calls, so they have to satisfy the criteria.
type RequestHandler struct {
	conf         *config.Opts          // Configuration parameters for this instance
	shutdownChan chan struct{}           // Channel to broadcast server shutdown
	currentTask  msg.Task               // The currently active task, if any
	listeners    []*notificationListener // Listeners for task change notifications
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
	// return h.backend.Close()
	return nil
}

// // Gather a query response from the database.
// func (h *RequestHandler) Query(req msg.Request, resp *msg.Response) error {
//	h.logRequest(req)
//	var summaries []msg.Summary
//	for _, detail := range req.QueryArgs {
//		for _, task := range req.Tasks {
//			newSummaries, err := h.backend.Query(task, detail)
//			if err != nil {
//				return errors.Wrapf(err, "backend.Query failed for task %s with detail %v",
//					task, detail)
//			}
//			summaries = append(summaries, newSummaries...)
//		}
//	}
//	*resp = msg.QueryResponse(summaries)
//	h.logResponse(resp)
//	return nil
// }
