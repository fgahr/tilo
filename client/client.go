// Package client describes all client-side operations.
package client

import (
	"encoding/json"
	"fmt"
	"github.com/fgahr/tilo/argparse"
	"github.com/fgahr/tilo/config"
	"github.com/fgahr/tilo/msg"
	"github.com/fgahr/tilo/server"
	"github.com/pkg/errors"
	"io"
	"net"
	"os"
	"text/tabwriter"
	"time"
)

var operations = make(map[string]ClientOperation)

type ClientOperation interface {
	// Execute client-side behaviour based on args
	ClientExec(cl *Client, cmd msg.Cmd) error
	// Command line argument parser for this operation
	Parser() *argparse.Parser
	// Write the operation's usage information to the given writer
	PrintUsage(w io.Writer)
}

// Make a client-side operation available.
func RegisterOperation(name string, operation ClientOperation) {
	operations[name] = operation
}

// Execute the appropriate action based on the configuration and the arguments.
func Dispatch(conf *config.Opts, args []string) bool {
	if len(args) == 0 {
		panic("Empty argument list")
	}
	command := args[0]
	op := operations[command]
	if op == nil {
		panic("No such command: " + command)
	}
	cl := newClient(conf)
	if cmd, err := op.Parser().Parse(args[1:]); err != nil {
		// TODO: Wrap with operation's usage etc.
		cl.PrintMessage(err.Error())
		op.PrintUsage(os.Stderr)
		return false
	} else if err := op.ClientExec(cl, cmd); err != nil {
		cl.PrintMessage(err.Error())
		return false
	} else {
		return true
	}
}

type Client struct {
	conf   *config.Opts
	conn   net.Conn
	msgout io.Writer
	err    error
}

// Read from the client's connection.
func (cl *Client) Read(p []byte) (n int, err error) {
	if cl.Failed() {
		return 0, errors.Wrap(cl.err, "Cannot read from socket. Previous error")
	}
	if cl.conn == nil {
		panic("Connection not yet established.")
	}
	return cl.conn.Read(p)
}

func newClient(conf *config.Opts) *Client {
	return &Client{conf: conf, msgout: os.Stderr}
}

func (c *Client) Failed() bool {
	return c.err != nil
}

func (c *Client) Connected() bool {
	return c.conn != nil
}

// Close the client's underlying connection.
func (c *Client) Close() error {
	err := c.conn.Close()
	if !c.Failed() {
		// NOTE: c.err can still be nil afterwards
		c.err = err
	}
	return err
}

// The first error the client may have encountered. Nil on successful operation.
func (c *Client) Error() error {
	return c.err
}

// Send the command to the server, receive and print the response.
func (c *Client) SendReceivePrint(cmd msg.Cmd) {
	c.EstablishConnection()
	c.SendToServer(cmd)
	resp := c.ReceiveFromServer()
	c.PrintResponse(resp)
}

// Establish a connection to the server.
// Will start the server if it isn't running yet.
func (c *Client) EstablishConnection() {
	if c.Failed() {
		return
	}
	c.EnsureServerIsRunning()
	socket := c.conf.Socket.Value
	if conn, err := net.Dial(c.conf.Protocol.Value, socket); err != nil {
		c.err = errors.Wrap(err, "Failed to connect to socket "+socket)
	} else {
		c.conn = conn
	}
}

// Send the command to the server.
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

// Receive a response from the server.
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
	// FIXME: Pre-failure parts of the response should be printed as well.
	// Response type might be rewritten.
	if resp.Failed() {
		c.err = resp.Err()
	} else {
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
}

// Make sure the server is running, start it if necessary.
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

// Whether the server appears to be running.
func (c *Client) ServerIsRunning() bool {
	running, _ := server.IsRunning(c.conf)
	return running
}

// Run the server in the foreground.
func (c *Client) RunServer() {
	c.err = server.Run(c.conf)
}

// Print a message for the user.
func (c *Client) PrintMessage(message string) {
	fmt.Fprintln(c.msgout, message)
}

// Print an error message for the user.
func (c *Client) PrintError(err error) {
	fmt.Fprintln(c.msgout, err.Error())
}
