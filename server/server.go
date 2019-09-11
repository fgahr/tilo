// Package server describes all server-side operations.
package server

import (
	"github.com/fgahr/tilo/config"
	"github.com/fgahr/tilo/msg"
	"github.com/fgahr/tilo/server/db"
	"github.com/fgahr/tilo/server/gui"
	_ "github.com/mattn/go-sqlite3"
	"github.com/pkg/errors"
	"github.com/therecipe/qt/widgets"
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
	// FIXME: Shutdown being communicated via a simple variable is bad in a
	// setting featuring some (albeit primitive) concurrency as it may not be
	// properly shared across CPU cores. Consider closing a channel instead.
	shuttingDown bool            // True when shutting down
	shutdownChan chan struct{}   // Used to communicate shutdown requests
	conf         *config.Params  // Configuration parameters for this instance
	handler      *RequestHandler // Client request handler
	rpcEndpoint  *rpc.Server     // Server for RPC requests
	listener     net.Listener    // Listener for the client request socket
}

// Start server operation.
// This function will block until server shutdown.
func Run(conf *config.Params) error {
	s := newServer(conf)
	if err := s.init(); err != nil {
		return errors.Wrap(err, "Failed to initialize server")
	}
	// Ensure clean shutdown if at all possible. This cannot be moved to
	// s.main() because that might not be the main goroutine.
	defer s.enforceCleanup()

	if conf.Gui {
		qApp := gui.SetUpSystemTrayWidget(conf)
		if qApp != nil {
			go s.main(qApp)
			ret := qApp.Exec()
			if ret == 0 {
				return nil
			} else {
				return errors.Errorf("Caught error code %d.", ret)
			}
		}
	}

	if !conf.Gui {
		s.main(nil)
		return nil
	} else {
		// If GUI setup fails, the corresponding config option should be set
		// to false and hence if this point is reached we have a logic error.
		panic("Inconsistent code path: GUI option undecided")
	}
}

// Create and configure a new server.
func newServer(conf *config.Params) *server {
	s := new(server)
	s.conf = conf
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

// Start the server, initiating required connections.
func (s *server) init() error {
	running, err := IsRunning(s.conf)
	if err != nil {
		return err
	}

	if running {
		return errors.New("Cannot start server: Already running.")
	}

	// Shutdown channel needs to be buffered to avoid deadlock.
	// FIXME: To support proper concurrent server operation, buffer size needs
	// to match concurrent thread count. This is not an issue yet.
	s.shutdownChan = make(chan struct{}, 1)

	// Create directories if necessary
	err = ensureDirExists(s.conf.ConfDir)
	if err != nil {
		return err
	}

	err = ensureDirExists(s.conf.TempDir)
	if err != nil {
		return err
	}

	handler := RequestHandler{conf: s.conf, shutdownChan: s.shutdownChan, activeTask: nil}
	// Establish database connection.
	backend, err := db.NewBackend(s.conf)
	if err != nil {
		s.listener.Close()
		backend.Close()
		return err
	}

	handler.backend = backend
	s.handler = &handler
	// Establish socket connection.
	listener, err := net.Listen("unix", s.conf.Socket())
	if err != nil {
		return err
	}
	s.listener = listener

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
func (s *server) main(qApp *widgets.QApplication) {
	// Signal channel needs to be buffered, see documentation.
	signalChan := make(chan os.Signal, 1)
	connectChan := make(chan net.Conn)
	defer close(signalChan)
	defer close(connectChan)

	defer

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

	if qApp != nil {
		qApp.Exit(0)
	}
}

// Wait for a client to connect. Send connections to the given channel.
func (s *server) waitForConnection(connectChan chan<- net.Conn) {
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
func (s *server) serveConnection(conn net.Conn) {
	codec := jsonrpc.NewServerCodec(conn)
	s.rpcEndpoint.ServeCodec(codec)
}

// Initiate shutdown, closing open connections.
func (s *server) shutdown() {
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
