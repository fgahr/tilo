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
	CLI_VAR_PREFIX = "--"
)

func GetConfig(args []string) (*Opts, []string) {
	// TODO
	return defaultConfig(), args
}

// Configuration parameters.
type Opts struct {
	ConfFile string `cli:"config-file" env:"CONFIG_FILE"`  // The location of the configuration file.
	DBFile   string `cli:"db-file" env:"DB_FILE"`          // The name of the DB file.
	Socket   string `cli:"socket" env:"SERVER_SOCKET"`     // The name of the request socket file.
	Protocol string `cli:"protocol" env:"SOCKET_PROTOCOL"` // The protocol to use for server communication.
	LogLevel int    `cli:"log-level" env:"LOG_LEVEL"`      // Determines the amount of additional log output.
}

func (c *Opts) ConfigDir() string {
	return filepath.Dir(c.ConfFile)
}

func (c *Opts) SocketDir() string {
	return filepath.Dir(c.Socket)
}

// Emit the configuration in a format suitable as environment variables.
func (c *Opts) AsEnvKeyValue() []string {
	v := reflect.ValueOf(*c)
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
func defaultConfig() *Opts {
	socket := filepath.Join(os.TempDir(), fmt.Sprintf("%s%d", "tilo", os.Getuid()), "server")
	// There's nothing we can do with an error here so we ignore it.
	homeDir, _ := os.UserHomeDir()
	confFile := filepath.Join(homeDir, ".config", "tilo", "config")
	dbFile := filepath.Join(homeDir, ".config", "tilo", "tilo.db")
	return &Opts{
		ConfFile: confFile,
		DBFile:   dbFile,
		Socket:   socket,
		Protocol: "unix",
		LogLevel: LOG_INFO,
	}
}

func (c *Opts) ApplyCommandLine(args []string) *Opts {
	// TODO
	return c
}
