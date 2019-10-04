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
	return nil
}

func TestBackendSetFromArgs(t *testing.T) {
	backend := "test1"
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
	backend := "test2"
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
