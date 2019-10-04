package config

import (
	"io/ioutil"
	"os"
	"testing"
)

func unsetBackendConfig(name string) {
	backendConfigs[name] = nil
}

type testBackendConfig struct {
	name string
	foo  Item
	bar  Item
}

func newTestBackendConfig(name string) *testBackendConfig {
	foo := Item{InFile: "foo", InEnv: "FOO", InArgs: "foo", Value: "foo"}
	bar := Item{InFile: "bar", InEnv: "BAR", InArgs: "bar", Value: "bar"}
	return &testBackendConfig{name: name, foo: foo, bar: bar}
}

func (c *testBackendConfig) BackendName() string {
	return c.name
}

func (c *testBackendConfig) AcceptedItems() []*Item {
	return []*Item{&c.foo, &c.bar}
}

func cliVar(name string) string {
	return CLI_VAR_PREFIX + name
}

func cliVal(name, value string) string {
	return CLI_VAR_PREFIX + name + "=" + value
}

func envVar(name string) string {
	return ENV_VAR_PREFIX + name
}

func envVal(name, value string) string {
	return ENV_VAR_PREFIX + name + "=" + value
}

func expect(t *testing.T, varName string, value string, expected string) {
	if value != expected {
		t.Errorf("%s not set to '%s', instead: %s", varName, expected, value)
	}
}

func TestBackendSetFromArgs(t *testing.T) {
	backend := "backendFromArgs"
	RegisterBackend(newTestBackendConfig(backend))
	defer unsetBackendConfig(backend)

	args := []string{"--backend=" + backend}
	defer func() {
		if r := recover(); r != nil {
			t.Error("Failed to recognize backend:", backend, r)
		}
	}()
	GetConfig(args, nil)
}

func TestBackendSetFromEnv(t *testing.T) {
	backend := "backendFromEnv"
	RegisterBackend(newTestBackendConfig(backend))
	defer unsetBackendConfig(backend)

	env := []string{envVal("BACKEND", backend)}
	defer func() {
		if r := recover(); r != nil {
			t.Error("Failed to recognize backend:", backend, r)
		}
	}()
	GetConfig(nil, env)
}

func TestBackendParametersFromArgs(t *testing.T) {
	backendName := "backendParametersFromArgs"
	backendConf := newTestBackendConfig(backendName)
	RegisterBackend(backendConf)
	defer unsetBackendConfig(backendName)

	newFoo := "new-foo"
	newBar := "new-bar"
	args := []string{cliVal("backend", backendName), cliVar("foo"), newFoo}
	env := []string{envVal("BAR", newBar)}
	_, _, err := GetConfig(args, env)
	if err != nil {
		t.Error(err)
	}

	expect(t, "foo", backendConf.foo.Value, newFoo)
	expect(t, "bar", backendConf.bar.Value, newBar)
}

func TestParametersFromFile(t *testing.T) {
	backendName := "backendParametersFromFile"
	backendConf := newTestBackendConfig(backendName)
	RegisterBackend(backendConf)
	defer unsetBackendConfig(backendName)

	file, err := ioutil.TempFile(os.TempDir(), "tilo_config")
	if err != nil {
		t.Error(err)
	}
	defer os.Remove(file.Name())

	if _, err = file.WriteString("foo=fooValue\n#bar=notBar\nlog_level=trace"); err != nil {
		t.Error(err)
	}

	args := []string{cliVal("conf-file", file.Name()), cliVar("backend"), backendName}
	conf, _, err := GetConfig(args, nil)
	if err != nil {
		t.Fatal(err)
	}

	expect(t, "config file", conf.ConfFile.Value, file.Name())
	expect(t, "log level", conf.LogLevel.Value, "trace")
	expect(t, "foo", backendConf.foo.Value, "fooValue")
	expect(t, "bar", backendConf.bar.Value, "bar")
}
