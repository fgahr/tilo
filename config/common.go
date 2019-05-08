package config

import (
	"fmt"
	"os"
	"path/filepath"
)

// Configuration parameters.
type Params struct {
	ConfDir    string // Where to keep the DB file.
	TempDir    string // Where to keep the domain socket.
	DBFileName string // The name of the DB file.
	SocketName string // The name of the socket file.
}
// The socket to use for communication with the server.
func (p *Params) Socket() string {
	return filepath.Join(p.TempDir, p.SocketName)
}

// The database file used by SQLite.
func (p *Params) DBFile() string {
	return filepath.Join(p.ConfDir, p.DBFileName)
}

// Create a set of default parameters.
func DefaultParams() (*Params, error) {
	tempDir := filepath.Join(os.TempDir(), fmt.Sprintf("%s%d", "tilo", os.Getuid()))
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	confDir := filepath.Join(homeDir, ".config", "tilo")
	return &Params{confDir, tempDir, "tilo.db", "server"}, nil
}
