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
	"strings"

	"github.com/pkg/errors"
)

const (
	// LogOff is the config value to set to disable logging
	LogOff = "off"
	// LogWarn is the config value to set to log warnings only
	LogWarn = "warn"
	// LogInfo is the config value to set for logging to include most infos
	LogInfo = "info"
	// LogDebug is the config value to set for logging to include debug data
	LogDebug = "debug"
	// LogTrace is the config value to set for logging to include all data
	LogTrace = "trace"
)

func logLevel(description string) int {
	switch description {
	case LogOff:
		return 0
	case LogWarn:
		return 1
	case LogInfo:
		return 2
	case LogDebug:
		return 3
	case LogTrace:
		return 4
	default:
		return 1
	}
}

const (
	// EnvVarPrefix is the common prefix for environment variable names
	EnvVarPrefix = "__TILO_"
	// CliVarPrefix is the common prefix for command line variable names
	CliVarPrefix = "--"
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

// Opts is a bundle of all configuration options
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
	// Output determines the type of output printed to the user
	Output Item
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
		LogLevel: Item{InFile: "log_level", InArgs: "log-level", InEnv: "LOG_LEVEL", Value: LogInfo},
		Output:   Item{InFile: "output", InArgs: "output", InEnv: "OUTPUT", Value: "tabular"},
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
	return c.logLevel() > logLevel(LogOff)
}

func (c *Opts) ShouldLogWarnings() bool {
	return c.logLevel() >= logLevel(LogWarn)
}

func (c *Opts) ShouldLogInfo() bool {
	return c.logLevel() >= logLevel(LogInfo)
}

func (c *Opts) ShouldLogDebug() bool {
	return c.logLevel() >= logLevel(LogDebug)
}

func (c *Opts) ShouldLogTrace() bool {
	return c.logLevel() >= logLevel(LogTrace)
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
		result[i] = fmt.Sprintf("%s%s=%v", EnvVarPrefix, name, v.Field(i))

	}
	return result
}

// Take a list of environment-compatible key=value pairs and add tilo-options.
func (c *Opts) MergeIntoEnv(env []string) []string {
	var result []string
	for _, keyValuePair := range env {
		// Skip tilo-related config. We assume we already have the definitive
		// configuration and append it afterwards
		if !strings.HasPrefix(keyValuePair, EnvVarPrefix) {
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
			return result, errors.Errorf("Error in file %s, line %d: %s", configFile, lnum, fullLine)
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
		if strings.HasPrefix(param, CliVarPrefix) {
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
				} else if strings.HasPrefix(args[i+1], CliVarPrefix) {
					return result, args, errors.New("Not a valid value for parameter " + param + ": " + args[i+1])
				}
				i++
				value = args[i]
			}
			key := strings.Replace(rawKey, CliVarPrefix, "", 1)
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
		if strings.HasPrefix(keyValuePair, EnvVarPrefix) {
			rawKey, value := splitKeyValue(keyValuePair)
			if rawKey == "" {
				// No need to save empty values
				continue
			}
			key := strings.Replace(rawKey, EnvVarPrefix, "", 1)
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
