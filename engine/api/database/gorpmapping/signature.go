package gorpmapping

import (
	"bytes"
	"context"
	"crypto/sha1"
	"fmt"
	"sync"
	"text/template"

	"github.com/go-gorp/gorp"
	"github.com/ovh/symmecrypt"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

// Constant for gorp mapping.
const (
	KeySignIdentifier = "db-sign"
)

// SignedEntity struct for signed entity stored in database.
type SignedEntity struct {
	Signature []byte `json:"-" db:"sig"`
}

func (s SignedEntity) GetSignature() []byte {
	return s.Signature
}

type Signed interface {
	GetSignature() []byte
}

// Canonicaller returns a byte array that represent its data.
type Canonicaller interface {
	Canonical() CanonicalForms
}

type CanonicalForms []CanonicalForm
type CanonicalForm string

func (f *CanonicalForm) Bytes() []byte {
	return []byte(string(*f))
}

func (f *CanonicalForm) String() string {
	return string(*f)
}

func (fs CanonicalForms) Latest() (*CanonicalForm, CanonicalForms) {
	if len(fs) == 0 {
		return nil, nil
	}
	if len(fs) == 1 {
		return &fs[0], []CanonicalForm{}
	}
	return &fs[0], fs[1:]
}

// InsertAndSign a data in database, given data should implement canonicaller interface.
func InsertAndSign(ctx context.Context, db gorp.SqlExecutor, i Canonicaller) error {
	if err := Insert(db, i); err != nil {
		return err
	}
	return sdk.WithStack(dbSign(ctx, db, i))
}

// UpdateAndSign a data in database, given data should implement canonicaller interface.
func UpdateAndSign(ctx context.Context, db gorp.SqlExecutor, i Canonicaller) error {
	if err := Update(db, i); err != nil {
		return err
	}
	return sdk.WithStack(dbSign(ctx, db, i))
}

var CanonicalFormTemplates = struct {
	m map[string]*template.Template
	l sync.RWMutex
}{
	m: make(map[string]*template.Template),
}

func getSigner(f *CanonicalForm) string {
	h := sha1.New()
	_, _ = h.Write(f.Bytes())
	bs := h.Sum(nil)
	sha := fmt.Sprintf("%x", bs)
	return sha
}

func canonicalTemplate(data Canonicaller) (string, *template.Template, error) {
	f, _ := data.Canonical().Latest()
	if f == nil {
		return "", nil, sdk.WithStack(fmt.Errorf("no canonical function available for %T", data))
	}

	sha := getSigner(f)

	CanonicalFormTemplates.l.RLock()
	t, has := CanonicalFormTemplates.m[sha]
	CanonicalFormTemplates.l.RUnlock()

	if !has {
		return "", nil, sdk.WithStack(fmt.Errorf("no canonical function available for %T", data))
	}

	return sha, t, nil
}

func getCanonicalTemplate(f *CanonicalForm) (*template.Template, error) {
	sha := getSigner(f)

	CanonicalFormTemplates.l.RLock()
	t, has := CanonicalFormTemplates.m[sha]
	CanonicalFormTemplates.l.RUnlock()

	if !has {
		return nil, sdk.WithStack(fmt.Errorf("no canonical function available"))
	}

	return t, nil
}

func sign(data Canonicaller) (string, []byte, error) {
	signer, tmpl, err := canonicalTemplate(data)
	if err != nil {
		return "", nil, err
	}

	if tmpl == nil {
		err := fmt.Errorf("unable to get canonical form template for %T", data)
		return "", nil, sdk.WrapError(err, "unable to sign data")
	}

	var clearContent = new(bytes.Buffer)
	if err := tmpl.Execute(clearContent, data); err != nil {
		return "", nil, sdk.WrapError(err, "unable to sign data")
	}

	btes, err := signatureKey.Encrypt(clearContent.Bytes())
	if err != nil {
		return "", nil, sdk.WithStack(fmt.Errorf("unable to encrypt content: %v", err))
	}

	return signer, btes, nil
}

// CheckSignature return true if a given signature is valid for given object.
func CheckSignature(i Canonicaller, sig []byte) (bool, error) {
	var CanonicalForms = i.Canonical()
	var f *CanonicalForm
	for {
		f, CanonicalForms = CanonicalForms.Latest()
		if f == nil {
			return false, nil
		}
		ok, err := checkSignature(i, signatureKey, f, sig)
		if err != nil {
			return ok, err
		}
		if ok {
			return true, nil
		}
	}
}

func checkSignature(i Canonicaller, k symmecrypt.Key, f *CanonicalForm, sig []byte) (bool, error) {
	tmpl, err := getCanonicalTemplate(f)
	if err != nil {
		return false, err
	}

	var clearContent = new(bytes.Buffer)
	if err := tmpl.Execute(clearContent, i); err != nil {
		return false, nil
	}

	decryptedSig, err := k.Decrypt(sig)
	if err != nil {
		return false, sdk.WrapError(err, "unable to decrypt content")
	}

	res := clearContent.String() == string(decryptedSig)

	return res, nil
}

func dbSign(ctx context.Context, db gorp.SqlExecutor, i Canonicaller) error {
	signer, signature, err := sign(i)
	if err != nil {
		return err
	}

	table, key, id, err := dbMappingPKey(i)
	if err != nil {
		return sdk.WrapError(err, "primary key field not found in table: %s", table)
	}

	query := fmt.Sprintf(`UPDATE "%s" SET sig = $2, signer = $3 WHERE %s = $1`, table, key)
	res, err := db.Exec(query, id, signature, signer)
	if err != nil {
		log.Error(ctx, "error executing query %s with parameters %s, %s: %v", query, table, key, err)
		return sdk.WithStack(err)
	}

	n, _ := res.RowsAffected()
	if n != 1 {
		return sdk.WithStack(fmt.Errorf("%d number of rows affected (table=%s, key=%s, id=%v)", n, table, key, id))
	}
	return nil
}
