package configstore

import (
	"fmt"
	"sync"
)

const (
	// ConfigEnvVar defines the environment variable used to set up the configuration providers via InitFromEnvironment
	ConfigEnvVar = "CONFIGURATION_FROM"
)

var (
	providerFactories = map[string]ProviderFactory{}
	pFactMut          sync.Mutex
)

func init() {
	RegisterProviderFactory("file", fileProvider)
	RegisterProviderFactory("filelist", fileListProvider)
	RegisterProviderFactory("filetree", fileTreeProvider)
	RegisterProviderFactory("env", envProvider)
}

// A Provider retrieves config items and makes them available to the configstore,
// Their implementations can vary wildly (HTTP API, file, env, hardcoded test, ...)
// and their results will get merged by the configstore library.
// It's the responsability of the application using configstore to register suitable providers.
type Provider func() (ItemList, error)

// A ProviderFactory is a function that instantiates a Provider and registers it
// to a store instance.
type ProviderFactory func(*Store, string)

// RegisterProviderFactory registers a factory function so that InitFromEnvironment can properly
// instantiate configuration providers via name + argument.
func RegisterProviderFactory(name string, f ProviderFactory) {
	pFactMut.Lock()
	defer pFactMut.Unlock()
	_, ok := providerFactories[name]
	if ok {
		panic(fmt.Sprintf("conflict on configuration provider factory: %s", name))
	}
	providerFactories[name] = f
}
