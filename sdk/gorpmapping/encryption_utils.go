package gorpmapping

import (
	"errors"

	"github.com/ovh/cds/sdk"

	"github.com/go-gorp/gorp"
)

func ListEncryptedEntities() []string {
	var encryptedEntities []string
	for k, v := range Mapping {
		if v.EncryptedEntity {
			encryptedEntities = append(encryptedEntities, k)
		}
	}
	return encryptedEntities
}

func RollEncryptedTupleByPrimaryKey(db gorp.SqlExecutor, entity string, pk interface{}) error {
	e, ok := Mapping[entity]
	if !ok {
		return sdk.WithStack(errors.New("unknown entity"))
	}
	if !e.EncryptedEntity {
		return sdk.WithStack(errors.New("entity is not encrypted"))
	}

	tuple, err := LoadTupleByPrimaryKey(db, entity, pk)
	if err != nil {
		return err
	}

	if err := Update(db, tuple); err != nil {
		return err
	}

	return nil
}
