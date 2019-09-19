// Package client describes all client-side operations.
package client

import (
	"fmt"
	"github.com/fgahr/tilo/config"
	"github.com/fgahr/tilo/msg"
	"github.com/fgahr/tilo/server"
	"github.com/pkg/errors"
	"io"
	"net"
	"net/rpc"
	"net/rpc/jsonrpc"
	"os"
	"text/tabwriter"
	"time"
)

// A struct holding a connection to the server and performing communication
// with it.
type Client struct {
	conn           net.Conn       // Connection to the communication socket
	requestTimeout time.Duration  // Timeout for requests
	Conf           *config.Opts // Configuration for this process
	rpcClient      *rpc.Client    // RPC Client to call server-side functions
	err            error          // Any error that may have occured
}

// Create a new client to communicate with the server.
func NewClient(params *config.Opts) (*Client, error) {
	c := Client{
		conn:           nil,
		requestTimeout: 5 * time.Second,
		Conf:           params,
	}
	return &c, nil
}

// Interact with the server based on the program's line arguments.
func (c *Client) HandleArgs(args []string) error {
	c.ensureServerIsRunning()
	fnName, request := c.parseArgs(args)
	c.connectToRequestSocket()
	c.performRequest(fnName, request)
	return c.err
}

// Listen to notifications from the server and print them to stdout.
func (c *Client) PrintNotifications(w io.Writer) error {
	c.ensureServerIsRunning()
	conn, err := net.Dial("unix", c.Conf.NotificationSocket())
	if err != nil {
		return errors.Wrap(err, "Cannot connect to socket")
	}
	defer conn.Close()
	_, err = io.Copy(w, conn)
	return errors.Wrap(err, "Transmission failed")
}

func (c *Client) parseArgs(args []string) (string, msg.Request) {
	if c.err != nil {
		return "", msg.Request{}
	}
	fnName, request, err := msg.ParseRequest(args, time.Now())
	if err != nil {
		c.err = errors.Wrap(err, "Unable to parse command line arguments")
		return "", msg.Request{}
	}
	return fnName, request
}

// Close the client's connection to the server.
func (c *Client) Close() error {
	if c.conn == nil {
		return errors.New("Client is not connected.")
	}
	err := c.conn.Close()
	if err != nil {
		return err
	}
	return c.err
}

// Establish a server connection.
func (c *Client) connectToRequestSocket() {
	if c.err != nil {
		return
	}
	rpcClient, err := jsonrpc.Dial("unix", c.Conf.RequestSocket())
	if err != nil {
		c.err = err
	}
	c.rpcClient = rpcClient
}

// Perform a request-response-cycle, evaluating the server response to the request.
func (c *Client) performRequest(fnName string, req msg.Request) {
	if c.err != nil {
		return
	}

	var resp msg.Response
	err := c.rpcClient.Call(fnName, req, &resp)
	if err != nil {
		c.err = errors.Wrapf(
			err, "Unable to call remote procedure %s for request %v", fnName, req)
		return
	}

	err = resp.Err()
	if err != nil {
		c.err = err
		return
	} else {
		c.err = c.printResponse(resp)
		return
	}
}

// Print a response as formatted output.
func (c *Client) printResponse(resp msg.Response) error {
	// NOTE: This function could easily exist without depending on a client.
	// However, this allows to configure the output in some way at a later date.
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

// If the server is not already running, start it in a new background thread
// and wait for it to come online.
func (c *Client) ensureServerIsRunning() {
	if c.err != nil {
		return
	}
	// If connected we already know it is running.
	if c.conn != nil {
		return
	}

	// Query server status.
	running, err := server.IsRunning(c.Conf)
	if err != nil {
		c.err = errors.Wrap(err, "Could not determine server status")
		return
	}
	if running {
		return
	}

	// Start server if it isn't running.
	err = server.StartInBackground(c.Conf)
	if err != nil {
		c.err = errors.Wrap(err, "Could not start server")
	}

	// Wait for server to become available
	notifyChan := make(chan struct{})
	go func(ch chan<- struct{}) {
		for {
			up, _ := server.IsRunning(c.Conf)
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
