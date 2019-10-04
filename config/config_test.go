package config

import (
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

	env := []string{ENV_VAR_PREFIX + "BACKEND=" + backend}
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
	args := []string{CLI_VAR_PREFIX + "backend=" + backendName, CLI_VAR_PREFIX + "foo", newFoo}
	env := []string{ENV_VAR_PREFIX + "BAR=" + newBar}
	_, _, err := GetConfig(args, env)
	if err != nil {
		t.Error(err)
	}
	if backendConf.foo.Value != newFoo {
		t.Errorf("foo not set to '%s', instead: %s", newFoo, backendConf.foo.Value)
	}
	if backendConf.bar.Value != newBar {
		t.Errorf("foo not set to '%s', instead: %s", newBar, backendConf.bar.Value)
	}
}
