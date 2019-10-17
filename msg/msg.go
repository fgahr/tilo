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

// TODO: Doc comments. This one is important.
type Quantity struct {
	Type  string
	Elems []string
}

type QueryParam []string

type Cmd struct {
	Op          string            `json:"operation"`    // The operation to perform
	Flags       map[string]bool   `json:"flags"`        // Possible flags
	Opts        map[string]string `json:"options"`      // Possible options
	Tasks       []string          `json:"tasks"`        // The tasks for any related requests
	Body        [][]string        `json:"body"`         // The body containing the command information
	Quantities  []Quantity        `json:"quantifiers"`  // Quantifiers, e.g. for queries
	QueryParams []QueryParam      `json:"query_params"` // The parameters for a query
}

// Type representing a named task with start and end times.
type Task struct {
	Name     string
	Started  time.Time
	Ended    time.Time
	HasEnded bool
}

// Initiate a new task, started just now.
func NewTask(name string) *Task {
	task := FreshTask(name)
	return &task
}

func FreshTask(name string) Task {
	return Task{Name: name, Started: rightNow(), HasEnded: false}
}

func IdleTask() Task {
	t := rightNow()
	return Task{Name: "", Started: t, Ended: t, HasEnded: true}
}

// Stop the task.
func (t *Task) Stop() {
	if !t.HasEnded {
		t.Ended = rightNow()
		t.HasEnded = true
	}
}

func (t *Task) IsRunning() bool {
	return !t.HasEnded
}

// The current local time, truncated to seconds.
func rightNow() time.Time {
	return time.Now().Truncate(time.Second)
}

// Type repserenting a server's response to a client's request.
type Response struct {
	Status string     `json:"status"`
	Error  string     `json:"error"`
	Body   [][]string `json:"body"`
}

// Type representing summary of a single request.
type Summary struct {
	Task    string
	Details Quantity
	Total   time.Duration
	Start   time.Time
	End     time.Time
}

func (r *Response) SetError(err error) {
	if err == nil {
		return
	}

	if r.Failed() {
		r.Status = RespError
	}
	r.Error = err.Error()
}

func (r *Response) Failed() bool {
	return r.Status == RespError
}

func (r *Response) SetListening() {
	if !r.Failed() {
		r.Status = RespSuccess
	}
	r.addToBody(line("Listening"))
}

func (r *Response) AddPong() {
	pongTime := time.Now().Format(time.RFC3339)
	r.addToBody(line(pongTime))
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

func (r *Response) AddShutdownMessage() {
	if !r.statusIsSet() {
		r.Status = RespSuccess
	}
	r.addToBody(line("Server shutting down: " + formatTime(time.Now())))
}

// Create a response containing the given query summaries.
func (r *Response) AddQuerySummaries(sum []Summary) {
	if !r.statusIsSet() {
		r.Status = RespSuccess
	}
	for _, s := range sum {
		header := []string{s.Task}
		header = append(header, s.Details.Type)
		header = append(header, s.Details.Elems...)
		r.addToBody(line(strings.Join(header, " ")))
		r.addToBody(line("First logged", formatTime(s.Start)))
		r.addToBody(line("Last logged", formatTime(s.End)))
		r.addToBody(line("Total time", s.Total.String()))
	}
}

// The error encapsulated in the response, if any.
func (r *Response) Err() error {
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
