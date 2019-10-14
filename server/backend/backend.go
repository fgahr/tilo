// Package backend describes possible database backends for the server.
package backend

import (
	"github.com/fgahr/tilo/config"
	"github.com/fgahr/tilo/msg"
	"time"
)

// Type representing a database backend.
// TODO: Figure out how to handle malfunctions in remote backends.
type Backend interface {
	Name() string
	Init() error
	Close() error
	Save(task msg.Task) error
	Config() config.BackendConfig
	// TODO: Split into several meaningful methods?
	GetTaskBetween(task string, start time.Time, end time.Time) ([]msg.Summary, error)
	GetAllTasksBetween(start time.Time, end time.Time) ([]msg.Summary, error)
}

var backends = make(map[string]Backend)

func RegisterBackend(b Backend) {
	if backends[b.Name()] != nil {
		panic("Double registration of backend with name " + b.Name())
	}
	backends[b.Name()] = b
	config.RegisterBackend(b.Config())
}

// Get the appropriate backend.
func From(conf *config.Opts) Backend {
	// TODO: Adjust to conf
	return backends["sqlite3"]
}
