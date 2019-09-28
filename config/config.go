package config

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
)

const (
	LOG_OFF = iota
	LOG_TRACE
	LOG_WARN
	LOG_INFO
	LOG_DEBUG
)

const (
	ENV_VAR_PREFIX = "__TILO_"
)

// Configuration parameters.
type Opts struct {
	ConfDir    string `env:"CONFIG_DIR"`    // Where to keep the DB file.
	TempDir    string `env:"TEMP_DIR"`      // Where to keep the domain socket.
	DBFileName string `env:"DB_FILE_NAME"`  // The name of the DB file.
	Socket     string `env:"SERVER_SOCKET"` // The name of the request socket file.
	LogLevel   int    `env:"LOG_LEVEL"`     // Determines the amount of additional log output.
}

// The socket to use for requests to the server.
func (p *Opts) ServerSocket() string {
	return filepath.Join(p.TempDir, p.Socket)
}

// The database file used by SQLite.
func (p *Opts) DBFile() string {
	return filepath.Join(p.ConfDir, p.DBFileName)
}

func (p *Opts) AsEnvironVars() []string {
	v := reflect.ValueOf(*p)
	result := make([]string, v.NumField())
	for i := 0; i < v.NumField(); i++ {
		fieldInfo := v.Type().Field(i)
		tag := fieldInfo.Tag
		name := tag.Get("env")
		result[i] = fmt.Sprintf("%s%s=%v", ENV_VAR_PREFIX, name, v.Field(i))

	}
	return result
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
		Socket:     "server",
		LogLevel:   LOG_TRACE, // TODO: Make this a non-default and flexible.
	}, nil
}
