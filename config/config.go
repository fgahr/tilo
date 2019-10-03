// Package config handles everything related to runtime-configuration.
//
// Three configuration sources are supported. In order of ascending priority:
// configuration file, environment variables, command line arguments.
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

type rawConf map[string]taggedString

type Item struct {
	InFile string
	InArgs string
	InEnv  string
	Value  string
}

func nameInFile(item *Item) string {
	return item.InFile
}

func nameInArgs(item *Item) string {
	return item.InArgs
}

func nameInEnv(item *Item) string {
	return item.InEnv
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

type BackendConfig interface {
	// The name of the corresponding backend.
	BackendName() string
	// The items accepted by this parser
	AcceptedItems() []*Item
}

var backendConfigs = make(map[string]BackendConfig)

func RegisterBackend(bcp BackendConfig) {
	if backendConfigs[bcp.BackendName()] != nil {
		panic("Double registration of backend with name " + bcp.BackendName())
	}
	backendConfigs[bcp.BackendName()] = bcp
}

func GetConfig(args []string) (*Opts, []string) {
	conf := defaultConfig()

	fromEnv := FromEnvironment()
	fromArgs, unused := FromCommandLineParams(args)

	// Determine whether we are dealing with an alternative config file location
	apply([]*Item{&conf.ConfFile}, fromEnv, nameInEnv)
	apply([]*Item{&conf.ConfFile}, fromArgs, nameInArgs)
	fromFile := FromFile(conf.ConfFile.Value)

	// Build up the base configuration.
	apply(conf.AcceptedItems(), fromFile, nameInFile)
	apply(conf.AcceptedItems(), fromEnv, nameInEnv)
	apply(conf.AcceptedItems(), fromArgs, nameInArgs)

	// Build up the backend configuration.
	if bc := backendConfigs[conf.Backend.Value]; bc == nil {
		panic("Unknown backend: " + conf.Backend.Value)
	} else {
		apply(bc.AcceptedItems(), fromFile, nameInFile)
		apply(bc.AcceptedItems(), fromEnv, nameInEnv)
		apply(bc.AcceptedItems(), fromArgs, nameInArgs)
	}

	return conf, unused
}

func apply(items []*Item, kvPairs map[string]taggedString, namer func(*Item) string) {
	for _, item := range items {
		if tagged := kvPairs[namer(item)]; tagged.value != "" {
			item.Value = tagged.value
			tagged.inUse = true
		}
	}
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
		// TODO: Use proper string keys for log levels instead of stringified numbers
		LogLevel: Item{InFile: "log_level", InArgs: "log-level", InEnv: "LOG_LEVEL", Value: strconv.Itoa(LOG_INFO)},
	}
}

func (c *Opts) AcceptedItems() []*Item {
	return []*Item{
		&c.ConfFile,
		&c.Socket,
		&c.Protocol,
		&c.Backend,
		&c.LogLevel,
	}
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

func FromFile(configFile string) rawConf {
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

func FromCommandLineParams(params []string) (rawConf, []string) {
	result := make(map[string]taggedString)
	var unused []string
	for i := 0; i < len(params); i++ {
		param := params[i]
		if strings.HasPrefix(param, CLI_VAR_PREFIX) {
			var key, value string
			if strings.Contains(param, "=") {
				pair := strings.Split(param, "=")
				key, value = pair[0], pair[1]
			} else {
				key = param
				if i+1 == len(params) || strings.HasPrefix(params[i+1], CLI_VAR_PREFIX) {
					// TODO: Handle error
				}
				i++
				value = params[i]
			}
			result[key] = taggedString{false, value}
		}
	}
	return result, unused
}

func FromEnvironment() rawConf {
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
