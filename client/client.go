// Package client describes all client-side operations.
package client

import (
	"encoding/json"
	"fmt"
	"github.com/fgahr/tilo/command"
	"github.com/fgahr/tilo/config"
	"github.com/fgahr/tilo/msg"
	"github.com/fgahr/tilo/server"
	"github.com/pkg/errors"
	"net"
	"os"
	"text/tabwriter"
	"time"
)

// Send the command to the server, receive the response.
func SendToServer(conf *config.Opts, cmd command.Cmd) (msg.Response, error) {
	resp := msg.Response{}
	conn, err := EstablishConnection(conf)
	if err != nil {
		return resp, errors.Wrap(err, "Failed to establish connection")
	}
	enc := json.NewEncoder(conn)
	dec := json.NewDecoder(conn)
	if err = enc.Encode(cmd); err != nil {
		return resp, errors.Wrap(err, "Failed to encode command")
	}
	if err = dec.Decode(&resp); err != nil {
		return resp, errors.Wrap(err, "Failed to decode response")
	}
	return resp, nil
}

// Print a response to stdout.
// NOTE: Options might be relevant in the future to determine formatting etc.
func PrintResponse(_ *config.Opts, resp msg.Response) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 4, 1, ' ', 0)
	for _, line := range resp.Body {
		noTab := true
		for _, word := range line {
			if noTab {
				noTab = false
			} else {
				fmt.Fprint(w, "\t")
			}
			fmt.Fprint(w, word)
		}
		fmt.Fprint(w, "\n")
	}
	return w.Flush()
}

// Establish a connection with the server.
func EstablishConnection(conf *config.Opts) (net.Conn, error) {
	if err := EnsureServerIsRunning(conf); err != nil {
		// Nothing useful to add here, just pass it as-is
		return nil, err
	}

	conn, err := net.Dial("unix", conf.ServerSocket())
	return conn, errors.Wrapf(err, "Cannot connect to socket %s", conf.ServerSocket())
}

// If the server is not already running, start it in a new background thread
// and wait for it to come online.
func EnsureServerIsRunning(conf *config.Opts) error {
	// Query server status.
	if running, err := server.IsRunning(conf); err != nil {
		return errors.Wrap(err, "Could not determine server status")
	} else if running {
		return nil
	}

	// Start server if it isn't running.
	if pid, err := server.StartInBackground(conf); err != nil {
		return errors.Wrap(err, "Could not start server")
	} else {
		fmt.Printf("Server started in background process: PID %d\n", pid)
	}

	// Wait for server to become available
	notifyChan := make(chan struct{})
	go func(ch chan<- struct{}) {
		for {
			up, _ := server.IsRunning(conf)
			if up {
				ch <- struct{}{}
				return
			}
			time.Sleep(20 * time.Millisecond)
		}
	}(notifyChan)
	select {
	case <-notifyChan:
		return nil
	// TODO: Make timeout configurable
	case <-time.After(5 * time.Second):
		close(notifyChan)
		return errors.New("Timeout exceeded trying to bring up server.")
	}
}
