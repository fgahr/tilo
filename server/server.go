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

// A tilo server. When the configuration is provided, the remaining fields
// are filled by the .init() method.
type server struct {
	shutdownChan    chan struct{}   // Used to communicate shutdown requests
	conf            *config.Params  // Configuration parameters for this instance
	handler         *RequestHandler // Client request handler
	rpcEndpoint     *rpc.Server     // Server for RPC requests
	reqSockListener net.Listener    // Listener on the client request socket
	ntfSockListener net.Listener    // Listener on the notification socket
}

// Start server operation.
// This function will block until server shutdown.
func Run(conf *config.Params) error {
	s := newServer(conf)
	if err := s.init(); err != nil {
		return errors.Wrap(err, "Failed to initialize server")
	}

	// Ensure clean shutdown if at all possible.
	defer s.enforceCleanup()
	defer close(s.shutdownChan)

	s.main()
	return nil
}

// Create and configure a new server.
func newServer(conf *config.Params) *server {
	s := new(server)
	s.conf = conf
	return s
}

// Check whether the server is running.
func IsRunning(params *config.Params) (bool, error) {
	_, err := os.Stat(params.RequestSocket())
	if os.IsNotExist(err) {
		return false, nil
	} else if err != nil {
		return false, errors.Wrap(err, "Could not determine server status")
	}
	return true, nil
}

// Check whether the server is currently in shutdown.
func (s *server) shuttingDown() bool {
	select {
	case <-s.shutdownChan:
		return true
	default:
		return false
	}
}

// Make sure the configuration directory exists, creating it if necessary.
func ensureDirExists(dir string) error {
	return os.MkdirAll(dir, 0700)
}

// Start the server, initiating required connections.
func (s *server) init() error {
	if running, err := IsRunning(s.conf); err != nil {
		return err
	} else if running {
		return errors.New("Cannot start server: Already running.")
	}

	s.shutdownChan = make(chan struct{})

	// Create directories if necessary
	if err := ensureDirExists(s.conf.ConfDir); err != nil {
		return err
	}

	if err := ensureDirExists(s.conf.TempDir); err != nil {
		return err
	}

	// Establish database connection.
	backend, err := db.NewBackend(s.conf)
	if err != nil {
		s.reqSockListener.Close()
		backend.Close()
		return err
	}

	// Hand backend over to request handler.
	handler := newRequestHandler(s.conf, s.shutdownChan, backend)
	s.handler = handler

	// Open request socket.
	requestListener, err := net.Listen("unix", s.conf.RequestSocket())
	if err != nil {
		return err
	}
	s.reqSockListener = requestListener

	// Open notification socket.
	notificationListener, err := net.Listen("unix", s.conf.NotificationSocket())
	if err != nil {
		return err
	}
	s.ntfSockListener = notificationListener

	// Configure endpoint for remote procedure calls.
	rpcEndpoint := rpc.NewServer()
	rpcEndpoint.Register(&handler)
	s.rpcEndpoint = rpcEndpoint

	return nil
}

// Enforce cleanup when the server stops.
func (s *server) enforceCleanup() {
	if r := recover(); r != nil {
		log.Println("Shutting down.", r)
	}
	s.shutdown()
}

// Server main loop: process incoming requests.
func (s *server) main() {
	// Signal channel needs to be buffered, see documentation.
	signalChan := make(chan os.Signal, 1)
	reqChan := make(chan net.Conn)
	ntfChan := make(chan net.Conn)
	defer close(signalChan)
	defer close(reqChan)
	defer close(ntfChan)

	// Enable cleanup on receiving SIGTERM.
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
	// Enable connection processing.
	go s.waitForConnection(s.reqSockListener, reqChan)
	go s.waitForConnection(s.ntfSockListener, ntfChan)

	log.Println("Starting server main loop.")
MainLoop:
	for {
		select {
		case rconn := <-reqChan:
			s.serveRequestConnection(rconn)
		case nconn := <-ntfChan:
			s.serveNotificationConnection(nconn)
		case sig := <-signalChan:
			log.Println("Received signal: ", sig)
			break MainLoop
		case <-s.shutdownChan:
			break MainLoop
		}
	}
}

// Wait for a client to connect. Send connections to the given channel.
func (s *server) waitForConnection(lst net.Listener, channel chan<- net.Conn) {
	for {
		conn, err := lst.Accept()
		if err != nil {
			if s.shuttingDown() {
				// Ignore shutdown-related errors.
				break
			}
			log.Println(err)
		} else {
			channel <- conn
		}
	}
}

// Receive a request from the connection and process it. Send a response back.
func (s *server) serveRequestConnection(conn net.Conn) {
	codec := jsonrpc.NewServerCodec(conn)
	s.rpcEndpoint.ServeCodec(codec)
}

// Serve a notification listener connection, keeping it open.
func (s *server) serveNotificationConnection(conn net.Conn) {
	s.handler.registerListener(&notificationListener{conn})
}

// Initiate shutdown, closing open connections.
func (s *server) shutdown() {
	var err error
	log.Println("Shutting down server..")
	if s.handler.activeTask != nil {
		log.Println("Aborting current task:", s.handler.activeTask.Name)
		err = s.handler.StopCurrentTask(msg.Request{}, nil)
		if err != nil {
			log.Println(err)
		}
	}

	log.Print("Closing request socket..")
	err = s.reqSockListener.Close()
	if err != nil {
		log.Println(err)
	} else {
		log.Println("OK")
	}

	log.Print("Closing notification socket..")
	err = s.ntfSockListener.Close()
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
	err = os.RemoveAll(s.conf.TempDir)
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
		return errors.Wrap(err, "Unable to determine server executable")
	}
	proc, err := os.StartProcess(executable, []string{executable, "server", "run"}, &procAttr)
	if err != nil {
		return errors.Wrap(err, "Unable to start server process")
	}
	log.Printf("Server started in background process: PID %d\n", proc.Pid)
	return nil
}
