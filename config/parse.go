package config

type KeyAndValue struct {
	Key   string
	Value string
}

type BaseConfig struct {
	Backend    string
	Protocol   string
	backParser BackendConfigParser
	// TODO: ProtocolConfigParser
}

type BackendConf struct {
	// TODO
}

type BackendConfigParser interface {
	// The name of the corresponding backend.
	BackendName() string
	// Apply the given arguments, return all unused
	ApplyValues(knv []KeyAndValue) []KeyAndValue
	// Construct the resulting configuration.
	EmitConfig() BackendConf
}

var backendParsers = make(map[string]BackendConfigParser)

func RegisterBackendParser(bcp BackendConfigParser) {
	if backendParsers[bcp.BackendName()] != nil {
		panic("Double registration of backend with name " + bcp.BackendName())
	}
	backendParsers[bcp.BackendName()] = bcp
}

func FromFile(configFile string) []KeyAndValue {
	// TODO
	return nil
}

func FromCommandLineParams(params []string) []KeyAndValue {
	// TODO
	return nil
}

func FromEnvironment() []KeyAndValue {
	// TODO
	return nil
}
