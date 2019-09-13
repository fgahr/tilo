package server

import (
	"encoding/json"
	"net"
	"time"
)

// The notification to send to listeners.
type notification struct {
	task  string    // The name of the task; empty if idle
	since time.Time // Time of the last status change, formatted
}

// An entity awaiting notifications about task changes.
type notificationListener struct {
	conn net.Conn // The connection to notify
}

// A notification informing listeners about server shutdown.
func shutdownNotification() notification {
	// --shutdown is not a valid task name and hence can be used as a signal.
	return notification{"--shutdown", time.Now().Truncate(time.Second)}
}

// Disconnect this listener.
func (lst *notificationListener) disconnect() error {
	if lst == nil {
		return nil
	}
	return lst.conn.Close()
}

// Notify this listener.
func (lst *notificationListener) notify(ntf notification) error {
	data, err := json.Marshal(ntf)
	if err != nil {
		panic(err)
	}
	_, err = lst.conn.Write(data)
	return err
}
