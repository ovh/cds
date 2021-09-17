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

	if err := m.Update(db, tuple); err != nil {
		return err
	}

	return nil
}
