package config

import (
	"fmt"
	"os"
	"path/filepath"
)

const (
	LOG_OFF = iota
	LOG_TRACE
	LOG_WARN
	LOG_INFO
	LOG_DEBUG
)

// Configuration parameters.
type Opts struct {
	ConfDir    string // Where to keep the DB file.
	TempDir    string // Where to keep the domain socket.
	DBFileName string // The name of the DB file.
	SocketName string // The name of the request socket file.
	LogLevel   int    // Determines the amount of additional log output.
}

// The socket to use for requests to the server.
func (p *Opts) ServerSocket() string {
	return filepath.Join(p.TempDir, p.SocketName)
}

// The database file used by SQLite.
func (p *Opts) DBFile() string {
	return filepath.Join(p.ConfDir, p.DBFileName)
}

// Create a set of default parameters.
func DefaultConfig() (*Opts, error) {
	tempDir := filepath.Join(os.TempDir(), fmt.Sprintf("%s%d", "tilo", os.Getuid()))
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	confDir := filepath.Join(homeDir, ".config", "tilo")
	return &Opts{
		ConfDir:    confDir,
		TempDir:    tempDir,
		DBFileName: "tilo.db",
		SocketName: "server",
		LogLevel:   LOG_TRACE, // TODO: Make this a non-default and flexible.
	}, nil
}
