package configstore

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
)

const (
	// ConfigEnvVar defines the environment variable used to set up the configuration providers via InitFromEnvironment
	ConfigEnvVar = "CONFIGURATION_FROM"
)

var (
	providers             = map[string]Provider{}
	pMut                  sync.Mutex
	allowProviderOverride bool

	providerFactories = map[string]func(string){}
	pFactMut          sync.Mutex
)

func init() {
	RegisterProviderFactory("file", File)
	RegisterProviderFactory("filelist", FileList)
	RegisterProviderFactory("filetree", FileTree)
}

// A Provider retrieves config items and makes them available to the configstore,
// Their implementations can vary wildly (HTTP API, file, env, hardcoded test, ...)
// and their results will get merged by the configstore library.
// It's the responsability of the application using configstore to register suitable providers.
type Provider func() (ItemList, error)

// RegisterProvider registers a provider
func RegisterProvider(name string, f Provider) {
	pMut.Lock()
	defer pMut.Unlock()
	_, ok := providers[name]
	if ok && !allowProviderOverride {
		panic(fmt.Sprintf("conflict on configuration provider: %s", name))
	}
	providers[name] = f
}

// AllowProviderOverride allows multiple calls to RegisterProvider() with the same provider name.
// This is useful for controlled test cases, but is not recommended in the context of a real
// application.
func AllowProviderOverride() {
	fmt.Fprintln(os.Stderr, "configstore: ATTENTION: PROVIDER OVERRIDE ALLOWED/ENABLED")
	pMut.Lock()
	defer pMut.Unlock()
	allowProviderOverride = true
}

// RegisterProviderFactory registers a factory function so that InitFromEnvironment can properly
// instantiate configuration providers via name + argument.
func RegisterProviderFactory(name string, f func(string)) {
	pMut.Lock()
	defer pMut.Unlock()
	_, ok := providerFactories[name]
	if ok {
		panic(fmt.Sprintf("conflict on configuration provider factory: %s", name))
	}
	providerFactories[name] = f
}

// InitFromEnvironment initializes configuration providers via their name and an optional argument.
// Suitable provider factories should have been registered via RegisterProviderFactory for this to work.
// Built-in providers (File, FileList, FileTree, ...) are registered by default.
//
// Valid example:
// CONFIGURATION_FROM=file:/etc/myfile.conf,file:/etc/myfile2.conf,filelist:/home/foobar/configs
func InitFromEnvironment() {

	pFactMut.Lock()
	defer pFactMut.Unlock()

	cfg := os.Getenv(ConfigEnvVar)
	if cfg == "" {
		return
	}
	cfgList := strings.Split(cfg, ",")
	for _, c := range cfgList {
		parts := strings.SplitN(c, ":", 2)
		name := c
		arg := ""
		if len(parts) > 1 {
			name = parts[0]
			arg = parts[1]
		}
		name = strings.TrimSpace(name)
		arg = strings.TrimSpace(arg)
		f := providerFactories[name]
		if f == nil {
			ErrorProvider(fmt.Sprintf("%s:%s", name, arg), errors.New("failed to instantiate provider factory"))
		} else {
			f(arg)
		}
	}
}

var (
	watchers    []chan struct{}
	watchersMut sync.Mutex
)

// Watch returns a channel which you can range over.
// You will get unblocked every time a provider notifies of a configuration change.
func Watch() chan struct{} {
	// buffer size == 1, notifications will never use a blocking write
	newCh := make(chan struct{}, 1)
	watchersMut.Lock()
	watchers = append(watchers, newCh)
	watchersMut.Unlock()
	return newCh
}

// NotifyWatchers is used by providers to notify of configuration changes.
// It unblocks all the watchers which are ranging over a watch channel.
func NotifyWatchers() {
	watchersMut.Lock()
	for _, ch := range watchers {
		select {
		case ch <- struct{}{}:
		default:
		}
	}
	watchersMut.Unlock()
}
