// Package client describes all client-side operations.
package client

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"sort"
	"time"

	"github.com/fgahr/tilo/argparse"
	"github.com/fgahr/tilo/config"
	"github.com/fgahr/tilo/msg"
	"github.com/fgahr/tilo/server"
	"github.com/pkg/errors"
)

var operations = make(map[string]Operation)

// Operation is the common interface for all client-side operations.
type Operation interface {
	// Execute client-side behaviour based on args.
	ClientExec(cl *Client, cmd msg.Cmd) error
	// Command line argument parser for this operation.
	Parser() *argparse.Parser
	// Describe usage for this operation.
	DescribeShort() argparse.Description
	// Header and footer for this operation's help message
	HelpHeaderAndFooter() (string, string)
}

// RegisterOperation makes a client-side operation available.
// This function is called indirectly from other packages' init() functions.
func RegisterOperation(name string, operation Operation) {
	operations[name] = operation
}

// Dispatch to the appropriate command handler based on the given arguments.
// Returns true if all operations succeeded, false otherwise.
func Dispatch(conf *config.Opts, args []string) bool {
	c := newClient(conf)
	if len(args) == 0 {
		c.PrintAllOperationsHelp()
		c.fmt.Error(errors.New("No command given"))
		return false
	}

	if args[0] == "-h" || args[0] == "--help" {
		c.PrintAllOperationsHelp()
		return true
	}

	command := args[0]
	op, ok := operations[command]
	if !ok {
		c.PrintAllOperationsHelp()
		c.fmt.Error(errors.Errorf("No such command: %s", command))
	}

	if cmd, err := op.Parser().Parse(args[1:]); err != nil {
		c.fmt.Error(err)
		c.printShortDescription(op.DescribeShort())
		return false
	} else if err := op.ClientExec(c, cmd); err != nil {
		c.fmt.Error(err)
		return false
	} else {
		return true
	}
}

// Client is a type bundling everything required for client-side operation.
type Client struct {
	conf   *config.Opts
	conn   net.Conn
	msgout io.Writer
	err    error
	fmt    Formatter
}

// Read from the client's connection.
// Client satisfies the io.Reader interface.
func (c *Client) Read(p []byte) (n int, err error) {
	if c.Failed() {
		return 0, errors.Wrap(c.err, "cannot read from socket: preceding error")
	}
	if c.conn == nil {
		panic("cannot read: connection not yet established")
	}
	return c.conn.Read(p)
}

func newClient(conf *config.Opts) *Client {
	f := GetFormatter(conf.Output.Value)
	if f == nil {
		fmt.Fprintf(os.Stderr, "No such output format: %s, using default", conf.Output.Value)
		f = GetFormatter("tabular")
	}
	return &Client{conf: conf, fmt: f}
}

// Failed returns whether the client has encountered an error.
func (c *Client) Failed() bool {
	return c.err != nil
}

// Connected returns whether the client has is connected to the server.
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

// Error returns the first error the client may have encountered, or nil.
func (c *Client) Error() error {
	return c.err
}

// SendReceivePrint executes a typical client lifecycle: a server round-trip.
// This will establish a connection, send the command, receive a response, and
// print it.
func (c *Client) SendReceivePrint(cmd msg.Cmd) {
	c.EstablishConnection()
	c.SendToServer(cmd)
	resp := c.ReceiveFromServer()
	c.PrintResponse(resp)
}

// EstablishConnection ensures the server is up and the client is connected.
func (c *Client) EstablishConnection() {
	if c.Failed() {
		return
	}
	c.EnsureServerIsRunning()
	socket := c.conf.Socket.Value
	if conn, err := net.Dial(c.conf.Protocol.Value, socket); err != nil {
		c.err = errors.Wrap(err, "failed to connect to socket "+socket)
	} else {
		c.conn = conn
	}
}

// SendToServer sends the given command to the server.
func (c *Client) SendToServer(cmd msg.Cmd) {
	if c.Failed() {
		return
	}
	if !c.Connected() {
		c.err = errors.New("cannot send to server: not connected")
	}
	enc := json.NewEncoder(c.conn)
	c.err = errors.Wrap(enc.Encode(cmd), "failed to send command to server")
}

// ReceiveFromServer receives a response from the server.
func (c *Client) ReceiveFromServer() msg.Response {
	resp := msg.Response{}
	if c.Failed() {
		resp.SetError(errors.Wrap(c.err, "preceding failure in communication"))
		return resp
	}
	if !c.Connected() {
		c.err = errors.New("cannot receive from server: not connected")
	}
	dec := json.NewDecoder(c.conn)
	c.err = errors.Wrap(dec.Decode(&resp), "failed to decode response")
	return resp
}

// PrintResponse print a server response for the user to read.
func (c *Client) PrintResponse(resp msg.Response) {
	if c.Failed() {
		return
	}

	if resp.Failed() {
		c.err = resp.Err()
	} else {
		c.fmt.Response(resp)
	}
}

// EnsureServerIsRunning will do nothing if the server is up, else it will start it.
func (c *Client) EnsureServerIsRunning() {
	var running bool
	var err error

	// Query server status.
	running, err = server.IsRunning(c.conf)
	if err != nil {
		c.err = errors.Wrap(err, "unable to determine server status")
		return
	}
	// Nothing to do
	if running {
		return
	}

	// Start the server
	pid, err := server.StartInBackground(c.conf)
	if err != nil {
		c.err = errors.Wrap(err, "Could not start server")
		return
	}
	fmt.Printf("Server started in background process: PID %d\n", pid)

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
		c.err = errors.New("timeout exceeded trying to bring up server")
	}
}

// ServerIsRunning tries to determine whether the server is running.
func (c *Client) ServerIsRunning() bool {
	running, _ := server.IsRunning(c.conf)
	return running
}

// RunServer will yield the current process to a freshly started server.
func (c *Client) RunServer() {
	c.err = server.Run(c.conf)
}

// PrintMessage prints the given message for the user.
func (c *Client) PrintMessage(message string) {
	fmt.Fprintln(c.msgout, message)
}

// Print a short command description to the user.
func (c *Client) printShortDescription(desc argparse.Description) {
	fmt.Fprintln(c.msgout, os.Args[0], desc.Cmd, desc.First, desc.Second, desc.What)
}

// Gather descriptions of operations in alphabetical order.
func operationDescriptions() []argparse.Description {
	var descriptions []argparse.Description
	for _, op := range operations {
		descriptions = append(descriptions, op.DescribeShort())
	}
	byCmdAsc := func(i, j int) bool {
		return descriptions[i].Cmd < descriptions[j].Cmd
	}
	sort.Slice(descriptions, byCmdAsc)

	return descriptions
}

// CommandExists determines whether a command with that name is available.
func (c *Client) CommandExists(cmd string) bool {
	_, ok := operations[cmd]
	return ok
}

// PrintSingleOperationHelp prints the detailed help for a single command.
func (c *Client) PrintSingleOperationHelp(cmd string) error {
	if op, ok := operations[cmd]; ok {
		c.fmt.HelpSingleOperation(op)
		return nil
	}
	return errors.Errorf("No such operation: %s", cmd)
}

// PrintAllOperationsHelp prints a command usage overview for the user.
func (c *Client) PrintAllOperationsHelp() {
	ops := operationDescriptions()
	c.fmt.HelpAllOperations(ops)
}
