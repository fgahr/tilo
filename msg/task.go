// Package msg provides means for client and server to communicate.
// TODO: Move to server package? Into backend?
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
	task := FreshTask(name)
	return &task
}

func FreshTask(name string) Task {
	return Task{Name: name, Started: rightNow(), HasEnded: false}
}


func NewIdleTask() *Task {
	t := rightNow()
	return &Task{Name: "", Started: t, Ended: t, HasEnded: true}
}

// Stop the task.
func (t *Task) Stop() time.Time {
	if !t.HasEnded {
		t.Ended = rightNow()
		t.HasEnded = true
	}
	return t.Ended
}

func (t *Task) IsRunning() bool {
	return !t.HasEnded
}

// The current local time, truncated to seconds.
func rightNow() time.Time {
	return time.Now().Truncate(time.Second)
}
