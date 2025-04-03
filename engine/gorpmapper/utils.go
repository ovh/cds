package gorpmapper

import (
	"context"
	"errors"
	"fmt"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/sdk"
)

func (m *Mapper) ListCanonicalFormsByEntity(db gorp.SqlExecutor, entity string) ([]sdk.DatabaseCanonicalForm, error) {
	e, ok := m.Mapping[entity]
	if !ok {
		return nil, sdk.WithStack(errors.New("unknown entity"))
	}
	if !e.SignedEntity {
		return nil, sdk.WithStack(errors.New("entity is not signed"))
	}

	var res []sdk.DatabaseCanonicalForm
	if _, err := db.Select(&res, fmt.Sprintf(`SELECT signer, count(sig) AS number FROM "%s" GROUP BY signer`, e.Name)); err != nil {
		return nil, sdk.WithStack(err)
	}

	x := e.Target.(Canonicaller)
	lastestCanonicalForm, _ := x.Canonical().Latest()
	sha := GetSigner(lastestCanonicalForm)

	for i := range res {
		if res[i].Signer == sha {
			res[i].Latest = true
		}
	}

	return res, nil
}

func (m *Mapper) ListTuplesByEntity(db gorp.SqlExecutor, entity string) ([]string, error) {
	e, ok := m.Mapping[entity]
	if !ok {
		return nil, sdk.WithStack(errors.New("unknown entity"))
	}
	if !e.SignedEntity && !e.EncryptedEntity {
		return nil, errors.New("entity is not signed or encrypted")
	}

	var res []string
	if _, err := db.Select(&res, fmt.Sprintf(`SELECT %s::text FROM "%s"`, e.Keys[0], e.Name)); err != nil {
		return nil, sdk.WithStack(err)
	}

	return res, nil
}

func (m *Mapper) ListTuplesByCanonicalForm(db gorp.SqlExecutor, entity, signer string) ([]string, error) {
	e, ok := m.Mapping[entity]
	if !ok {
		return nil, sdk.WithStack(errors.New("unknown entity"))
	}
	if !e.SignedEntity {
		return nil, sdk.WithStack(errors.New("entity is not signed"))
	}

	query := fmt.Sprintf(`SELECT %s::text FROM "%s" WHERE signer = $1`, e.Keys[0], e.Name)
	var res []string
	if _, err := db.Select(&res, query, signer); err != nil {
		return nil, sdk.WithStack(err)
	}

	return res, nil
}

func (m *Mapper) InfoTupleByPrimaryKey(ctx context.Context, db gorp.SqlExecutor, entity string, pk string) (*sdk.DatabaseEntityInfo, error) {
	e, ok := m.Mapping[entity]
	if !ok {
		return nil, sdk.WithStack(errors.New("unknown entity"))
	}
	if !e.SignedEntity && !e.EncryptedEntity {
		return nil, errors.New("entity is not signed or encrypted")
	}

	res := sdk.DatabaseEntityInfo{
		PK: pk,
	}

	tuple, err := m.LoadTupleByPrimaryKey(ctx, db, entity, pk)
	if err != nil {
		if sdk.ErrorIs(err, sdk.ErrCorruptedData) {
			res.Corrupted = true
			return &res, nil
		}
		return nil, err
	}
	if tuple == nil {
		// Ignore missing tuple cause the data may be deleted while rolling is in progress
		return nil, nil
	}

	if e.SignedEntity {
		s, ok := tuple.(Signed)
		if !ok {
			return nil, sdk.WithStack(errors.New("invalid signed entity"))
		}
		_, keyIdx, err := m.CheckSignatureUncap(tuple.(Canonicaller), s.GetSignature())
		if err != nil {
			return nil, err
		}
		res.Signed = true
		res.SignatureTS = m.signatureKeyTimestamp[keyIdx]
	}

	if e.EncryptedEntity {
		keyIdx, err := getEncryptedDataUncap(ctx, m, db, tuple)
		if err != nil {
			return nil, err
		}
		res.Encrypted = true
		res.EncryptionTS = m.encryptionKeyTimestamp[keyIdx]
	}

	return &res, nil
}

func (m *Mapper) RollTupleByPrimaryKey(ctx context.Context, db SqlExecutorWithTx, entity string, pk string) (*sdk.DatabaseEntityInfo, error) {
	e, ok := m.Mapping[entity]
	if !ok {
		return nil, errors.New("unknown entity")
	}
	if !e.SignedEntity && !e.EncryptedEntity {
		return nil, errors.New("entity is not signed or encrypted")
	}

	tuple, err := m.LoadAndLockTupleByPrimaryKey(ctx, db, entity, pk)
	if err != nil {
		return nil, err
	}
	if tuple == nil {
		// Ignore missing tuple cause the data may be deleted while rolling is in progress
		return nil, nil
	}

	res := sdk.DatabaseEntityInfo{
		PK: pk,
	}

	if e.SignedEntity {
		if err := m.UpdateAndSign(ctx, db, tuple.(Canonicaller)); err != nil {
			return nil, err
		}
	} else {
		if err := m.Update(db, tuple); err != nil {
			return nil, err
		}
	}

	if e.SignedEntity {
		res.Signed = true
		res.SignatureTS = m.signatureKeyTimestamp[0]
	}

	if e.EncryptedEntity {
		res.Encrypted = true
		res.EncryptionTS = m.encryptionKeyTimestamp[0]
	}

	return &res, nil
}
