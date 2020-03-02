package environment

import (
	"context"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

// InsertKey a new environment key in database
func InsertKey(db gorp.SqlExecutor, key *sdk.EnvironmentKey) error {
	dbEnvironmentKey := dbEnvironmentKey{EnvironmentKey: *key}
	if err := gorpmapping.InsertAndSign(context.Background(), db, &dbEnvironmentKey); err != nil {
		return err
	}
	*key = dbEnvironmentKey.EnvironmentKey
	return nil
}

func UpdateKey(db gorp.SqlExecutor, key *sdk.EnvironmentKey) error {
	dbEnvironmentKey := dbEnvironmentKey{EnvironmentKey: *key}
	if err := gorpmapping.UpdateAndSign(context.Background(), db, &dbEnvironmentKey); err != nil {
		return err
	}
	*key = dbEnvironmentKey.EnvironmentKey
	return nil
}

func getAllKeys(db gorp.SqlExecutor, query gorpmapping.Query) ([]sdk.EnvironmentKey, error) {
	var ctx = context.Background()
	var res []dbEnvironmentKey
	keys := make([]sdk.EnvironmentKey, 0, len(res))

	if err := gorpmapping.GetAll(ctx, db, query, &res); err != nil {
		return nil, err
	}

	for i := range res {
		isValid, err := gorpmapping.CheckSignature(res[i], res[i].Signature)
		if err != nil {
			return nil, err
		}
		if !isValid {
			log.Error(ctx, "environment.getAllKeys> environment key %d data corrupted", res[i].ID)
			continue
		}
		keys = append(keys, res[i].EnvironmentKey)
	}
	return keys, nil
}

// LoadAllKeys load all keys for the given environment
func LoadAllKeys(db gorp.SqlExecutor, envID int64) ([]sdk.EnvironmentKey, error) {
	query := gorpmapping.NewQuery("SELECT * FROM environment_key WHERE environment_id = $1").Args(envID)
	return getAllKeys(db, query)
}

// LoadAllKeysWithPrivateContent load all keys for the given environment
func LoadAllKeysWithPrivateContent(db gorp.SqlExecutor, envID int64) ([]sdk.EnvironmentKey, error) {
	keys, err := LoadAllKeys(db, envID)
	if err != nil {
		return nil, err
	}

	res := make([]sdk.EnvironmentKey, 0, len(keys))
	for _, k := range keys {
		x, err := LoadKey(db, k.ID, k.Name)
		if err != nil {
			return nil, err
		}
		res = append(res, *x)
	}

	return res, nil
}

func LoadKey(db gorp.SqlExecutor, id int64, keyName string) (*sdk.EnvironmentKey, error) {
	query := gorpmapping.NewQuery(`
	SELECT * 
	FROM environment_key
	WHERE id = $1 
	AND name = $2
	`).Args(id, keyName)
	var k dbEnvironmentKey
	found, err := gorpmapping.Get(context.Background(), db, query, &k, gorpmapping.GetOptions.WithDecryption)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, sdk.WithStack(sdk.ErrNotFound)
	}
	isValid, err := gorpmapping.CheckSignature(k, k.Signature)
	if err != nil {
		return nil, err
	}
	if !isValid {
		log.Error(context.Background(), "environment.LoadKey> project key %d data corrupted", k.ID)
		return nil, sdk.WithStack(sdk.ErrNotFound)
	}
	return &k.EnvironmentKey, nil
}

// DeleteEnvironmentKey Delete the given key from the given project
func DeleteEnvironmentKey(db gorp.SqlExecutor, envID int64, keyName string) error {
	_, err := db.Exec("DELETE FROM environment_key WHERE environment_id = $1 AND name = $2", envID, keyName)
	return sdk.WrapError(err, "Cannot delete key %s", keyName)
}
