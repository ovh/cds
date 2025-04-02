package sdk

import (
	"time"
)

type DatabaseMigrationStatus struct {
	ID        string     `json:"id" db:"id" cli:"id,key"`
	Migrated  bool       `json:"migrated" db:"-" cli:"migrated"`
	AppliedAt *time.Time `json:"applied_at" db:"applied_at" cli:"applied_at"`
	// aggregates
	Database string `json:"database" db:"-" cli:"database"`
}

type DatabaseEntity struct {
	Name           string                  `json:"name"`
	Encrypted      bool                    `json:"encrypted,omitempty"`
	Signed         bool                    `json:"signed,omitempty"`
	CanonicalForms []DatabaseCanonicalForm `json:"canonical_forms,omitempty"`
}

type DatabaseCanonicalForm struct {
	Signer string `json:"signer" db:"signer"`
	Number int64  `json:"number" db:"number"`
	Latest bool   `json:"latest,omitempty"`
}

type DatabaseEntityInfo struct {
	PK           string `json:"pk"`
	Encrypted    bool   `json:"encrypted,omitempty"`
	EncryptionTS int64  `json:"encryption_ts,omitempty"`
	Signed       bool   `json:"signed,omitempty"`
	SignatureTS  int64  `json:"signature_ts,omitempty"`
}
