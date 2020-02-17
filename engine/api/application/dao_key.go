package application

import (
	"context"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

// InsertKey a new application key in database
func InsertKey(db gorp.SqlExecutor, key *sdk.ApplicationKey) error {
	var dbAppKey = dbApplicationKey{ApplicationKey: *key}
	if err := gorpmapping.InsertAndSign(context.Background(), db, &dbAppKey); err != nil {
		return err
	}
	*key = dbAppKey.ApplicationKey
	return nil
}

// UpdateKey a new application key in database.
// This function should be use only for migration purpose and should be removed
func UpdateKey(ctx context.Context, db gorp.SqlExecutor, key *sdk.ApplicationKey) error {
	var dbAppKey = dbApplicationKey{ApplicationKey: *key}
	if err := gorpmapping.UpdateAndSign(ctx, db, &dbAppKey); err != nil {
		return err
	}
	*key = dbAppKey.ApplicationKey
	return nil
}

func getAllKeys(db gorp.SqlExecutor, query gorpmapping.Query) ([]sdk.ApplicationKey, error) {
	var ctx = context.Background()
	var res []dbApplicationKey
	keys := make([]sdk.ApplicationKey, 0, len(res))

	if err := gorpmapping.GetAll(ctx, db, query, &res); err != nil {
		return nil, err
	}

	for i := range res {
		isValid, err := gorpmapping.CheckSignature(res[i], res[i].Signature)
		if err != nil {
			return nil, err
		}
		if !isValid {
			log.Error(ctx, "application.getAllKeys> application key %d data corrupted", res[i].ID)
			continue
		}
		keys = append(keys, res[i].ApplicationKey)
	}
	return keys, nil
}

// LoadAllKeys load all keys for the given application
func LoadAllKeys(db gorp.SqlExecutor, appID int64) ([]sdk.ApplicationKey, error) {
	query := gorpmapping.NewQuery(`
	SELECT * 
	FROM application_key
	WHERE application_id = $1`).Args(appID)
	return getAllKeys(db, query)
}

// LoadAllKeysWithPrivateContent load all keys for the given application
func LoadAllKeysWithPrivateContent(db gorp.SqlExecutor, appID int64) ([]sdk.ApplicationKey, error) {
	keys, err := LoadAllKeys(db, appID)
	if err != nil {
		return nil, err
	}

	var res []sdk.ApplicationKey
	for _, k := range keys {
		x, err := LoadKey(db, k.ID, k.Name)
		if err != nil {
			return nil, err
		}
		res = append(res, *x)
	}

	return res, nil
}

func LoadKey(db gorp.SqlExecutor, id int64, keyName string) (*sdk.ApplicationKey, error) {
	query := gorpmapping.NewQuery(`
	SELECT * 
	FROM application_key
	WHERE id = $1 AND name = $2`).Args(id, keyName)
	var k dbApplicationKey
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
		log.Error(context.Background(), "application.LoadKey> application key %d data corrupted", k.ID)
		return nil, sdk.WithStack(sdk.ErrNotFound)
	}
	return &k.ApplicationKey, nil
}

// DeleteKey Delete the given key from the given application
func DeleteKey(db gorp.SqlExecutor, appID int64, keyName string) error {
	_, err := db.Exec("DELETE FROM application_key WHERE application_id = $1 AND name = $2", appID, keyName)
	return sdk.WrapError(err, "Cannot delete key %s", keyName)
}
