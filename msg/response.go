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

// A response informing about an error that occured.
func ErrorResponse(err error) Response {
	resp := Response{Status: RespError}
	resp.addToBody(line(err.Error()))
	return resp
}

// A response informing about receiving and complying to a shutdown request.
func ShutdownResponse(task *Task, err error) Response {
	resp := Response{Status: RespSuccess}
	resp.addToBody(line("Received shutdown request -- shutting down"))
	if err != nil {
		resp.addToBody(line(err.Error()))
	} else if task != nil {
		resp.addTaskDescription("Stopped", task)
	}
	return resp
}

// A response informing that newTask has been started, superseding oldTask.
func StartTaskResponse(newTask *Task, oldTask *Task) Response {
	resp := Response{Status: RespSuccess}
	resp.addTaskDescription("Now", newTask)
	if oldTask != nil {
		resp.addTaskDescription("Stopped", oldTask)
	}
	return resp
}

// A response informing about a currently running task.
func CurrentTaskResponse(task *Task) Response {
	resp := Response{Status: RespSuccess}
	resp.addTaskDescription("Currently", task)
	return resp
}

// A response informing that a task has been stopped.
func StoppedTaskResponse(task *Task) Response {
	return stopWithTag("Previously", task)
}

// A response informing that a task has been aborted.
func AbortedTaskResponse(task *Task) Response {
	return stopWithTag("Aborted", task)
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

// A stop response for a task, featuring the given tag.
func stopWithTag(tag string, task *Task) Response {
	if !task.HasEnded {
		panic("Task needs to end before responding to stop!")
	}
	resp := Response{Status: RespSuccess}
	resp.addTaskDescription(tag, task)
	return resp
}

// The error encapsulated in the response, if any.
func (r Response) Err() error {
	if r.Status == RespError {
		return errors.Errorf("%s", r.Body[0][0])
	}
	return nil
}

// Add data describing a task to the response body.
func (r *Response) addTaskDescription(tag string, task *Task) {
	if task.HasEnded {
		r.addToBody(
			line(tag, "Since", "Until"),
			line(task.Name, formatTime(task.Started), formatTime(task.Ended)),
		)
	} else {
		r.addToBody(
			line(tag, "Since"),
			line(task.Name, formatTime(task.Started)),
		)
	}
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
