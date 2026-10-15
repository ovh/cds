package sdk

import "time"

type UserGPGKey struct {
	ID                 string      `json:"id" db:"id"`
	AuthentifiedUserID string      `json:"authentified_user_id" db:"authentified_user_id"`
	KeyID              string      `json:"key_id" db:"key_id" cli:"key_id"`
	SubKeyIDs          StringSlice `json:"sub_key_ids" db:"sub_key_ids"`
	PublicKey          string      `json:"public_key" db:"public_key"`
	Created            time.Time   `json:"created" db:"created"`
}
