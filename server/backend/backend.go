// Package backend describes possible database backends for the server.
package backend

import (
	"time"

	"github.com/fgahr/tilo/config"
	"github.com/fgahr/tilo/msg"
)

// Backend represents storage of task information, typically a database.
// TODO: Figure out how to handle malfunctions in remote backends.
type Backend interface {
	Name() string
	Init() error
	Close() error
	Save(task msg.Task) error
	Config() config.BackendConfig
	// RecentTasks gives a summary of the latest activity, limited to the `maxNumber` most recent tasks
	RecentTasks(maxNumber int) ([]msg.Summary, error)
	// TODO: Split into several meaningful methods?
	GetTaskBetween(task string, start time.Time, end time.Time) ([]msg.Summary, error)
	GetAllTasksBetween(start time.Time, end time.Time) ([]msg.Summary, error)
}

var backends = make(map[string]Backend)

// RegisterBackend needs to be called to make a backend available for use.
func RegisterBackend(b Backend) {
	if backends[b.Name()] != nil {
		panic("Double registration of backend with name " + b.Name())
	}
	backends[b.Name()] = b
	config.RegisterBackend(b.Config())
}

// From determines and sets up a backend based on configuration options.
func From(conf *config.Opts) Backend {
	// TODO: Adjust to conf
	return backends["sqlite3"]
}
