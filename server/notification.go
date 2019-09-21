package server

import (
	"encoding/json"
	"github.com/fgahr/tilo/msg"
	"net"
	"time"
)

// The notification to send to listeners.
type Notification struct {
	Task  string    `json:"task"`  // The name of the task; empty if idle
	Since time.Time `json:"since"` // Time of the last status change, formatted
}

// An entity awaiting notifications about task changes.
type NotificationListener struct {
	conn net.Conn // The connection to notify
}

// A notification informing listeners about server shutdown.
func shutdownNotification() Notification {
	// --shutdown is not a valid task name and hence can be used as a signal.
	return Notification{"--shutdown", time.Now().Truncate(time.Second)}
}

// A notification about a task, presumed to be the currently set one.
// If the task has been stopped, it sends an empty task name, signalling
// idle state.
func taskNotification(t msg.Task) Notification {
	if t.IsRunning() {
		return Notification{Task: t.Name, Since: t.Started}
	} else {
		return Notification{Task: "", Since: t.Ended}
	}
}

// Disconnect this listener.
func (lst *NotificationListener) disconnect() error {
	if lst == nil {
		return nil
	}
	return lst.conn.Close()
}

// Notify this listener.
func (lst *NotificationListener) notify(ntf Notification) error {
	data, err := json.Marshal(ntf)
	if err != nil {
		panic(err)
	}
	data = append(data, '\n')
	_, err = lst.conn.Write(data)
	return err
}
