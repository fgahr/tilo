package config

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
)

const (
	LOG_OFF = iota
	LOG_WARN
	LOG_INFO
	LOG_DEBUG
	LOG_TRACE
)

const (
	ENV_VAR_PREFIX = "__TILO_"
	CLI_VAR_PREFIX = "--"
)

type taggedString struct {
	inUse bool
	value string
}

type Item struct {
	InFile string
	InArgs string
	InEnv  string
	Value  string
}

func GetConfig(args []string) (*Opts, []string) {
	// TODO
	return defaultConfig(), args
}

// Configuration parameters.
type Opts struct {
	// The location of the configuration file.
	ConfFile Item
	// The protocol to use for server communication.
	Protocol Item
	// The name of the request socket file.
	Socket Item
	// The server's backend
	Backend Item
	// Determines the amount of additional log output.
	LogLevel Item
}

func (c *Opts) ConfigDir() string {
	return filepath.Dir(c.ConfFile.Value)
}

func (c *Opts) SocketDir() string {
	return filepath.Dir(c.Socket.Value)
}

func (c *Opts) ShouldLogAny() bool {
	return c.logLevelNumeric() > LOG_OFF
}

func (c *Opts) ShouldLogWarnings() bool {
	return c.logLevelNumeric() >= LOG_WARN
}

func (c *Opts) ShouldLogInfo() bool {
	return c.logLevelNumeric() >= LOG_INFO
}

func (c *Opts) ShouldLogDebug() bool {
	return c.logLevelNumeric() >= LOG_DEBUG
}

func (c *Opts) logLevelNumeric() int {
	if c.LogLevel.Value == "" {
		panic("Log level not defined")
	}
	level, _ := strconv.Atoi(c.LogLevel.Value)
	return level
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
	return &Opts{
		ConfFile: Item{InFile: "", InArgs: "conf-file", InEnv: "CONF_FILE", Value: confFile},
		Socket:   Item{InFile: "socket", InArgs: "socket", InEnv: "SOCKET", Value: socket},
		Protocol: Item{InFile: "protocol", InArgs: "protocol", InEnv: "PROTOCOL", Value: "unix"},
		Backend:  Item{InFile: "backend", InArgs: "backend", InEnv: "BACKEND", Value: "sqlite3"},
		LogLevel: Item{InFile: "log_level", InArgs: "log-level", InEnv: "LOG_LEVEL", Value: strconv.Itoa(LOG_INFO)},
	}
}

type BackendConf struct {
	// TODO
}

type BackendConfig interface {
	// The name of the corresponding backend.
	BackendName() string
	// The items accepted by this parser
	AcceptedItems() []*Item
}

var backendParsers = make(map[string]BackendConfig)

func RegisterBackend(bcp BackendConfig) {
	if backendParsers[bcp.BackendName()] != nil {
		panic("Double registration of backend with name " + bcp.BackendName())
	}
	backendParsers[bcp.BackendName()] = bcp
}

func FromFile(configFile string) map[string]taggedString {
	result := make(map[string]taggedString)
	// TODO: Print errors to user
	data, _ := ioutil.ReadFile(configFile)
	asString := string(data)
	lines := strings.Split(asString, "\n")
	for _, fullLine := range lines {
		line := strings.Split(fullLine, "#")[0]
		trimmed := strings.TrimSpace(line)
		keyAndValue := strings.Split(trimmed, "=")
		if len(keyAndValue) == 1 || len(keyAndValue) > 2 {
			// TODO: Error? Warning?
			continue
		}
		key := strings.TrimSpace(keyAndValue[0])
		value := strings.TrimSpace(keyAndValue[1])
		if key == "" || value == "" {
			// TODO: Error? Warning?
			continue
		}
		result[key] = taggedString{false, value}
	}
	return result
}

func FromCommandLineParams(params []string) map[string]taggedString {
	// TODO
	return nil
}

func FromEnvironment() map[string]taggedString {
	result := make(map[string]taggedString)
	env := os.Environ()
	for _, keyValuePair := range env {
		if strings.HasPrefix(keyValuePair, ENV_VAR_PREFIX) {
			keyAndValue := strings.Split(keyValuePair, "=")
			// No need to save empty values
			if keyAndValue[1] == "" {
				continue
			}
			result[keyAndValue[0]] = taggedString{false, keyAndValue[1]}
		}
	}
	return result
}
