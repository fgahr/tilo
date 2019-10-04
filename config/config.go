// Package config handles everything related to runtime-configuration.
//
// Three configuration sources are supported. In order of ascending priority:
// configuration file, environment variables, command line arguments.
package config

import (
	"fmt"
	"github.com/pkg/errors"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"strings"
)

const (
	LOG_OFF   = "off"
	LOG_WARN  = "warn"
	LOG_INFO  = "info"
	LOG_DEBUG = "debug"
	LOG_TRACE = "trace"
)

func logLevel(description string) int {
	switch description {
	case LOG_OFF:
		return 0
	case LOG_WARN:
		return 1
	case LOG_INFO:
		return 2
	case LOG_DEBUG:
		return 3
	case LOG_TRACE:
		return 4
	default:
		return 1
	}
}

const (
	ENV_VAR_PREFIX = "__TILO_"
	CLI_VAR_PREFIX = "--"
)

type taggedString struct {
	inUse bool
	value string
}

type rawConf struct {
	values map[string]string
	inUse  map[string]bool
}

func makeRawConf() rawConf {
	values := make(map[string]string)
	inUse := make(map[string]bool)
	return rawConf{values: values, inUse: inUse}
}

// TODO: Add Description field for help messages?
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

func GetConfig(args []string, env []string) (*Opts, []string, error) {
	conf := defaultConfig()

	fromEnv := FromEnvironment(env)
	fromArgs, unused, err := FromCommandLineParams(args)
	if err != nil {
		return nil, args, errors.Wrap(err, "Failed to establish configuration")
	}

	// Determine whether we are dealing with an alternative config file location
	apply([]*Item{&conf.ConfFile}, fromEnv, nameInEnv)
	apply([]*Item{&conf.ConfFile}, fromArgs, nameInArgs)
	fromFile, err := FromFile(conf.ConfFile.Value)
	if err != nil {
		return nil, args, errors.Wrap(err, "Failed to establish configuration")
	}

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

	warnUnused(fromFile, fromEnv, fromArgs)

	return conf, unused, nil
}

func apply(items []*Item, conf rawConf, namer func(*Item) string) {
	for _, item := range items {
		key := namer(item)
		if value := conf.values[key]; value != "" {
			item.Value = value
			conf.inUse[key] = true
		}
	}
}

func warnUnused(confs ...rawConf) {
	for _, conf := range confs {
		for key, value := range conf.values {
			if !conf.inUse[key] {
				warn("Unused parameter:", key, "with value:", value)
			}
		}
	}
}

func warn(message ...interface{}) {
	fmt.Fprintln(os.Stderr, message...)
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
		LogLevel: Item{InFile: "log_level", InArgs: "log-level", InEnv: "LOG_LEVEL", Value: LOG_INFO},
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
	return c.logLevel() > logLevel(LOG_OFF)
}

func (c *Opts) ShouldLogWarnings() bool {
	return c.logLevel() >= logLevel(LOG_WARN)
}

func (c *Opts) ShouldLogInfo() bool {
	return c.logLevel() >= logLevel(LOG_INFO)
}

func (c *Opts) ShouldLogDebug() bool {
	return c.logLevel() >= logLevel(LOG_DEBUG)
}

func (c *Opts) ShouldLogTrace() bool {
	return c.logLevel() >= logLevel(LOG_TRACE)
}

func (c *Opts) logLevel() int {
	return logLevel(c.LogLevel.Value)
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

// Take a list of environment-compatible key=value pairs and add tilo-options.
func (c *Opts) MergeIntoEnv(env []string) []string {
	var result []string
	for _, keyValuePair := range env {
		// Skip tilo-related config. We assume we already have the definitive
		// configuration and append it afterwards
		if !strings.HasPrefix(keyValuePair, ENV_VAR_PREFIX) {
			result = append(result, keyValuePair)
		}
	}
	return append(result, c.AsEnvKeyValue()...)
}

// Read configuration from a config file.
func FromFile(configFile string) (rawConf, error) {
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		return rawConf{}, nil
	}
	result := makeRawConf()
	data, _ := ioutil.ReadFile(configFile)
	asString := string(data)
	lines := strings.Split(asString, "\n")
	for i, fullLine := range lines {
		lnum := i + 1
		line := strings.Split(fullLine, "#")[0]
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}

		rawKey, rawValue := splitKeyValue(trimmed)
		key := strings.TrimSpace(rawKey)
		value := strings.TrimSpace(rawValue)
		if key == "" || value == "" {
			return result, errors.Errorf("Error in file %s, line %d: %s", configFile, lnum, line)
		}
		result.values[key] = value
		result.inUse[key] = false
	}
	return result, nil
}

// Read a configuration from command line parameters.
func FromCommandLineParams(args []string) (rawConf, []string, error) {
	result := makeRawConf()
	var unused []string
	for i := 0; i < len(args); i++ {
		param := args[i]
		if strings.HasPrefix(param, CLI_VAR_PREFIX) {
			var rawKey, value string
			// Value in the same arg?
			if strings.Contains(param, "=") {
				rawKey, value = splitKeyValue(param)
				if value == "" {
					return result, args, errors.New("No value for parameter: " + param)
				}
			} else { // Value in the next arg
				rawKey = param
				if i+1 == len(args) {
					return result, args, errors.New("No value for parameter: " + param)
				} else if strings.HasPrefix(args[i+1], CLI_VAR_PREFIX) {
					return result, args, errors.New("Not a valid value for parameter " + param + ": " + args[i+1])
				}
				i++
				value = args[i]
			}
			key := strings.Replace(rawKey, CLI_VAR_PREFIX, "", 1)
			result.values[key] = value
			result.inUse[key] = false
		} else {
			unused = append(unused, param)
		}
	}
	return result, unused, nil
}

// Read a configuration from environment-compatible key=value pairs.
func FromEnvironment(env []string) rawConf {
	result := makeRawConf()
	for _, keyValuePair := range env {
		if strings.HasPrefix(keyValuePair, ENV_VAR_PREFIX) {
			rawKey, value := splitKeyValue(keyValuePair)
			if rawKey == "" {
				// No need to save empty values
				continue
			}
			key := strings.Replace(rawKey, ENV_VAR_PREFIX, "", 1)
			result.values[key] = value
			result.inUse[key] = false
		}
	}
	return result
}

func splitKeyValue(str string) (string, string) {
	if !strings.Contains(str, "=") {
		return "", ""
	}

	pair := strings.Split(str, "=")
	return pair[0], pair[1]
}
