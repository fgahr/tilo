package config

import (
	"testing"
)

func init() {
	RegisterBackend(newTestBackendConfig())
}

type testBackendConfig struct {
	foo Item
	bar Item
}

func newTestBackendConfig() *testBackendConfig {
	foo := Item{InFile: "foo", InEnv: "FOO", InArgs: "foo", Value: "foo"}
	bar := Item{InFile: "bar", InEnv: "BAR", InArgs: "bar", Value: "bar"}
	return &testBackendConfig{foo: foo, bar: bar}
}

func (c *testBackendConfig) BackendName() string {
	return "test"
}

func (c *testBackendConfig) AcceptedItems() []*Item {
	return nil
}

func TestBackendSetCorrectly(t *testing.T) {
	args := []string{"--backend=test"}
	defer func() {
		if r := recover(); r != nil {
			t.Error("Failed to recognize 'test' backend:", r)
		}
	}()
	GetConfig(args)
}
