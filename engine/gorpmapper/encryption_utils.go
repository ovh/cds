package gorpmapper

import (
	"context"
	"errors"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/sdk"
)

func (m *Mapper) ListEncryptedEntities() []string {
	var encryptedEntities []string
	for k, v := range m.Mapping {
		if v.EncryptedEntity {
			encryptedEntities = append(encryptedEntities, k)
		}
	}
	return encryptedEntities
}

func (m *Mapper) RollEncryptedTupleByPrimaryKey(ctx context.Context, db gorp.SqlExecutor, entity string, pk interface{}) error {
	e, ok := m.Mapping[entity]
	if !ok {
		return sdk.WithStack(errors.New("unknown entity"))
	}
	if !e.EncryptedEntity {
		return sdk.WithStack(errors.New("entity is not encrypted"))
	}

	tuple, err := m.LoadAndLockTupleByPrimaryKey(ctx, db, entity, pk)
	if err != nil {
		return err
	}

	if tuple == nil {
		// Ignore missing tuple cause the data may be deleted while rolling is in progress
		return nil
	}

	if err := m.Update(db, tuple); err != nil {
		return err
	}

	return nil
}

func (m *Mapper) InfoEncryptedTupleByPrimaryKey(ctx context.Context, db gorp.SqlExecutor, entity string, pk interface{}) (int64, error) {
	e, ok := m.Mapping[entity]
	if !ok {
		return 0, sdk.WithStack(errors.New("unknown entity"))
	}
	if !e.EncryptedEntity {
		return 0, sdk.WithStack(errors.New("entity is not encrypted"))
	}

	tuple, err := m.LoadTupleByPrimaryKey(ctx, db, entity, pk)
	if err != nil {
		return 0, err
	}
	if tuple == nil {
		// Ignore missing tuple cause the data may be deleted while rolling is in progress
		return 0, nil
	}
	keyIdx, err := getEncryptedDataUncap(ctx, m, db, tuple)
	if err != nil {
		return 0, err
	}

	keyTimestamp := m.encryptionKeyTimestamp[keyIdx]

	return keyTimestamp, nil
}
