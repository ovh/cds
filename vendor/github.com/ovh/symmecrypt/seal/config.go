package seal

import (
	"sync"

	"github.com/ovh/configstore"
)

var (
	knownSealedConfigs    []string
	knownSealedConfigsMut sync.Mutex
)

/*
** SEALED CONFIG MANAGEMENT
** These functions are optional helpers to let you seal some of your configstore items easily.
 */

// SealedConfigFilter helps managing configstore items encrypted with the seal.
// By using this filter like you would configstore's filter, you can directly manipulate
// sealed config items. They will be transformed from their encrypted blob form back to their plain form automatically.
// Also, KnownSealedConfigs() will be populated with the key name.
type SealedConfigFilter struct {
	filter configstore.ItemFilter
}

// ConfigFilter returns a new SealedConfigFilter.
func ConfigFilter() *SealedConfigFilter {
	return &SealedConfigFilter{}
}

// Slice filters the list items, keeping only those matching key.
// If the global singleton is configured, it appends -sealed to the lookup key,
// and decrypts it before passing it to the rest of the chain.
func (s *SealedConfigFilter) Slice(key string) *configstore.ItemFilter {
	declareSealedConfig(key)
	return s.filter.Slice(
		key,
		func(s string) string {
			if Exists() {
				return SealedConfigName(s)
			}
			return s
		},
	).Transform(UnsealConfig)
}

// SealedConfigName returns the suffixed name to use for a configstore (sealed) item.
func SealedConfigName(name string) string {
	return name + "-sealed"
}

// DeclareSealedConfig adds the config item name to a list of known sealed config items.
// Useful when regenerating seal shards and needing to automatically re-seal existing config items.
func declareSealedConfig(name string) {
	knownSealedConfigsMut.Lock()
	knownSealedConfigs = append(knownSealedConfigs, name)
	knownSealedConfigsMut.Unlock()
}

// KnownSealedConfigs returns a list of known sealed config items.
// Useful when regenerating seal shards and needing to automatically re-seal existing config items.
func KnownSealedConfigs() []string {
	knownSealedConfigsMut.Lock()
	defer knownSealedConfigsMut.Unlock()
	return knownSealedConfigs
}

// UnsealConfig can be used with configstore.ItemList.Transform() to unseal configstore items.
func UnsealConfig(s *configstore.Item) (string, error) {
	val, err := s.Value()
	if err != nil {
		return "", err
	}
	if !Exists() {
		return val, err
	}
	b, err := global.Decrypt(val, []byte(s.Key()))
	return string(b), err
}

// SealConfigWith seals the configstore item with a given seal, and returns the sealed value.
func SealConfigWith(s *configstore.Item, se *Seal) (string, error) {
	val, err := s.Value()
	if err != nil {
		return "", err
	}
	b, err := se.Encrypt([]byte(val), []byte(SealedConfigName(s.Key())))
	return string(b), err
}
