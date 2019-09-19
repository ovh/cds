package keyloader

import (
	"fmt"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/juju/errors"
	"github.com/ovh/configstore"
	"github.com/ovh/symmecrypt"
	"github.com/ovh/symmecrypt/seal"
	"github.com/sirupsen/logrus"

	// aes-gcm cipher
	_ "github.com/ovh/symmecrypt/ciphers/aesgcm"

	// chacha20-poly1305 cipher
	_ "github.com/ovh/symmecrypt/ciphers/chacha20poly1305"

	// xchacha20-poly1305 cipher
	"github.com/ovh/symmecrypt/ciphers/xchacha20poly1305"

	// aes-pmac-siv cipher
	_ "github.com/ovh/symmecrypt/ciphers/aespmacsiv"

	// hmac cipher
	_ "github.com/ovh/symmecrypt/ciphers/hmac"
)

const (
	// EncryptionKeyConfigName is the name of the config items representing encryption keys in configstore.
	EncryptionKeyConfigName = "encryption-key"

	// DefaultCipher is the cipher used by default if an empty ciper name is passed to GenerateKey().
	DefaultCipher = xchacha20poly1305.CipherName
)

var (
	// ConfigFilter is the configstore manipulation filter used to retrieve the encryption keys
	ConfigFilter = configstore.Filter().Slice(EncryptionKeyConfigName).Unmarshal(configFactory).Rekey(rekeyConfigByIdentifier).Reorder(reorderTimestamp)
)

// KeyConfig is the representation of an encryption key in the configuration.
// - Identifier is a free name to uniquely reference this key (and its revisions). It is used when loading the key.
// - Cipher controls which cipher is used (aes-gcm, ...)
// - Timestamp dictates priority between encryption keys, and is useful to identify new versions of a key
// - Sealed controls whether the key should be used as-is, or decrypted using symmecrypt/seal
//   See RegisterCipher() to register a factory. The cipher field should be the same as the factory name.
type KeyConfig struct {
	Identifier string `json:"identifier"`
	Cipher     string `json:"cipher"`
	Timestamp  int64  `json:"timestamp,omitempty"`
	Sealed     bool   `json:"sealed,omitempty"`
	Key        string `json:"key"`
}

// A watchKey is an implementation of a key that watches for configstore updates
// and hot reloads itself.
type watchKey struct {
	identifier string
	k          symmecrypt.Key
	mut        sync.RWMutex
}

// A sealedKey is an implementation of an encryption key that is encrypted using symmecrypt/seal.
type sealedKey struct {
	decryptedKey symmecrypt.Key
	decrypted    uint32
	waitCh       chan struct{}
}

/*
** CONFIG HELPERS
 */

// GenerateKey generates a new random key and returns its configuration object representation.
func GenerateKey(cipher string, identifier string, sealed bool, timestamp time.Time) (*KeyConfig, error) {

	if cipher == "" {
		cipher = DefaultCipher
	}

	k, err := symmecrypt.NewRandomKey(cipher)
	if err != nil {
		return nil, err
	}

	if timestamp.IsZero() {
		timestamp = time.Now()
	}

	str, err := k.String()
	if err != nil {
		return nil, err
	}
	ret := &KeyConfig{Key: str, Identifier: identifier, Sealed: false, Timestamp: timestamp.Unix(), Cipher: cipher}

	if sealed {
		return SealKey(ret, seal.Global())
	}

	return ret, nil
}

// SealKey returns a copy of the key configuration object, ensuring it is sealed.
func SealKey(k *KeyConfig, s *seal.Seal) (*KeyConfig, error) {
	if k.Sealed {
		return &KeyConfig{
			Key:        k.Key,
			Identifier: k.Identifier,
			Timestamp:  k.Timestamp,
			Cipher:     k.Cipher,
			Sealed:     k.Sealed,
		}, nil
	}

	enc, err := s.Encrypt([]byte(k.Key), []byte(k.Identifier), []byte("|"), []byte(strconv.FormatInt(k.Timestamp, 10)), []byte("|"), []byte(k.Cipher))
	if err != nil {
		return nil, err
	}

	return &KeyConfig{
		Key:        enc,
		Identifier: k.Identifier,
		Timestamp:  k.Timestamp,
		Cipher:     k.Cipher,
		Sealed:     true,
	}, nil
}

// UnsealKey returns a copy of the key configuration object, ensuring it is unsealed.
func UnsealKey(k *KeyConfig, s *seal.Seal) (*KeyConfig, error) {
	if !k.Sealed {
		return &KeyConfig{
			Key:        k.Key,
			Identifier: k.Identifier,
			Timestamp:  k.Timestamp,
			Cipher:     k.Cipher,
			Sealed:     k.Sealed,
		}, nil
	}

	dec, err := s.Decrypt(k.Key, []byte(k.Identifier), []byte("|"), []byte(strconv.FormatInt(k.Timestamp, 10)), []byte("|"), []byte(k.Cipher))
	if err != nil {
		return nil, err
	}

	return &KeyConfig{
		Key:        string(dec),
		Identifier: k.Identifier,
		Cipher:     k.Cipher,
		Timestamp:  k.Timestamp,
		Sealed:     false,
	}, nil
}

// ConfiguredKeys returns a list of all the encryption keys present in the default store in configstore
// ensuring they are unsealed.
func ConfiguredKeys() ([]*KeyConfig, error) {
	return ConfiguredKeysFromStore(configstore.DefaultStore)
}

// ConfiguredKeys returns a list of all the encryption keys present in a specific store instance
// ensuring they are unsealed.
func ConfiguredKeysFromStore(store *configstore.Store) ([]*KeyConfig, error) {

	ret := []*KeyConfig{}

	items, err := ConfigFilter.Store(store).GetItemList()
	if err != nil {
		return nil, err
	}

	for _, item := range items.Items {
		i, err := item.Unmarshaled()
		if err != nil {
			return nil, err
		}
		newK, err := UnsealKey(i.(*KeyConfig), seal.Global())
		if err != nil {
			return nil, err
		}
		ret = append(ret, newK)
	}

	return ret, nil
}

// Helper to manipulate the configuration encryption keys by identifier
func rekeyConfigByIdentifier(s *configstore.Item) string {
	i, err := s.Unmarshaled()
	if err == nil {
		return i.(*KeyConfig).Identifier
	}
	return ""
}

// Helper to sort the configuration encryption keys by timestamp
func reorderTimestamp(s *configstore.Item) int64 {
	i, err := s.Unmarshaled()
	if err == nil {
		ret := i.(*KeyConfig).Timestamp
		if i.(*KeyConfig).Sealed {
			ret++
		}
		return ret
	}
	return s.Priority()
}

func configFactory() interface{} {
	return &KeyConfig{}
}

/*
** CONSTRUCTORS
 */

// LoadKey instantiates a new encryption key for a given identifier from the default store in configstore.
//
// If several keys are found for the identifier, they are sorted by timestamp, and a composite key is returned.
// The most recent key will be used for encryption, and decryption will be done by any of them.
// There needs to be _only one_ key with the highest priority for the identifier.
//
// If the key configuration specifies it is sealed, the key returned will be wrapped by an unseal mechanism.
// When the symmecrypt/seal global singleton gets unsealed, the key will become usable instantly. It will return errors in the meantime.
//
// The key cipher name is expected to match a KeyFactory that got registered through RegisterCipher().
// Either use a built-in cipher, or make sure to register a proper factory for this cipher.
// This KeyFactory will be called, either directly or when the symmecrypt/seal global singleton gets unsealed, if applicable.
func LoadKey(identifier string) (symmecrypt.Key, error) {
	return LoadKeyFromStore(identifier, configstore.DefaultStore)
}

// LoadKeyFromStore instantiates a new encryption key for a given identifier from a specific store instance.
//
// If several keys are found for the identifier, they are sorted by timestamp, and a composite key is returned.
// The most recent key will be used for encryption, and decryption will be done by any of them.
// There needs to be _only one_ key with the highest priority for the identifier.
//
// If the key configuration specifies it is sealed, the key returned will be wrapped by an unseal mechanism.
// When the symmecrypt/seal global singleton gets unsealed, the key will become usable instantly. It will return errors in the meantime.
//
// The key cipher name is expected to match a KeyFactory that got registered through RegisterCipher().
// Either use a built-in cipher, or make sure to register a proper factory for this cipher.
// This KeyFactory will be called, either directly or when the symmecrypt/seal global singleton gets unsealed, if applicable.
func LoadKeyFromStore(identifier string, store *configstore.Store) (symmecrypt.Key, error) {

	items, err := ConfigFilter.Slice(identifier).Store(store).GetItemList()
	if err != nil {
		return nil, err
	}

	switch configstore.Filter().Squash().Apply(items).Len() {
	case 0:
		return nil, fmt.Errorf("encryption key '%s' not found", identifier)
	case 1: // OK, single key with highest prio
	default:
		return nil, fmt.Errorf("ambiguous config: several encryption keys conflicting for '%s'", identifier)
	}

	comp := symmecrypt.CompositeKey{}

	hadNonSealed := false

	for _, item := range items.Items {

		i, err := item.Unmarshaled()
		if err != nil {
			return nil, err
		}
		var ref symmecrypt.Key
		cfg := i.(*KeyConfig)
		factory, err := symmecrypt.GetKeyFactory(cfg.Cipher)
		if err != nil {
			return nil, err
		}
		if cfg.Sealed {
			if hadNonSealed {
				panic(fmt.Sprintf("encryption key '%s': DANGER! Detected downgrade to non-sealed encryption key. Non-sealed key has higher priority, this looks malicious. Aborting!", identifier))
			}
			ref = newSealedKey(cfg, factory)
		} else {
			hadNonSealed = true
			ref, err = factory.NewKey(cfg.Key)
			if err != nil {
				return nil, err
			}
		}

		comp = append(comp, ref)
	}

	if len(comp) == 1 {
		return comp[0], nil
	}

	return comp, nil
}

// LoadSingleKey instantiates a new encryption key using LoadKey from the default store in configstore without specifying its identifier.
// It will error if several different identifiers are found.
func LoadSingleKey() (symmecrypt.Key, error) {
	return LoadSingleKeyFromStore(configstore.DefaultStore)
}

// LoadSingleKey instantiates a new encryption key using LoadKey from a specific store instance without specifying its identifier.
// It will error if several different identifiers are found.
func LoadSingleKeyFromStore(store *configstore.Store) (symmecrypt.Key, error) {
	ident, err := singleKeyIdentifier(store)
	if err != nil {
		return nil, err
	}
	return LoadKeyFromStore(ident, store)
}

func singleKeyIdentifier(store *configstore.Store) (string, error) {
	items, err := ConfigFilter.Store(store).GetItemList()
	if err != nil {
		return "", err
	}

	keys := items.Keys()
	switch len(keys) {
	case 0:
		return "", errors.New("no encryption key found")
	case 1:
		return keys[0], nil
	}

	return "", errors.New("ambiguous config: several encryption keys found and no identifier supplied")
}

// WatchKey instantiates a new hot-reloading encryption key from the default store in configstore.
// It uses LoadKey(), so the underlying implementation can be anything supported (composite, sealed, any cipher, ...)
func WatchKey(identifier string) (symmecrypt.Key, error) {
	return WatchKeyFromStore(identifier, configstore.DefaultStore)
}

// WatchKeyFromStore instantiates a new hot-reloading encryption key from a specific store instance.
// It uses LoadKey(), so the underlying implementation can be anything supported (composite, sealed, any cipher, ...)
func WatchKeyFromStore(identifier string, store *configstore.Store) (symmecrypt.Key, error) {
	b, err := LoadKeyFromStore(identifier, store)
	if err != nil {
		return nil, err
	}

	holder := &watchKey{identifier: identifier, k: b}
	go holder.watch(store)

	return holder, nil
}

// WatchSingleKey instantiates a new hot-reloading encryption key from the default store in configstore without specifying its identifier.
// It will error if several different identifiers are found.
func WatchSingleKey() (symmecrypt.Key, error) {
	return WatchSingleKeyFromStore(configstore.DefaultStore)
}

// WatchSingleKey instantiates a new hot-reloading encryption key from a specific store instance without specifying its identifier.
// It will error if several different identifiers are found.
func WatchSingleKeyFromStore(store *configstore.Store) (symmecrypt.Key, error) {
	ident, err := singleKeyIdentifier(store)
	if err != nil {
		return nil, err
	}
	return WatchKeyFromStore(ident, store)
}

/*
** WATCH implementation: self updating encryption keys
 */

// Watch for configstore update notifications, then reload the key through LoadKey().
func (kh *watchKey) watch(store *configstore.Store) {
	for range store.Watch() {
		time.Sleep(10 * time.Millisecond)
		// small sleep to yield to symmecrypt/seal in case of seal change
		b, err := LoadKeyFromStore(kh.identifier, store)
		if err != nil {
			logrus.Errorf("symmecrypt/keyloader: configuration fetch error for key '%s': %s", kh.identifier, err)
			continue
		}
		k := kh.Key()
		k.Wait()
		kh.mut.Lock()
		kh.k = b
		kh.mut.Unlock()
	}
}

func (kh *watchKey) Encrypt(text []byte, extra ...[]byte) ([]byte, error) {
	k := kh.Key()
	return k.Encrypt(text, extra...)
}

func (kh *watchKey) Decrypt(text []byte, extra ...[]byte) ([]byte, error) {
	k := kh.Key()
	return k.Decrypt(text, extra...)
}

func (kh *watchKey) EncryptMarshal(i interface{}, extra ...[]byte) (string, error) {
	k := kh.Key()
	return k.EncryptMarshal(i, extra...)
}

func (kh *watchKey) DecryptMarshal(s string, target interface{}, extra ...[]byte) error {
	k := kh.Key()
	return k.DecryptMarshal(s, target, extra...)
}

func (kh *watchKey) Wait() {
	k := kh.Key()
	k.Wait()
}

func (kh *watchKey) String() (string, error) {
	k := kh.Key()
	return k.String()
}

func (kh *watchKey) Key() symmecrypt.Key {
	kh.mut.RLock()
	ret := kh.k
	kh.mut.RUnlock()
	return ret
}

/*
** SEALED implementation: encryption keys sealed/encrypted by a master encryption key which is split in shamir shards
 */

// Return an instance of sealedKey, that will decrypt itself with the crypto/seal singleton when it gets unsealed.
// If there is a misconfiguration (no crypto/seal configured, the key decryption fails, or the key factory fails), THIS WILL PANIC.
func newSealedKey(cfg *KeyConfig, factory symmecrypt.KeyFactory) symmecrypt.Key {
	ret := &sealedKey{waitCh: make(chan struct{})}
	go func() {
		if !seal.WaitUnseal() {
			panic(fmt.Sprintf("Trying to unseal encryption key '%s': no seal configured", cfg.Identifier))
		}
		decK, err := UnsealKey(cfg, seal.Global())
		if err != nil {
			panic(fmt.Sprintf("Sealed encryption key '%s' cannot be decrypted: %s", cfg.Identifier, err.Error()))
		}
		ret.decryptedKey, err = factory.NewKey(decK.Key)
		if err != nil {
			panic(fmt.Sprintf("Sealed encryption key '%s' cannot be initialized: %s", cfg.Identifier, err.Error()))
		}
		atomic.StoreUint32(&ret.decrypted, 1)
		close(ret.waitCh)
	}()
	return ret
}

func (s *sealedKey) Key() (symmecrypt.Key, error) {
	if atomic.LoadUint32(&s.decrypted) == 0 {
		return nil, errors.NewMethodNotAllowed(nil, "encryption key is sealed")
	}
	return s.decryptedKey, nil
}

func (s *sealedKey) Encrypt(text []byte, extra ...[]byte) ([]byte, error) {
	k, err := s.Key()
	if err != nil {
		return nil, err
	}
	return k.Encrypt(text, extra...)
}

func (s *sealedKey) Decrypt(text []byte, extra ...[]byte) ([]byte, error) {
	k, err := s.Key()
	if err != nil {
		return nil, err
	}
	return k.Decrypt(text, extra...)
}

func (s *sealedKey) EncryptMarshal(i interface{}, extra ...[]byte) (string, error) {
	k, err := s.Key()
	if err != nil {
		return "", err
	}
	return k.EncryptMarshal(i, extra...)
}

func (s *sealedKey) DecryptMarshal(str string, target interface{}, extra ...[]byte) error {
	k, err := s.Key()
	if err != nil {
		return err
	}
	return k.DecryptMarshal(str, target, extra...)
}

func (s *sealedKey) Wait() {
	<-s.waitCh
}

func (s *sealedKey) String() (string, error) {
	k, err := s.Key()
	if err != nil {
		return "", err
	}
	return k.String()
}
