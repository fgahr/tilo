// Package server describes all server-side operations.
package server

import (
	"encoding/json"
	"github.com/fgahr/tilo/config"
	"github.com/fgahr/tilo/msg"
	"github.com/fgahr/tilo/server/db"
	_ "github.com/mattn/go-sqlite3"
	"github.com/pkg/errors"
	"io"
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
	s := Server{conf: conf}
	if err := s.init(); err != nil {
		return errors.Wrap(err, "Failed to initialize server")
	}

	// Ensure clean shutdown if at all possible.
	defer s.enforceCleanup()
	defer close(s.shutdownChan)

	s.main()
	return nil
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
	backend := db.NewBackend(s.conf)
	if err := backend.Init(); err != nil {
		s.socketListener.Close()
		backend.Close()
		return err
	} else {
		s.backend = backend
	}

	// Open request socket.
	if requestListener, err := net.Listen("unix", s.conf.ServerSocket()); err != nil {
		return err
	} else {
		s.socketListener = requestListener
	}

	s.CurrentTask = msg.IdleTask()

	return nil
}

// Enforce cleanup when the server stops.
func (s *Server) enforceCleanup() {
	if r := recover(); r != nil {
		s.logWarn("Shutting down.", r)
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

	s.logDebug("Starting server main loop.")
MainLoop:
	for {
		select {
		case conn := <-srvChan:
			s.serveConnection(conn)
		case sig := <-sigChan:
			s.logDebug("Received signal: ", sig)
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
			s.logError(errors.Wrap(err, "Error listening for connections"))
		} else {
			srvChan <- conn
		}
	}
}

// Serve a notification listener connection, keeping it open.
func (s *Server) serveConnection(conn net.Conn) {
	dec := json.NewDecoder(conn)
	cmd := msg.Cmd{}
	if err := dec.Decode(&cmd); err != nil {
		s.logError(errors.Wrap(err, "Failed to decode command"))
	}
	if err := s.Dispatch(&Request{conn, cmd}); err != nil {
		s.logError(errors.Wrap(err, "Unable to execute command"))
	}
}

func (s *Server) Dispatch(req *Request) error {
	s.logCommand(req.Cmd)
	command := req.Cmd.Op
	op := operations[command]
	if op == nil {
		return errors.New("No such operation: " + command)
	}
	op.ServerExec(s, req)
	return nil
}

// Send a notification to all registered listeners.
func (s *Server) notifyListeners() {
	ntf := TaskNotification(s.CurrentTask)
	s.logDebug("Notifying listeners:", ntf)
	if len(s.listeners) > 0 {
		remainingListeners := make([]NotificationListener, 0)
		for _, lst := range s.listeners {
			if err := lst.Notify(ntf); err != nil {
				s.logInfo("Could not notify listener, disconnecting:", err)
				lst.disconnect()
			} else {
				remainingListeners = append(remainingListeners, lst)
			}
		}
		s.listeners = remainingListeners
	}
}

// Notify all connected listeners of shutdown and disconnect them.
func (s *Server) disconnectAllListeners() {
	ntf := shutdownNotification()
	for _, lst := range s.listeners {
		lst.Notify(ntf)
		if err := lst.disconnect(); err != nil {
			s.logWarn("Error closing listener connection:", err)
		}
	}
}

// Initiate shutdown, closing open connections.
func (s *Server) shutdown() {
	var err error
	s.logInfo("Shutting down server..")
	// When the shutdown is initiated by a message, the task is stopped prior.
	// If shutdown is in response to a signal, there is nothing else to do here.
	s.StopCurrentTask()

	if len(s.listeners) > 0 {
		s.logInfo("Disconnecting listeners")
		s.disconnectAllListeners()
	}

	s.logInfo("Closing socket..")
	err = s.socketListener.Close()
	if err != nil {
		s.logError(err)
	} else {
		s.logInfo("OK")
	}

	s.logInfo("Removing temporary directory..")
	err = os.RemoveAll(s.conf.TempDir)
	if err != nil {
		s.logError(err)
	} else {
		s.logInfo("OK")
	}

	s.logInfo("Shutdown complete.")
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
	// NOTE: Due to dependency resolution issues, there is no direct way to tie
	// the arguments to the corresponding operation and its arguments. It could
	// be done indirectly.
	proc, err := os.StartProcess(executable, []string{executable, "server", "run"}, &procAttr)
	if err != nil {
		return 0, errors.Wrap(err, "Unable to start server process")
	}
	return proc.Pid, nil
}

// Serialize obj to JSON, add a linebreak, and send it to the writer.
func writeJsonLine(obj interface{}, w io.Writer) error {
	data, err := json.Marshal(obj)
	if err != nil {
		panic(err)
	}
	// Ending messages with a linebreak makes writing listeners easier.
	data = append(data, '\n')
	_, err = w.Write(data)
	return err
}

func (s *Server) logError(err error) {
	if err == nil {
		return
	}
	if s.conf.LogLevel >= config.LOG_OFF {
		log.Println(err)
	}
}

func (s *Server) logWarn(msg ...interface{}) {
	if s.conf.LogLevel >= config.LOG_WARN {
		log.Println(msg...)
	}
}

func (s *Server) logFmtWarn(format string, v ...interface{}) {
	if s.conf.LogLevel >= config.LOG_WARN {
		log.Printf(format, v...)
	}
}

func (s *Server) logInfo(msg ...interface{}) {
	if s.conf.LogLevel >= config.LOG_INFO {
		log.Println(msg...)
	}
}

func (s *Server) logFmtInfo(format string, v ...interface{}) {
	if s.conf.LogLevel >= config.LOG_INFO {
		log.Printf(format, v...)
	}
}

func (s *Server) logDebug(msg ...interface{}) {
	if s.conf.LogLevel >= config.LOG_DEBUG {
		log.Println(msg...)
	}
}

func (s *Server) logFmtDebug(format string, v ...interface{}) {
	if s.conf.LogLevel >= config.LOG_DEBUG {
		log.Printf(format, v...)
	}
}
