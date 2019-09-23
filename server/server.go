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
	"os"
	"os/signal"
	"syscall"
)

var operations = make(map[string]ServerOperation)

type Request struct {
	Conn net.Conn
	Cmd  msg.Cmd
}

func (req *Request) Close() error {
	return req.Conn.Close()
}

type ServerOperation interface {
	// Execute server-side behaviour based on the command
	ServerExec(srv *Server, req *Request) error
}

func RegisterOperation(name string, operation ServerOperation) {
	operations[name] = operation
}


// A tilo Server. When the configuration is provided, the remaining fields
// are filled by the .init() method.
type Server struct {
	shutdownChan   chan struct{}          // Used to communicate shutdown requests
	conf           *config.Opts           // Configuration parameters for this instance
	backend        *db.Backend            // The database backend
	socketListener net.Listener           // Listener on the client request socket
	CurrentTask    msg.Task               // The currently active task, if any
	listeners      []NotificationListener // Listeners for task change notifications
}

// Start server operation.
// This function will block until server shutdown.
func Run(conf *config.Opts) error {
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
func newServer(conf *config.Opts) *Server {
	s := new(Server)
	s.conf = conf
	return s
}

// Check whether the server is running.
func IsRunning(conf *config.Opts) (bool, error) {
	_, err := os.Stat(conf.ServerSocket())
	if os.IsNotExist(err) {
		return false, nil
	} else if err != nil {
		return false, errors.Wrap(err, "Could not determine server status")
	}
	return true, nil
}

// Check whether the server is currently in shutdown.
func (s *Server) shuttingDown() bool {
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
func (s *Server) init() error {
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
		s.socketListener.Close()
		backend.Close()
		return err
	}

	// Open request socket.
	requestListener, err := net.Listen("unix", s.conf.ServerSocket())
	if err != nil {
		return err
	}
	s.socketListener = requestListener

	return nil
}

// Enforce cleanup when the server stops.
func (s *Server) enforceCleanup() {
	if r := recover(); r != nil {
		log.Println("Shutting down.", r)
	}
	s.shutdown()
}

// Server main loop: process incoming requests.
func (s *Server) main() {
	// Signal channel needs to be buffered, see documentation.
	sigChan := make(chan os.Signal, 1)
	srvChan := make(chan net.Conn)
	defer close(srvChan)

	// Enable cleanup on receiving SIGTERM.
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	// Enable connection processing.
	go s.waitForConnection(s.socketListener, srvChan)

	log.Println("Starting server main loop.")
MainLoop:
	for {
		select {
		case conn := <-srvChan:
			s.serveConnection(conn)
		case sig := <-sigChan:
			log.Println("Received signal: ", sig)
			break MainLoop
		case <-s.shutdownChan:
			break MainLoop
		}
	}
}

// Wait for a client to connect. Send connections to the given channel.
func (s *Server) waitForConnection(lst net.Listener, srvChan chan<- net.Conn) {
	for {
		if conn, err := lst.Accept(); err != nil {
			if s.shuttingDown() {
				// Ignore shutdown-related errors.
				break
			}
			log.Println(errors.Wrap(err, "Error listening for connections"))
		} else {
			srvChan <- conn
		}
	}
}

func (s *Server) Execute(conn net.Conn, cmd msg.Cmd) error {
	command := cmd.Op
	op := operations[command]
	if op == nil {
		return errors.New("No such operation: " + command)
	}
	op.ServerExec(s, &Request{conn, cmd})
	return nil
}

// Serve a notification listener connection, keeping it open.
func (s *Server) serveConnection(conn net.Conn) {
	// TODO
}

// Send a notification to all registered listeners.
func (s *Server) notifyListeners() {
	ntf := taskNotification(s.CurrentTask)
	if s.conf.DebugLevel == config.DebugAll {
		log.Println("Notifying listeners:", ntf)
	}
	if len(s.listeners) > 0 {
		remainingListeners := make([]NotificationListener, 0)
		for _, lst := range s.listeners {
			if err := lst.notify(ntf); err != nil {
				log.Println("Could not notify listener, disconnecting:", err)
				lst.disconnect()
			} else {
				remainingListeners = append(remainingListeners, lst)
			}
		}
		s.listeners = remainingListeners
	}
}

// Initiate shutdown, closing open connections.
func (s *Server) shutdown() {
	var err error
	log.Println("Shutting down server..")
	// TODO: Handle return values, possibly include in response? Skip?
	s.StopCurrentTask()

	// TODO: Close listener connections
	if len(s.listeners) > 0 {
		log.Println("Disconnecting listeners")
	}
	for _, lst := range s.listeners {
		if err := lst.disconnect(); err != nil {
			log.Println("Error closing listener connection:", err)
		}
	}

	log.Print("Closing socket..")
	err = s.socketListener.Close()
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
func StartInBackground(conf *config.Opts) (int, error) {
	sysProcAttr := syscall.SysProcAttr{}
	// Prepare high-level process attributes
	err := ensureDirExists(conf.ConfDir)
	if err != nil {
		return 0, errors.Wrap(err, "Unable to start server in background")
	}
	procAttr := os.ProcAttr{
		Dir:   conf.ConfDir,
		Env:   os.Environ(),
		Files: []*os.File{nil, nil, nil},
		Sys:   &sysProcAttr,
	}

	// No need to keep track of the spawned process
	executable, err := os.Executable()
	if err != nil {
		return 0, errors.Wrap(err, "Unable to determine server executable")
	}
	proc, err := os.StartProcess(executable, []string{executable, "server", "run"}, &procAttr)
	if err != nil {
		return 0, errors.Wrap(err, "Unable to start server process")
	}
	return proc.Pid, nil
}
