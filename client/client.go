// Package client describes all client-side operations.
package client

import (
	"encoding/json"
	"fmt"
	"github.com/fgahr/tilo/config"
	"github.com/fgahr/tilo/msg"
	"github.com/fgahr/tilo/server"
	"github.com/pkg/errors"
	"net"
	"os"
	"text/tabwriter"
	"time"
)

var operations = make(map[string]ClientOperation)

type ClientOperation interface {
	// Execute client-side behaviour based on args
	ClientExec(cl *Client, args ...string) error
}

func RegisterOperation(name string, operation ClientOperation) {
	operations[name] = operation
}

type Client struct {
	conf *config.Opts
	conn net.Conn
	err  error
}

func (cl *Client) Read(p []byte) (n int, err error) {
	return cl.conn.Read(p)
}

func Execute(conf *config.Opts, args []string) error {
	if len(args) == 0 {
		panic("Empty argument list")
	}
	command := args[0]
	op := operations[command]
	if op == nil {
		panic("No such command: " + command)
	}
	cl := newClient(conf)
	return op.ClientExec(cl, args[1:]...)
}

func newClient(conf *config.Opts) *Client {
	return &Client{conf: conf}
}

func (c *Client) Failed() bool {
	return c.err != nil
}

func (c *Client) Connected() bool {
	return c.conn != nil
}

func (c *Client) Close() error {
	err := c.conn.Close()
	if !c.Failed() {
		// NOTE: c.err can still be nil afterwards
		c.err = err
	}
	return err
}

func (c *Client) Error() error {
	return c.err
}

func (c *Client) EstablishConnection() {
	if c.Failed() {
		return
	}
	c.EnsureServerIsRunning()
	socket := c.conf.ServerSocket()
	if conn, err := net.Dial("unix", socket); err != nil {
		c.err = errors.Wrap(err, "Failed to connect to socket"+socket)
	} else {
		c.conn = conn
	}
}

func (c *Client) SendToServer(cmd msg.Cmd) {
	if c.Failed() {
		return
	}
	if !c.Connected() {
		c.err = errors.New("Cannot send: not connected")
	}
	enc := json.NewEncoder(c.conn)
	c.err = errors.Wrap(enc.Encode(cmd), "Failed to send command to server")
}

func (c *Client) ReceiveFromServer() msg.Response {
	resp := msg.Response{}
	if c.Failed() {
		resp.SetError(errors.Wrap(c.err, "Prior failure in communication"))
		return resp
	}
	if !c.Connected() {
		c.err = errors.New("Cannot receive: not connected")
	}
	dec := json.NewDecoder(c.conn)
	c.err = errors.Wrap(dec.Decode(&resp), "Failed to decode response")
	return resp
}

func (c *Client) PrintResponse(resp msg.Response) {
	if c.Failed() {
		return
	}
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
	c.err = w.Flush()
}

func (c *Client) EnsureServerIsRunning() {
	// Query server status.
	if running, err := server.IsRunning(c.conf); err != nil {
		c.err = errors.Wrap(err, "Could not determine server status")
		return
	} else if running {
		return
	}

	// Start server if it isn't running.
	if pid, err := server.StartInBackground(c.conf); err != nil {
		c.err = errors.Wrap(err, "Could not start server")
		return
	} else {
		fmt.Printf("Server started in background process: PID %d\n", pid)
	}

	// Wait for server to become available
	notifyChan := make(chan struct{})
	go func(ch chan<- struct{}) {
		for {
			up, _ := server.IsRunning(c.conf)
			if up {
				ch <- struct{}{}
				return
			}
			time.Sleep(20 * time.Millisecond)
		}
	}(notifyChan)
	select {
	case <-notifyChan:
		return
	// TODO: Make timeout configurable
	case <-time.After(5 * time.Second):
		close(notifyChan)
		c.err = errors.New("Timeout exceeded trying to bring up server.")
	}
}

func (c *Client) RunServer() {
	c.err = server.Run(c.conf)
}
