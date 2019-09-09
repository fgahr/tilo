// Package server describes all server-side operations.
package server

import (
	"github.com/fgahr/tilo/config"
	"github.com/fgahr/tilo/msg"
	"github.com/fgahr/tilo/server/db"
	_ "github.com/mattn/go-sqlite3"
	"github.com/pkg/errors"
	"log"
	"net"
	"net/rpc"
	"net/rpc/jsonrpc"
	"os"
	"os/signal"
	"syscall"
)

type Server struct {
	// FIXME: Shutdown being communicated via a simple variable is bad in a
	// setting featuring some (albeit primitive) concurrency as it may not be
	// properly shared across CPU cores.
	shuttingDown bool            // True when shutting down
	shutdownChan chan struct{}   // Used to communicate shutdown requests
	Conf         *config.Params  // Configuration parameters for this instance
	handler      *RequestHandler // Client request handler
	rpcEndpoint  *rpc.Server     // Server for RPC requests
	listener     net.Listener    // Listener for the client request socket
}

func Run(params *config.Params) error {
	s := newServer(params)
	if err := s.init(); err != nil {
		return errors.Wrap(err, "Failed to initialize server")
	}
	s.main()
	return nil
}

func newServer(params *config.Params) *Server {
	s := new(Server)
	s.Conf = params
	return s
}

// Check whether the server is running.
func IsRunning(params *config.Params) (bool, error) {
	_, err := os.Stat(params.Socket())
	if os.IsNotExist(err) {
		return false, nil
	} else if err != nil {
		return false, errors.Wrap(err, "Could not determine server status")
	}
	return true, nil
}

// Make sure the configuration directory exists, creating it if necessary.
func ensureDirExists(dir string) error {
	return os.MkdirAll(dir, 0700)
}

// Start the server.
func (s *Server) init() error {
	running, err := IsRunning(s.Conf)
	if err != nil {
		return err
	}

	if running {
		return errors.New("Cannot start server: Already running.")
	}

	// Create directories if necessary
	err = ensureDirExists(s.Conf.ConfDir)
	if err != nil {
		return err
	}

	err = ensureDirExists(s.Conf.TempDir)
	if err != nil {
		return err
	}

	handler := RequestHandler{Conf: s.Conf, server: s, activeTask: nil}
	// Establish database connection.
	backend, err := db.NewBackend(s.Conf)
	if err != nil {
		s.listener.Close()
		backend.Close()
		return err
	}

	handler.backend = backend
	s.handler = &handler
	// Establish socket connection.
	listener, err := net.Listen("unix", s.Conf.Socket())
	// listener, err := net.Listen("tcp", "localhost:9999")
	if err != nil {
		return err
	}
	s.listener = listener

	rpcEndpoint := rpc.NewServer()
	rpcEndpoint.Register(&handler)
	s.rpcEndpoint = rpcEndpoint

	// Shutdown channel needs to be buffered to avoid deadlock.
	// FIXME: To support proper concurrent server operation, buffer size needs
	// to match concurrent thread count. This is not an issue yet.
	s.shutdownChan = make(chan struct{}, 1)

	return nil
}

// Server main loop: process incoming requests.
func (s *Server) main() {
	// Ensure clean shutdown if at all possible.
	defer func() {
		if r := recover(); r != nil {
			log.Println("Encountered panic in Server.main()", r)
		}
		s.shutdown()
	}()
	// Signal channel needs to be buffered, see documentation.
	signalChan := make(chan os.Signal, 1)
	connectChan := make(chan net.Conn)
	defer close(signalChan)
	defer close(connectChan)

	// Enable cleanup on receiving SIGTERM.
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
	// Enable connection processing.
	go s.waitForConnection(connectChan)

	log.Println("Starting server main loop.")
MainLoop:
	for {
		select {
		case conn := <-connectChan:
			s.serveConnection(conn)
		case sig := <-signalChan:
			log.Println("Received signal: ", sig)
			break MainLoop
		case <-s.shutdownChan:
			break MainLoop
		}
	}
}

// Wait for a client to connect. Send connections to the given channel.
func (s *Server) waitForConnection(connectChan chan<- net.Conn) {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			if s.shuttingDown {
				break
			}
			log.Println(err)
		} else {
			connectChan <- conn
		}
	}
}

// Receive a request from the connection and process it. Send a response back.
func (s *Server) serveConnection(conn net.Conn) {
	codec := jsonrpc.NewServerCodec(conn)
	s.rpcEndpoint.ServeCodec(codec)
}

// Initiate shutdown, closing open connections.
func (s *Server) shutdown() {
	var err error
	log.Println("Shutting down server..")
	s.shuttingDown = true
	// If a task is currently still running, stop it first.
	if s.handler.activeTask != nil {
		log.Println("Stopping current task:", s.handler.activeTask.Name)
		err = s.handler.StopCurrentTask(msg.Request{}, nil)
		if err != nil {
			log.Println(err)
		}
	}

	log.Print("Closing domain socket..")
	err = s.listener.Close()
	if err != nil {
		log.Println(err)
	} else {
		log.Println("OK")
	}

	log.Print("Closing database connection..")
	err = s.handler.close()
	if err != nil {
		log.Println(err)
	} else {
		log.Println("OK")
	}

	log.Print("Removing temporary directory..")
	err = os.RemoveAll(s.Conf.TempDir)
	if err != nil {
		log.Println(err)
	} else {
		log.Println("OK")
	}

	log.Println("Shutdown complete.")
}

// Start a server in a background process.
func StartInBackground(params *config.Params) error {
	sysProcAttr := syscall.SysProcAttr{}
	// Prepare high-level process attributes
	err := ensureDirExists(params.ConfDir)
	if err != nil {
		return errors.Wrap(err, "Unable to start server in background")
	}
	procAttr := os.ProcAttr{
		Dir:   params.ConfDir,
		Env:   os.Environ(),
		Files: []*os.File{nil, nil, nil},
		Sys:   &sysProcAttr,
	}

	// No need to keep track of the spawned process
	executable, err := os.Executable()
	if err != nil {
		return errors.Wrap(err, "Unable to start server in background")
	}
	proc, err := os.StartProcess(executable, []string{executable, "server", "run"}, &procAttr)
	if err != nil {
		return errors.Wrap(err, "Unable to start server in background")
	}
	log.Printf("Server started in background process: PID %d\n", proc.Pid)
	return nil
}
