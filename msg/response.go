// Package msg provides means for client and server to communicate.
package msg

import (
	"github.com/pkg/errors"
	"strings"
	"time"
)

const (
	// Status
	RespError   = "error"
	RespSuccess = "success"
	// Type
	RespStartTask   = "start"
	RespStopTask    = "stop"
	RespCurrentTask = "current"
)

// Type repserenting a server's response to a client's request.
type Response struct {
	Status string
	Error  string
	Body   [][]string
}

// Type representing summary of a single request.
type Summary struct {
	Task    string
	Details QueryDetails
	Total   time.Duration
	Start   time.Time
	End     time.Time
}

// Create a response containing the given query summaries.
func QueryResponse(summaries []Summary) Response {
	var resp Response
	if summaries == nil {
		resp = Response{Status: RespSuccess}
		resp.addToBody(line("Nothing found"))
		return resp
	}

	resp = Response{Status: RespSuccess}
	addNewline := false
	for _, s := range summaries {
		// Separate summaries by an empty line; skipped in first iteration.
		if addNewline {
			resp.addToBody(line())
		} else {
			addNewline = true
		}
		header := []string{s.Task}
		for _, detail := range s.Details {
			header = append(header, detail)
		}
		resp.addToBody(line(strings.Join(header, " ")))
		resp.addToBody(line("First logged", formatTime(s.Start)))
		resp.addToBody(line("Last logged", formatTime(s.End)))
		resp.addToBody(line("Total time", s.Total.String()))
	}
	return resp
}

func (r *Response) SetError(err error) {
	if err == nil {
		return
	}

	if r.Status != RespError {
		r.Status = RespError
	}
	r.Error = err.Error()
}

func (r *Response) SetListening() {
	if r.Status != RespError {
		r.Status = RespSuccess
	}
	r.addToBody(line("Listening"))
}

func (r *Response) statusIsSet() bool {
	return r.Status != ""
}

func (r *Response) AddCurrentTask(task Task) {
	if task.HasEnded {
		panic("Task not running but should be reported as started!")
	}
	r.addTaskWithDescription("Currently", task)
}

func (r *Response) AddStartedTask(task Task) {
	if task.HasEnded {
		panic("Task not running but should be reported as started!")
	}
	r.addTaskWithDescription("Now", task)
}

func (r *Response) AddStoppedTask(task Task) {
	if !task.HasEnded {
		panic("Task needs to end before responding to stop!")
	}
	r.addTaskWithDescription("Stopped", task)
}

func (r *Response) AddAbortedTask(task Task) {
	if !task.HasEnded {
		panic("Task needs to end before responding to abort!")
	}
	r.addTaskWithDescription("Aborted", task)
}

func (r *Response) addTaskWithDescription(description string, task Task) {
	if !r.statusIsSet() {
		r.Status = RespSuccess
	}
	if task.HasEnded {
		r.addToBody(
			line(description, "Since", "Until"),
			line(task.Name, formatTime(task.Started), formatTime(task.Ended)),
		)
	} else {
		r.addToBody(
			line(description, "Since"),
			line(task.Name, formatTime(task.Started)),
		)
	}
}

// The error encapsulated in the response, if any.
func (r Response) Err() error {
	if r.Status == RespError {
		return errors.New(r.Error)
	}
	return nil
}

// Add the given lines to the response body.
func (r *Response) addToBody(lines ...[]string) {
	for _, line := range lines {
		r.Body = append(r.Body, line)
	}
}

// Convenience function to fill the response body.
func line(words ...string) []string {
	return words
}

// Format a time instance as a string.
func formatTime(t time.Time) string {
	return t.Format("2006-01-02 15:04:05")
}
