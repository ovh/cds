package sdk

import (
	"database/sql/driver"
	"encoding/json"
	"time"

	"github.com/pkg/errors"
)

type UserGPGKey struct {
	ID                 string         `json:"id" db:"id"`
	AuthentifiedUserID string         `json:"authentified_user_id" db:"authentified_user_id"`
	KeyID              string         `json:"key_id" db:"key_id" cli:"key_id"`
	PublicKey          string         `json:"public_key" db:"public_key"`
	SubKeys            UserGPGSubKeys `json:"sub_keys" db:"sub_keys"`
	Created            time.Time      `json:"created" db:"created"`
}

type UserGPGSubKey struct {
	KeyID string `json:"key_id"`
}

// UserGPGSubKeys type used for database json storage.
type UserGPGSubKeys []UserGPGSubKey

// Scan subkeys from JSONB.
func (s *UserGPGSubKeys) Scan(src interface{}) error {
	if src == nil {
		return nil
	}
	source, ok := src.([]byte)
	if !ok {
		return WithStack(errors.New("type assertion .([]byte) failed"))
	}
	return WrapError(json.Unmarshal(source, s), "cannot unmarshal UserGPGSubKeys")
}

// Value returns driver.Value from subkeys slice.
func (s UserGPGSubKeys) Value() (driver.Value, error) {
	if s == nil {
		return nil, nil
	}
	j, err := json.Marshal(s)
	return j, WrapError(err, "cannot marshal UserGPGSubKeys")
}
