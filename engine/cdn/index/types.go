package index

import (
	"time"

	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
)

type ItemType string

func (t ItemType) Validate() error {
	switch t {
	case TypeItemStepLog, TypeItemServiceLog:
		return nil
	}
	return sdk.NewErrorFrom(sdk.ErrWrongRequest, "invalid item type")
}

const (
	TypeItemStepLog    ItemType = "step-log"
	TypeItemServiceLog ItemType = "service-log"
)

const (
	StatusItemIncoming  = "Incoming"
	StatusItemCompleted = "Completed"
)

type Item struct {
	gorpmapper.SignedEntity
	ID           string           `json:"id" db:"id"`
	Created      time.Time        `json:"created" db:"created"`
	LastModified time.Time        `json:"last_modified" db:"last_modified"`
	Hash         string           `json:"-" db:"cipher_hash" gorpmapping:"encrypted,ID,APIRefHash,Type"`
	APIRef       sdk.CDNLogAPIRef `json:"api_ref" db:"api_ref"`
	APIRefHash   string           `json:"api_ref_hash" db:"api_ref_hash"`
	Status       string           `json:"status" db:"status"`
	Type         ItemType         `json:"type" db:"type"`
	Size         int64            `json:"size" db:"size"`
	MD5          string           `json:"md5" db:"md5"`
}
