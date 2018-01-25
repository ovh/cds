package vault

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/vault/helper/jsonutil"
	"github.com/hashicorp/vault/physical"

	"github.com/keybase/go-crypto/openpgp"
	"github.com/keybase/go-crypto/openpgp/packet"
)

const (
	// barrierSealConfigPath is the path used to store our seal configuration.
	// This value is stored in plaintext, since we must be able to read it even
	// with the Vault sealed. This is required so that we know how many secret
	// parts must be used to reconstruct the master key.
	barrierSealConfigPath = "core/seal-config"

	// recoverySealConfigPath is the path to the recovery key seal
	// configuration. It is inside the barrier.
	recoverySealConfigPath = "core/recovery-seal-config"

	// recoveryKeyPath is the path to the recovery key
	recoveryKeyPath = "core/recovery-key"
)

type KeyNotFoundError struct {
	Err error
}

func (e *KeyNotFoundError) WrappedErrors() []error {
	return []error{e.Err}
}

func (e *KeyNotFoundError) Error() string {
	return e.Err.Error()
}

type Seal interface {
	SetCore(*Core)
	Init() error
	Finalize() error

	StoredKeysSupported() bool
	SetStoredKeys([][]byte) error
	GetStoredKeys() ([][]byte, error)

	BarrierType() string
	BarrierConfig() (*SealConfig, error)
	SetBarrierConfig(*SealConfig) error

	RecoveryKeySupported() bool
	RecoveryType() string
	RecoveryConfig() (*SealConfig, error)
	SetRecoveryConfig(*SealConfig) error
	SetRecoveryKey([]byte) error
	VerifyRecoveryKey([]byte) error
}

type DefaultSeal struct {
	config *SealConfig
	core   *Core
}

func (d *DefaultSeal) checkCore() error {
	if d.core == nil {
		return fmt.Errorf("seal does not have a core set")
	}
	return nil
}

func (d *DefaultSeal) SetCore(core *Core) {
	d.core = core
}

func (d *DefaultSeal) Init() error {
	return nil
}

func (d *DefaultSeal) Finalize() error {
	return nil
}

func (d *DefaultSeal) BarrierType() string {
	return "shamir"
}

func (d *DefaultSeal) StoredKeysSupported() bool {
	return false
}

func (d *DefaultSeal) RecoveryKeySupported() bool {
	return false
}

func (d *DefaultSeal) SetStoredKeys(keys [][]byte) error {
	return fmt.Errorf("core: stored keys are not supported")
}

func (d *DefaultSeal) GetStoredKeys() ([][]byte, error) {
	return nil, fmt.Errorf("core: stored keys are not supported")
}

func (d *DefaultSeal) BarrierConfig() (*SealConfig, error) {
	if d.config != nil {
		return d.config.Clone(), nil
	}

	if err := d.checkCore(); err != nil {
		return nil, err
	}

	// Fetch the core configuration
	pe, err := d.core.physical.Get(barrierSealConfigPath)
	if err != nil {
		d.core.logger.Error("core: failed to read seal configuration", "error", err)
		return nil, fmt.Errorf("failed to check seal configuration: %v", err)
	}

	// If the seal configuration is missing, we are not initialized
	if pe == nil {
		d.core.logger.Info("core: seal configuration missing, not initialized")
		return nil, nil
	}

	var conf SealConfig

	// Decode the barrier entry
	if err := jsonutil.DecodeJSON(pe.Value, &conf); err != nil {
		d.core.logger.Error("core: failed to decode seal configuration", "error", err)
		return nil, fmt.Errorf("failed to decode seal configuration: %v", err)
	}

	switch conf.Type {
	// This case should not be valid for other types as only this is the default
	case "":
		conf.Type = d.BarrierType()
	case d.BarrierType():
	default:
		d.core.logger.Error("core: barrier seal type does not match loaded type", "barrier_seal_type", conf.Type, "loaded_seal_type", d.BarrierType())
		return nil, fmt.Errorf("barrier seal type of %s does not match loaded type of %s", conf.Type, d.BarrierType())
	}

	// Check for a valid seal configuration
	if err := conf.Validate(); err != nil {
		d.core.logger.Error("core: invalid seal configuration", "error", err)
		return nil, fmt.Errorf("seal validation failed: %v", err)
	}

	d.config = &conf
	return d.config.Clone(), nil
}

func (d *DefaultSeal) SetBarrierConfig(config *SealConfig) error {
	if err := d.checkCore(); err != nil {
		return err
	}

	config.Type = d.BarrierType()

	// Encode the seal configuration
	buf, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to encode seal configuration: %v", err)
	}

	// Store the seal configuration
	pe := &physical.Entry{
		Key:   barrierSealConfigPath,
		Value: buf,
	}

	if err := d.core.physical.Put(pe); err != nil {
		d.core.logger.Error("core: failed to write seal configuration", "error", err)
		return fmt.Errorf("failed to write seal configuration: %v", err)
	}

	d.config = config.Clone()

	return nil
}

func (d *DefaultSeal) RecoveryType() string {
	return "unsupported"
}

func (d *DefaultSeal) RecoveryConfig() (*SealConfig, error) {
	return nil, fmt.Errorf("recovery not supported")
}

func (d *DefaultSeal) SetRecoveryConfig(config *SealConfig) error {
	return fmt.Errorf("recovery not supported")
}

func (d *DefaultSeal) VerifyRecoveryKey([]byte) error {
	return fmt.Errorf("recovery not supported")
}

func (d *DefaultSeal) SetRecoveryKey(key []byte) error {
	return fmt.Errorf("recovery not supported")
}

// SealConfig is used to describe the seal configuration
type SealConfig struct {
	// The type, for sanity checking
	Type string `json:"type"`

	// SecretShares is the number of shares the secret is split into. This is
	// the N value of Shamir.
	SecretShares int `json:"secret_shares"`

	// SecretThreshold is the number of parts required to open the vault. This
	// is the T value of Shamir.
	SecretThreshold int `json:"secret_threshold"`

	// PGPKeys is the array of public PGP keys used, if requested, to encrypt
	// the output unseal tokens. If provided, it sets the value of
	// SecretShares. Ordering is important.
	PGPKeys []string `json:"pgp_keys"`

	// Nonce is a nonce generated by Vault used to ensure that when unseal keys
	// are submitted for a rekey operation, the rekey operation itself is the
	// one intended. This prevents hijacking of the rekey operation, since it
	// is unauthenticated.
	Nonce string `json:"nonce"`

	// Backup indicates whether or not a backup of PGP-encrypted unseal keys
	// should be stored at coreUnsealKeysBackupPath after successful rekeying.
	Backup bool `json:"backup"`

	// How many keys to store, for seals that support storage.
	StoredShares int `json:"stored_shares"`
}

// Validate is used to sanity check the seal configuration
func (s *SealConfig) Validate() error {
	if s.SecretShares < 1 {
		return fmt.Errorf("shares must be at least one")
	}
	if s.SecretThreshold < 1 {
		return fmt.Errorf("threshold must be at least one")
	}
	if s.SecretShares > 1 && s.SecretThreshold == 1 {
		return fmt.Errorf("threshold must be greater than one for multiple shares")
	}
	if s.SecretShares > 255 {
		return fmt.Errorf("shares must be less than 256")
	}
	if s.SecretThreshold > 255 {
		return fmt.Errorf("threshold must be less than 256")
	}
	if s.SecretThreshold > s.SecretShares {
		return fmt.Errorf("threshold cannot be larger than shares")
	}
	if s.StoredShares > s.SecretShares {
		return fmt.Errorf("stored keys cannot be larger than shares")
	}
	if len(s.PGPKeys) > 0 && len(s.PGPKeys) != s.SecretShares-s.StoredShares {
		return fmt.Errorf("count mismatch between number of provided PGP keys and number of shares")
	}
	if len(s.PGPKeys) > 0 {
		for _, keystring := range s.PGPKeys {
			data, err := base64.StdEncoding.DecodeString(keystring)
			if err != nil {
				return fmt.Errorf("Error decoding given PGP key: %s", err)
			}
			_, err = openpgp.ReadEntity(packet.NewReader(bytes.NewBuffer(data)))
			if err != nil {
				return fmt.Errorf("Error parsing given PGP key: %s", err)
			}
		}
	}
	return nil
}

func (s *SealConfig) Clone() *SealConfig {
	ret := &SealConfig{
		Type:            s.Type,
		SecretShares:    s.SecretShares,
		SecretThreshold: s.SecretThreshold,
		Nonce:           s.Nonce,
		Backup:          s.Backup,
		StoredShares:    s.StoredShares,
	}
	if len(s.PGPKeys) > 0 {
		ret.PGPKeys = make([]string, len(s.PGPKeys))
		copy(ret.PGPKeys, s.PGPKeys)
	}
	return ret
}

type SealAccess struct {
	seal Seal
}

func (s *SealAccess) SetSeal(seal Seal) {
	s.seal = seal
}

func (s *SealAccess) StoredKeysSupported() bool {
	return s.seal.StoredKeysSupported()
}

func (s *SealAccess) BarrierConfig() (*SealConfig, error) {
	return s.seal.BarrierConfig()
}

func (s *SealAccess) RecoveryKeySupported() bool {
	return s.seal.RecoveryKeySupported()
}

func (s *SealAccess) RecoveryConfig() (*SealConfig, error) {
	return s.seal.RecoveryConfig()
}
