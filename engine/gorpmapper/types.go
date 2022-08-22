package gorpmapper

import (
	"github.com/go-gorp/gorp"
)

// SignedEntity struct for signed entity stored in database.
type SignedEntity struct {
	Signature []byte `json:"-" db:"sig"`
}

func (s SignedEntity) GetSignature() []byte {
	return s.Signature
}

type CanonicalForm string

func (f *CanonicalForm) Bytes() []byte {
	return []byte(string(*f))
}

func (f *CanonicalForm) String() string {
	return string(*f)
}

type CanonicalForms []CanonicalForm

func (fs CanonicalForms) Latest() (*CanonicalForm, CanonicalForms) {
	if len(fs) == 0 {
		return nil, nil
	}
	if len(fs) == 1 {
		return &fs[0], []CanonicalForm{}
	}
	return &fs[0], fs[1:]
}

type TestEncryptedData struct {
	SignedEntity
	ID                   int64             `db:"id"`
	Data                 string            `db:"data"`
	SensitiveData        string            `db:"sensitive_data" gorpmapping:"encrypted,Data"`
	AnotherSensitiveData string            `db:"another_sensitive_data" gorpmapping:"encrypted,ID,Data"`
	SensitiveJsonData    SensitiveJsonData `db:"sensitive_json_data" gorpmapping:"encrypted,ID"`
}

type SensitiveJsonData struct {
	Data string
}

func (e TestEncryptedData) Canonical() CanonicalForms {
	return CanonicalForms{
		"{{.ID}} {{.Data}}",
	}
}

type SqlExecutorWithTx interface {
	gorp.SqlExecutor
	Commit() error
	Rollback() error
}
