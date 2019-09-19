package config

import (
	"fmt"
	"os"
	"path/filepath"
)

const (
	DebugNone = 0
	DebugSome = 1
	DebugAll  = 2
)

// Configuration parameters.
type Opts struct {
	ConfDir     string // Where to keep the DB file.
	TempDir     string // Where to keep the domain socket.
	DBFileName  string // The name of the DB file.
	ReqSockName string // The name of the request socket file.
	NtfSockName string // The name of the notification socket file.
	DebugLevel  int    // Determines the amount of additional log output.
}

// The socket to use for requests to the server.
func (p *Opts) RequestSocket() string {
	return filepath.Join(p.TempDir, p.ReqSockName)
}

// The socket to connect to for notification task changes.
func (p *Opts) NotificationSocket() string {
	return filepath.Join(p.TempDir, p.NtfSockName)
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
		ConfDir:     confDir,
		TempDir:     tempDir,
		DBFileName:  "tilo.db",
		ReqSockName: "server",
		NtfSockName: "notify",
		DebugLevel:  DebugAll, // TODO: Make this a non-default and flexible.
	}, nil
}
