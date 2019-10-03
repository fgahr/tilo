package config

import (
	"testing"
)

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

func TestBackendSetCorrectlyArgs(t *testing.T) {
	RegisterBackend(newTestBackendConfig("test1"))

	args := []string{"--backend=test1"}
	defer func() {
		if r := recover(); r != nil {
			t.Error("Failed to recognize 'test1' backend:", r)
		}
	}()
	GetConfig(args, nil)
}

func TestBackendSetCorrectlyEnv(t *testing.T) {
	RegisterBackend(newTestBackendConfig("test2"))
	env := []string{ENV_VAR_PREFIX + "BACKEND=test2"}
	defer func() {
		if r := recover(); r != nil {
			t.Error("Failed to recognize 'test1' backend:", r)
		}
	}()
	GetConfig(nil, env)
}
