package seal

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sync"

	sssa "github.com/SSSaaS/sssa-golang"
	"github.com/juju/errors"
	"github.com/ovh/configstore"
	"github.com/ovh/symmecrypt"
	"github.com/ovh/symmecrypt/ciphers/aesgcm"
	"github.com/sirupsen/logrus"
)

const (
	nonceLen = 16

	// ConfigName is the name of the config item in the configstore
	ConfigName = "seal"

	encryptionCipher = aesgcm.CipherName
)

var (
	global    *Seal
	globalMut sync.Mutex

	// SealConfigFilter is the configstore manipulation filter used to retrieve the seal definition
	SealConfigFilter = configstore.Filter().Slice(ConfigName).Unmarshal(func() interface{} { return &Seal{} })
)

// Seal is a shamir-sharded encryption key.
type Seal struct {
	Min           uint   `json:"min"`
	Total         uint   `json:"total"`
	Nonce         string `json:"nonce"`
	Progress      uint   `json:"progress,omitempty"`
	Unsealed      bool   `json:"unsealed,omitempty"`
	shards        map[string]struct{}
	encryptionKey symmecrypt.Key
	mut           sync.Mutex
	unsealedCh    chan struct{}
}

type internalKey struct {
	Nonce string `json:"nonce"`
	Key   string `json:"key"`
}

/*
** SEAL SINGLETON
 */

// InitFromConfig initializes the global singleton seal from the configstore.
func InitFromConfig(onChange func(*Seal)) error {
	return InitFromStore(onChange, configstore.DefaultStore)
}

// InitFromStore initializes the global singleton seal from a specific store instance.
func InitFromStore(onChange func(*Seal), s *configstore.Store) error {
	seal, err := NewSealFromStore(s)
	if err != nil {
		return err
	}
	if seal != nil {
		seal.unsealedCh = make(chan struct{})
		setGlobal(seal)
	}
	go func() {
		for range s.Watch() {
			newSeal, err := NewSealFromStore(s)
			if err != nil {
				logrus.Errorf("symmecrypt/seal: configuration fetch error: %s", err)
				continue
			}
			if diff(seal, newSeal) && onChange != nil {
				onChange(newSeal)
			}
		}
	}()
	return nil
}

func setGlobal(r *Seal) {
	globalMut.Lock()
	defer globalMut.Unlock()
	global = r
}

// Global returns the global singleton seal.
func Global() *Seal {
	globalMut.Lock()
	defer globalMut.Unlock()
	return global
}

func diff(r1, r2 *Seal) bool {
	if r1 == nil && r2 == nil {
		return false
	}
	if (r1 == nil && r2 != nil) || (r1 != nil && r2 == nil) {
		return true
	}
	return r1.Nonce != r2.Nonce
}

// Exists returns true if a global singleton seal is configured and initialized.
func Exists() bool {
	return Global() != nil
}

/*
** CONSTRUCTORS
 */

// NewSealFromConfig instantiates a new Seal from the configstore.
func NewSealFromConfig() (*Seal, error) {
	return NewSealFromStore(configstore.DefaultStore)
}

// NewSealFromStore instantiates a new Seal from a specific store instance.
func NewSealFromStore(s *configstore.Store) (*Seal, error) {
	sec, err := SealConfigFilter.Store(s).GetItem(ConfigName)
	if err != nil {
		if _, ok := err.(configstore.ErrItemNotFound); ok {
			// Not found in config: disabled
			return nil, nil
		}
		return nil, err
	}
	i, err := sec.Unmarshaled()
	if err != nil {
		return nil, err
	}
	root := i.(*Seal)

	return &Seal{Min: root.Min, Total: root.Total, Nonce: root.Nonce, shards: map[string]struct{}{}}, nil
}

// NewRandom instantiates a new random Seal.
func NewRandom(min, total uint) (*Seal, []string, error) {
	k, err := symmecrypt.NewRandomKey(encryptionCipher)
	if err != nil {
		return nil, nil, err
	}
	hexK, err := k.String()
	if err != nil {
		return nil, nil, err
	}
	ik := &internalKey{Key: hexK, Nonce: generateNonce()}
	j, err := json.Marshal(ik)
	if err != nil {
		return nil, nil, err
	}
	shards, err := create(min, total, j)
	if err != nil {
		return nil, nil, err
	}
	return &Seal{Min: min, Total: total, Nonce: ik.Nonce, encryptionKey: k, shards: map[string]struct{}{}}, shards, nil
}

func create(min, total uint, content []byte) ([]string, error) {
	return sssa.Create(int(min), int(total), string(content))
}

func generateNonce() string {
	b := make([]byte, nonceLen)
	_, err := rand.Read(b)
	if err != nil {
		panic(err) // stop what you're doing RIGHT NOW
	}
	return hex.EncodeToString(b)
}

/*
** IMPLEMENTATION
 */

// Sealed returns false is the Seal instance is nil or unsealed.
// It returns true if it is initialized but still sealed.
func (r *Seal) Sealed() bool {
	if r == nil {
		return false
	}
	r.mut.Lock()
	defer r.mut.Unlock()

	return r.encryptionKey == nil
}

// WaitUnseal waits for the global singleton instance to become unsealed.
// It returns false if the global singleton instance is not initialized, otherwise it blocks and eventually returns true.
func WaitUnseal() bool {
	global := Global()
	if global == nil {
		return false
	}
	<-global.unsealedCh
	return true
}

// AddShard adds a new shard to the Seal instance, trying to unseal it.
// It returns an error if the instance is nil, already unsealed or if the shard is invalid.
// It returns true if the seal became unsealed.
func (r *Seal) AddShard(s string) (bool, error) {

	if r == nil {
		return false, errors.NewBadRequest(nil, "no shamir root!")
	}

	r.mut.Lock()
	defer r.mut.Unlock()

	if r.encryptionKey != nil {
		return false, errors.NewBadRequest(nil, "already unsealed!")
	}

	if !sssa.IsValidShare(s) {
		return false, errors.NewBadRequest(nil, "invalid shard")
	}

	r.shards[s] = struct{}{}
	r.Progress = uint(len(r.shards))

	if uint(len(r.shards)) < r.Min {
		return false, nil
	}

	// good to go!

	shards := []string{}
	for sh := range r.shards {
		shards = append(shards, sh)
	}

	plain, err := combine(shards)
	if err != nil {
		return r.reset(errors.NewBadRequest(nil, "bad shamir shards: invalid decoded payload"))
	}

	ik := &internalKey{}

	err = json.Unmarshal([]byte(plain), ik)
	if err != nil {
		return r.reset(errors.NewBadRequest(nil, "bad shamir shards: invalid decoded payload"))
	}

	if ik.Nonce != r.Nonce {
		return r.reset(errors.NewBadRequest(nil, "old shamir shards: nonce is outdated"))
	}

	k, err := symmecrypt.NewKey(encryptionCipher, ik.Key)
	if err != nil {
		return r.reset(errors.NewBadRequest(nil, fmt.Sprintf("embedded encryption key is invalid: %s", err)))
	}

	if r.unsealedCh != nil {
		close(r.unsealedCh)
	}
	r.Unsealed = true
	r.encryptionKey = k
	r.shards = map[string]struct{}{}

	return true, nil
}

func combine(shards []string) (string, error) {
	return sssa.Combine(shards)
}

// Encrypt arbitrary data. Extra data can be passed for MAC. Returns a printable hex-representation of the encrypted value.
func (r *Seal) Encrypt(b []byte, extra ...[]byte) (string, error) {

	if r == nil {
		return "", errors.NewBadRequest(nil, "seal is not initialized")
	}

	r.mut.Lock()
	defer r.mut.Unlock()

	if r.encryptionKey == nil {
		return "", errors.NewBadRequest(nil, "seal is still sealed!")
	}

	bEnc, err := r.encryptionKey.Encrypt(b, extra...)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(bEnc), nil
}

// Decrypt arbitrary data from a hex-representation string. Extra data can be passed for MAC.
func (r *Seal) Decrypt(s string, extra ...[]byte) ([]byte, error) {

	if r == nil {
		return nil, errors.NewBadRequest(nil, "seal is not initialized")
	}

	r.mut.Lock()
	defer r.mut.Unlock()

	if r.encryptionKey == nil {
		return nil, errors.NewBadRequest(nil, "seal is still sealed!")
	}

	bEnc, err := hex.DecodeString(s)
	if err != nil {
		return nil, errors.NewBadRequest(nil, "invalid hex")
	}
	retBytes, err := r.encryptionKey.Decrypt(bEnc, extra...)
	return retBytes, err
}

func (r *Seal) reset(err error) (bool, error) {
	r.Unsealed = false
	r.Progress = 0
	r.shards = map[string]struct{}{}
	return false, err
}
