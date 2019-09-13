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
func NewTask(name string) (time.Time, *Task) {
	startTime := rightNow()
	return startTime, &Task{Name: name, Started: startTime, HasEnded: false}
}

// Stop the task.
func (t *Task) Stop() time.Time {
	if !t.HasEnded {
		t.Ended = rightNow()
		t.HasEnded = true
	}
	return t.Ended
}

// The current local time, truncated to seconds.
func rightNow() time.Time {
	return time.Now().Truncate(time.Second)
}
