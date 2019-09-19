package configstore

import (
	"time"
)

var (
	DefaultStore = NewStore()
)

/*
** PROVIDERS
 */

// InitFromEnvironment initializes configuration providers via their name and an optional argument.
// Suitable provider factories should have been registered via RegisterProviderFactory for this to work.
// Built-in providers (File, FileList, FileTree, ...) are registered by default.
//
// Valid example:
// CONFIGURATION_FROM=file:/etc/myfile.conf,file:/etc/myfile2.conf,filelist:/home/foobar/configs
func InitFromEnvironment() {
	DefaultStore.InitFromEnvironment()
}

// RegisterProvider registers a provider
func RegisterProvider(name string, f Provider) {
	DefaultStore.RegisterProvider(name, f)
}

// AllowProviderOverride allows multiple calls to RegisterProvider() with the same provider name.
// This is useful for controlled test cases, but is not recommended in the context of a real
// application.
func AllowProviderOverride() {
	DefaultStore.AllowProviderOverride()
}

// ErrorProvider registers a configstore provider which always returns an error.
func ErrorProvider(name string, err error) {
	DefaultStore.ErrorProvider(name, err)
}

// File registers a configstore provider which reads from the file given in parameter (static content).
func File(filename string) {
	DefaultStore.File(filename)
}

// FileRefresh registers a configstore provider which readfs from the file given in parameter (provider watches file stat for auto refresh, watchers get notified).
func FileRefresh(filename string) {
	DefaultStore.FileRefresh(filename)
}

// FileCustom registers a configstore provider which reads from the file given in parameter, and loads the content using the given unmarshal function
func FileCustom(filename string, fn func([]byte) ([]Item, error)) {
	DefaultStore.FileCustom(filename, fn)
}

// FileCustomRefresh registers a configstore provider which reads from the file given in parameter, and loads the content using the given unmarshal function; and watches file stat for auto refresh
func FileCustomRefresh(filename string, fn func([]byte) ([]Item, error)) {
	DefaultStore.FileCustomRefresh(filename, fn)
}

// FileTree registers a configstore provider which reads from the files contained in the directory given in parameter.
// A limited hierarchy is supported: files can either be top level (in which case the file name will be used as the item key),
// or nested in a single sub-directory (in which case the sub-directory name will be used as item key for all the files contained in it).
// The content of the files should be the plain data, with no envelope.
// Capitalization can be used to indicate item priority for sub-directories containing multiple items which should be differentiated.
// Capitalized = higher priority.
func FileTree(dirname string) {
	DefaultStore.FileTree(dirname)
}

// FileList registers a configstore provider which reads from the files contained in the directory given in parameter.
// The content of the files should be JSON/YAML similar to the File provider.
func FileList(dirname string) {
	DefaultStore.FileList(dirname)
}

// InMemory registers an InMemoryProvider with a given arbitrary name and returns it.
// You can append any number of items to it, see Add().
func InMemory(name string) *InMemoryProvider {
	return DefaultStore.InMemory(name)
}

// Env registers a provider reading from the environment.
// Only variables beginning with "PREFIX_" will be considered.
// Trimmed variable names are used as keys. Keys are not case-sensitive.
// Underscores (_) in variable names are considered equivalent to dashes (-).
func Env(prefix string) {
	DefaultStore.Env(prefix)
}

/*
** WATCH / NOTIFY
 */

// Watch returns a channel which you can range over.
// You will get unblocked every time a provider notifies of a configuration change.
func Watch() chan struct{} {
	return DefaultStore.Watch()
}

// NotifyWatchers is used by providers to notify of configuration changes.
// It unblocks all the watchers which are ranging over a watch channel.
func NotifyWatchers() {
	DefaultStore.NotifyWatchers()
}

// NotifyMute prevents configstore from notifying watchers on configuration
// changes, until MotifyUnmute() is called.
func NotifyMute() {
	DefaultStore.NotifyMute()
}

// NotifyUnmute allows configstore to resume notifications to watchers
// on configuration changes. This will trigger a notification to catch up any change
// done during the time spent mute.
func NotifyUnmute() {
	DefaultStore.NotifyUnmute()
}

// NotifyIsMuted reports whether notifications are currently muted.
func NotifyIsMuted() bool {
	return DefaultStore.NotifyIsMuted()
}

/*
** GETTERS
 */

// GetItemList retrieves the full item list, merging the results from all providers.
// It does NOT cache, it's the responsability of the providers to keep an in-ram representation if desired.
func GetItemList() (*ItemList, error) {
	return DefaultStore.GetItemList()
}

// GetItem retrieves the full item list, merging the results from all providers, then returns a single item by key.
// If 0 or >=2 items are present with that key, it will return an error.
func GetItem(key string) (Item, error) {
	return DefaultStore.GetItem(key)
}

// GetItemValue fetches the full item list, merging the results from all providers, then returns a single item's value by key.
func GetItemValue(key string) (string, error) {
	return DefaultStore.GetItemValue(key)
}

// GetItemValueBool fetches the full item list, merging the results from all providers, then returns a single item's value by key.
func GetItemValueBool(key string) (bool, error) {
	return DefaultStore.GetItemValueBool(key)
}

// GetItemValueFloat fetches the full item list, merging the results from all providers, then returns a single item's value by key.
func GetItemValueFloat(key string) (float64, error) {
	return DefaultStore.GetItemValueFloat(key)
}

// GetItemValueInt fetches the full item list, merging the results from all providers, then returns a single item's value by key.
func GetItemValueInt(key string) (int64, error) {
	return DefaultStore.GetItemValueInt(key)
}

// GetItemValueUint fetches the full item list, merging the results from all providers, then returns a single item's value by key.
func GetItemValueUint(key string) (uint64, error) {
	return DefaultStore.GetItemValueUint(key)
}

// GetItemValueDuration fetches the full item list, merging the results from all providers, then returns a single item's value by key.
func GetItemValueDuration(key string) (time.Duration, error) {
	return DefaultStore.GetItemValueDuration(key)
}
