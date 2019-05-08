// Package msg provides means for client and server to communicate.
package msg

import (
	"time"
)

// Type representing a named task with start and end times.
type Task struct {
	Name     string
	Started  time.Time
	Ended    time.Time
	HasEnded bool
}

// Initiate a new task, started just now.
func NewTask(name string) *Task {
	return &Task{Name: name, Started: rightNow(), HasEnded: false}
}

// Stop the task.
func (t *Task) Stop() {
	t.Ended.Sub(t.Started).Seconds()
	if t.HasEnded {
		return
	} else {
		t.Ended = rightNow()
		t.HasEnded = true
	}
}

// The current local time, truncated to seconds.
func rightNow() time.Time {
	return time.Now().Truncate(time.Second)
}
