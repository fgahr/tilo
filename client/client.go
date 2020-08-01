// Package client describes all client-side operations.
package client

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"sort"
	"text/tabwriter"
	"time"

	"github.com/fgahr/tilo/argparse"
	"github.com/fgahr/tilo/client/output"
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
	if len(args) == 0 {
		showUsageAndDie(errors.New("No command given"))
	}

	if args[0] == "-h" || args[0] == "--help" {
		printAllOperationsHelp(os.Stderr)
		return true
	}

	command := args[0]
	op, ok := operations[command]
	if !ok {
		showUsageAndDie(errors.Errorf("No such command: %s", command))
	}

	c := newClient(conf)
	if cmd, err := op.Parser().Parse(args[1:]); err != nil {
		c.printError(err)
		c.printShortDescription(op.DescribeShort())
		return false
	} else if err := op.ClientExec(c, cmd); err != nil {
		c.printError(err)
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
	fmt    output.Formatter
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
	return &Client{conf: conf, msgout: os.Stderr}
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
		header, footer := op.HelpHeaderAndFooter()
		// Summary
		sdesc := op.DescribeShort()
		fmt.Fprintln(c.msgout, "Usage:", os.Args[0], sdesc.Cmd, sdesc.First, sdesc.Second)
		// Header
		fmt.Fprintf(c.msgout, "\n%s\n", header)
		// Describe required task name(s), if any
		if tdesc := op.Parser().TaskDescription(); tdesc != "" {
			fmt.Fprintf(c.msgout, "\nRequired task information\n\t%s\n", tdesc)
		}
		// Parameter description
		if pdesc := op.Parser().ParamDescription(); len(pdesc) > 0 {
			fmt.Fprintf(c.msgout, "\nPossible parameters\n")
			w := tabwriter.NewWriter(c.msgout, 4, 4, 2, ' ', 0)
			for _, par := range pdesc {
				fmt.Fprintf(w, "    %s\t%s\t%s\n",
					par.ParamName, par.ParamValues, par.ParamExplanation)
			}
			w.Flush()
		}
		// Footer
		if footer != "" {
			fmt.Fprintf(c.msgout, "\n%s\n", footer)
		}
		return nil
	} else {
		return errors.Errorf("No such operation: %s", cmd)
	}
}

// PrintAllOperationsHelp prints a command usage overview for the user.
func (c *Client) PrintAllOperationsHelp() {
	printAllOperationsHelp(c.msgout)
}

// Print the help text for all available commands.
func printAllOperationsHelp(out io.Writer) {
	fmt.Fprintf(out,
		"\nUsage: %s [command] <task(s)> <parameters>\n\n", os.Args[0])
	fmt.Fprintln(out, "Available commands")

	w := tabwriter.NewWriter(out, 4, 4, 2, ' ', 0)
	for _, descr := range operationDescriptions() {
		fmt.Fprintf(w, "    %s\t%s\t%s\t%s\n", descr.Cmd, descr.First, descr.Second, descr.What)
	}
	w.Flush()
}

// PrintError prints an error message for the user.
func (c *Client) printError(err error) {
	printError(err, c.msgout)
}

// Print an error message for the user.
func printError(err error, w io.Writer) {
	fmt.Fprintln(w, err.Error())
}

// Print the error, the usage message, then exit with error status.
func showUsageAndDie(err error) {
	printError(err, os.Stderr)
	printAllOperationsHelp(os.Stderr)
	os.Exit(2)
}
