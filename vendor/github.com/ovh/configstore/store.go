package configstore

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"
)

type Store struct {
	providers             map[string]Provider
	pMut                  sync.Mutex
	allowProviderOverride bool

	watchers      []chan struct{}
	watchersMut   sync.Mutex
	watchersNotif bool
}

func NewStore() *Store {
	return &Store{providers: map[string]Provider{}, watchersNotif: true}
}

/*
** PROVIDERS
 */

// InitFromEnvironment initializes configuration providers via their name and an optional argument.
// Suitable provider factories should have been registered via RegisterProviderFactory for this to work.
// Built-in providers (File, FileList, FileTree, ...) are registered by default.
//
// Valid example:
// CONFIGURATION_FROM=file:/etc/myfile.conf,file:/etc/myfile2.conf,filelist:/home/foobar/configs
func (s *Store) InitFromEnvironment() {

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
			errorProvider(s, fmt.Sprintf("%s:%s", name, arg), errors.New("failed to instantiate provider factory"))
		} else {
			f(s, arg)
		}
	}
}

const (
	ProviderConflictErrorLabel = "provider-conflict-error"
)

// RegisterProvider registers a provider
func (s *Store) RegisterProvider(name string, f Provider) {
	switch name {
	case ProviderConflictErrorLabel:
		return
	}
	s.pMut.Lock()
	defer s.pMut.Unlock()
	_, ok := s.providers[name]
	if ok && !s.allowProviderOverride {
		s.providers[ProviderConflictErrorLabel] = newErrorProvider(fmt.Errorf("configstore: conflict on configuration provider: %s", name))
		return
	}
	s.providers[name] = f
}

// AllowProviderOverride allows multiple calls to RegisterProvider() with the same provider name.
// This is useful for controlled test cases, but is not recommended in the context of a real
// application.
func (s *Store) AllowProviderOverride() {
	fmt.Fprintln(os.Stderr, "configstore: ATTENTION: PROVIDER OVERRIDE ALLOWED/ENABLED")
	s.pMut.Lock()
	defer s.pMut.Unlock()
	s.allowProviderOverride = true
}

// ErrorProvider registers a configstore provider which always returns an error.
func (s *Store) ErrorProvider(name string, err error) {
	errorProvider(s, name, err)
}

// File registers a configstore provider which reads from the file given in parameter (static content).
func (s *Store) File(filename string) {
	fileProvider(s, filename)
}

// FileRefresh registers a configstore provider which readfs from the file given in parameter (provider watches file stat for auto refresh, watchers get notified).
func (s *Store) FileRefresh(filename string) {
	fileRefreshProvider(s, filename)
}

// FileCustom registers a configstore provider which reads from the file given in parameter, and loads the content using the given unmarshal function
func (s *Store) FileCustom(filename string, fn func([]byte) ([]Item, error)) {
	fileCustomProvider(s, filename, fn)
}

// FileCustomRefresh registers a configstore provider which reads from the file given in parameter, and loads the content using the given unmarshal function; and watches file stat for auto refresh
func (s *Store) FileCustomRefresh(filename string, fn func([]byte) ([]Item, error)) {
	fileCustomRefreshProvider(s, filename, fn)
}

// FileTree registers a configstore provider which reads from the files contained in the directory given in parameter.
// A limited hierarchy is supported: files can either be top level (in which case the file name will be used as the item key),
// or nested in a single sub-directory (in which case the sub-directory name will be used as item key for all the files contained in it).
// The content of the files should be the plain data, with no envelope.
// Capitalization can be used to indicate item priority for sub-directories containing multiple items which should be differentiated.
// Capitalized = higher priority.
func (s *Store) FileTree(dirname string) {
	fileTreeProvider(s, dirname)
}

// FileList registers a configstore provider which reads from the files contained in the directory given in parameter.
// The content of the files should be JSON/YAML similar to the File provider.
func (s *Store) FileList(dirname string) {
	fileListProvider(s, dirname)
}

// InMemory registers an InMemoryProvider with a given arbitrary name and returns it.
// You can append any number of items to it, see Add().
func (s *Store) InMemory(name string) *InMemoryProvider {
	return inMemoryProvider(s, name)
}

// Env registers a provider reading from the environment.
// Only variables beginning with "PREFIX_" will be considered.
// Trimmed variable names are used as keys. Keys are not case-sensitive.
// Underscores (_) in variable names are considered equivalent to dashes (-).
func (s *Store) Env(prefix string) {
	envProvider(s, prefix)
}

/*
** WATCH / NOTIFY
 */

// Watch returns a channel which you can range over.
// You will get unblocked every time a provider notifies of a configuration change.
func (s *Store) Watch() chan struct{} {
	// buffer size == 1, notifications will never use a blocking write
	newCh := make(chan struct{}, 1)
	s.watchersMut.Lock()
	s.watchers = append(s.watchers, newCh)
	s.watchersMut.Unlock()
	return newCh
}

// NotifyWatchers is used by providers to notify of configuration changes.
// It unblocks all the watchers which are ranging over a watch channel.
func (s *Store) NotifyWatchers() {
	s.watchersMut.Lock()
	if !s.watchersNotif {
		s.watchersMut.Unlock()
		return
	}
	for _, ch := range s.watchers {
		select {
		case ch <- struct{}{}:
		default:
		}
	}
	s.watchersMut.Unlock()
}

// NotifyMute prevents configstore from notifying watchers on configuration
// changes, until MotifyUnmute() is called.
func (s *Store) NotifyMute() {
	s.watchersMut.Lock()
	s.watchersNotif = false
	s.watchersMut.Unlock()
}

// NotifyUnmute allows configstore to resume notifications to watchers
// on configuration changes. This will trigger a notification to catch up any change
// done during the time spent mute.
func (s *Store) NotifyUnmute() {
	var alreadyUnmute bool
	s.watchersMut.Lock()
	alreadyUnmute = s.watchersNotif
	s.watchersNotif = true
	s.watchersMut.Unlock()
	if !alreadyUnmute {
		go s.NotifyWatchers()
	}
}

// NotifyIsMuted reports whether notifications are currently muted.
func (s *Store) NotifyIsMuted() bool {
	s.watchersMut.Lock()
	defer s.watchersMut.Unlock()
	return !s.watchersNotif
}

/*
** GETTERS
 */

// GetItemList retrieves the full item list, merging the results from all providers.
// It does NOT cache, it's the responsability of the providers to keep an in-ram representation if desired.
func (s *Store) GetItemList() (*ItemList, error) {

	s.pMut.Lock()
	defer s.pMut.Unlock()

	ret := &ItemList{}

	for n, p := range s.providers {
		l, err := p()
		if err != nil {
			return nil, ErrProvider(fmt.Sprintf("configstore: provider '%s': %s", n, err))
		}
		ret.Items = append(ret.Items, l.Items...)
	}
	return ret.index(), nil
}

// GetItem retrieves the full item list, merging the results from all providers, then returns a single item by key.
// If 0 or >=2 items are present with that key, it will return an error.
func (s *Store) GetItem(key string) (Item, error) {
	items, err := s.GetItemList()
	if err != nil {
		return Item{}, err
	}
	return items.GetItem(key)
}

// GetItemValue fetches the full item list, merging the results from all providers, then returns a single item's value by key.
func (s *Store) GetItemValue(key string) (string, error) {
	i, err := s.GetItem(key)
	if err != nil {
		return "", err
	}
	return i.Value()
}

// GetItemValueBool fetches the full item list, merging the results from all providers, then returns a single item's value by key.
func (s *Store) GetItemValueBool(key string) (bool, error) {
	i, err := s.GetItem(key)
	if err != nil {
		return false, err
	}
	return i.ValueBool()
}

// GetItemValueFloat fetches the full item list, merging the results from all providers, then returns a single item's value by key.
func (s *Store) GetItemValueFloat(key string) (float64, error) {
	i, err := s.GetItem(key)
	if err != nil {
		return 0, err
	}
	return i.ValueFloat()
}

// GetItemValueInt fetches the full item list, merging the results from all providers, then returns a single item's value by key.
func (s *Store) GetItemValueInt(key string) (int64, error) {
	i, err := s.GetItem(key)
	if err != nil {
		return 0, err
	}
	return i.ValueInt()
}

// GetItemValueUint fetches the full item list, merging the results from all providers, then returns a single item's value by key.
func (s *Store) GetItemValueUint(key string) (uint64, error) {
	i, err := s.GetItem(key)
	if err != nil {
		return 0, err
	}
	return i.ValueUint()
}

// GetItemValueDuration fetches the full item list, merging the results from all providers, then returns a single item's value by key.
func (s *Store) GetItemValueDuration(key string) (time.Duration, error) {
	i, err := s.GetItem(key)
	if err != nil {
		return time.Duration(0), err
	}
	return i.ValueDuration()
}
