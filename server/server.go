// Package server describes all server-side operations.
package server

import (
	"encoding/json"
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
	// "net/rpc"
)

type Server struct {
	// FIXME: Shutdown being communicated via a simple variable is bad in a
	// setting featuring some (albeit primitive) concurrency as it may not be
	// properly shared across CPU cores.
	shuttingDown bool           // True when shutting down
	shutdownChan chan struct{}  // Used to communicate shutdown requests
	activeTask   *msg.Task      // The currently active task, if any
	Conf         *config.Params // Configuration parameters for this instance
	listener     net.Listener   // Listener for the client request socket
	backend      *db.Backend    // Datbase connection
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
	return &Server{Conf: params}
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

	// Establish socket connection.
	listener, err := net.Listen("unix", s.Conf.Socket())
	if err != nil {
		return err
	}
	s.listener = listener

	// Establish database connection.
	// FIXME: Passing a database file breaks the abstraction. Likely the entire
	// config will have to be passed at some point to support a more flexible
	// backend.
	backend := db.NewBackend(s.Conf)
	err = backend.Init()
	if err != nil {
		s.listener.Close()
		backend.Close()
		return err
	}
	s.backend = backend

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
	log.Print("Serving connection from", conn.RemoteAddr())
	decoder := json.NewDecoder(conn)
	encoder := json.NewEncoder(conn)

	var req msg.Request
	err := decoder.Decode(&req)
	if err != nil {
		log.Println("Error decoding request: ", err)
	}

	response := s.handleRequest(req)
	err = encoder.Encode(response)
	if err != nil {
		log.Println("Error encoding response: ", err)
	}
}

// Process the given request, producing a response accordingly.
func (s *Server) handleRequest(req msg.Request) msg.Response {
	if s.Conf.DebugLevel >= config.DebugSome {
		log.Printf("Processing request: %v\n", req)
	}
	var response msg.Response
	var err error = nil
	switch req.Cmd {
	case msg.CmdStart:
		response, err = s.startTimer(req.Tasks[0])
	case msg.CmdStop:
		response, err = s.stopTimer()
	case msg.CmdCurrent:
		response = s.currentTask()
	case msg.CmdAbort:
		response = s.abortCurrentTask()
	case msg.CmdQuery:
		response, err = s.evaluateQuery(req)
	case msg.CmdShutdown:
		// This causes the main loop to exit at the next iteration.
		// Note that the channel needs to be buffered to avoid deadlocking here.
		s.shutdownChan <- struct{}{}
		lastActive := s.activeTask
		_, err = s.stopTimer()
		response = msg.ShutdownResponse(lastActive, err)
	default:
		err = errors.Errorf("Not implemented: %s", req.Cmd)
	}

	if err != nil {
		log.Println(err)
		response = msg.ErrorResponse(err)
	}
	if s.Conf.DebugLevel >= config.DebugAll {
		log.Printf("Returning response: %v\n", response)
	}
	return response
}

// Start a timer for the given arguments, respond its details.
func (s *Server) startTimer(taskName string) (msg.Response, error) {
	oldTask := s.activeTask
	if oldTask != nil {
		_, err := s.stopTimer()
		if err != nil {
			return msg.Response{}, errors.Wrap(err, "Stopping previous timer failed")
		}
	}
	s.activeTask = msg.NewTask(taskName)
	return msg.StartTaskResponse(s.activeTask, oldTask), nil
}

// Stop the current timer, respond its details.
func (s *Server) stopTimer() (msg.Response, error) {
	if s.activeTask == nil {
		return msg.ErrorResponse(errors.New("No active task")), nil
	}
	s.activeTask.Stop()
	err := s.backend.Save(s.activeTask)
	response := msg.StoppedTaskResponse(s.activeTask)
	s.activeTask = nil
	return response, err
}

// Respond about the currently active task.
func (s *Server) currentTask() msg.Response {
	if s.activeTask == nil {
		return msg.ErrorResponse(errors.New("No active task"))
	}
	return msg.CurrentTaskResponse(s.activeTask)
}

// Abort the currently active task without saving it to the backend. Respond
// its details.
func (s *Server) abortCurrentTask() msg.Response {
	if s.activeTask == nil {
		return msg.ErrorResponse(errors.New("No active task"))
	}
	s.activeTask.Stop()
	aborted := s.activeTask
	s.activeTask = nil
	return msg.AbortedTaskResponse(aborted)
}

// Gather a query response from the database.
func (s *Server) evaluateQuery(req msg.Request) (msg.Response, error) {
	var summaries []msg.Summary
	for _, detail := range req.QueryArgs {
		for _, task := range req.Tasks {
			summary, err := s.backend.Query(task, detail)
			if err != nil {
				return msg.Response{}, err
			}
			// One query can given several summaries if all tasks were queried
			for _, taskSummary := range summary {
				summaries = append(summaries, taskSummary)
			}
		}
	}
	return msg.QueryResponse(summaries), nil
}

// Initiate shutdown, closing open connections.
func (s *Server) shutdown() {
	var err error
	log.Println("Shutting down server..")
	s.shuttingDown = true
	// If a task is currently still running, stop it first.
	if s.activeTask != nil {
		log.Println("Stopping current task:", s.activeTask.Name)
		_, err = s.stopTimer()
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
	err = s.backend.Close()
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
