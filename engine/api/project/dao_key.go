package project

import (
	"context"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

// InsertKey a new project key in database
func InsertKey(db gorp.SqlExecutor, key *sdk.ProjectKey) error {
	var dbProjKey = dbProjectKey{ProjectKey: *key}
	if err := gorpmapping.InsertAndSign(context.Background(), db, &dbProjKey); err != nil {
		return err
	}
	*key = dbProjKey.ProjectKey
	return nil
}

// UpdateKey a new project key in database.
// This function should be use only for migration purpose and should be removed
func UpdateKey(ctx context.Context, db gorp.SqlExecutor, key *sdk.ProjectKey) error {
	var dbProjKey = dbProjectKey{ProjectKey: *key}
	if err := gorpmapping.UpdateAndSign(ctx, db, &dbProjKey); err != nil {
		return err
	}
	*key = dbProjKey.ProjectKey
	return nil
}

func getAllKeys(db gorp.SqlExecutor, query gorpmapping.Query) ([]sdk.ProjectKey, error) {
	var ctx = context.Background()
	var res []dbProjectKey
	keys := make([]sdk.ProjectKey, 0, len(res))

	if err := gorpmapping.GetAll(ctx, db, query, &res); err != nil {
		return nil, err
	}

	for i := range res {
		isValid, err := gorpmapping.CheckSignature(res[i], res[i].Signature)
		if err != nil {
			return nil, err
		}
		if !isValid {
			log.Error(ctx, "project.getAllKeys> project key %d data corrupted", res[i].ID)
			continue
		}
		keys = append(keys, res[i].ProjectKey)
	}
	return keys, nil
}

// LoadAllKeys load all keys for the given application
func LoadAllKeys(db gorp.SqlExecutor, projectID int64) ([]sdk.ProjectKey, error) {
	query := gorpmapping.NewQuery(`
		SELECT * 
		FROM project_key 
		WHERE project_id = $1 
		AND builtin = false
	`).Args(projectID)

	return getAllKeys(db, query)
}

// LoadAllKeysWithPrivateContent load all keys for the given project
func LoadAllKeysWithPrivateContent(db gorp.SqlExecutor, appID int64) ([]sdk.ProjectKey, error) {
	keys, err := LoadAllKeys(db, appID)
	if err != nil {
		return nil, err
	}

	res := make([]sdk.ProjectKey, 0, len(keys))
	for _, k := range keys {
		x, err := LoadKey(db, k.ID, k.Name)
		if err != nil {
			return nil, err
		}
		res = append(res, *x)
	}

	return res, nil
}

func LoadKey(db gorp.SqlExecutor, id int64, keyName string) (*sdk.ProjectKey, error) {
	query := gorpmapping.NewQuery(`
	SELECT * 
	FROM project_key
	WHERE id = $1 
	AND name = $2
	AND builtin = false 
	`).Args(id, keyName)
	var k dbProjectKey
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
		log.Error(context.Background(), "project.LoadKey> project key %d data corrupted", k.ID)
		return nil, sdk.WithStack(sdk.ErrNotFound)
	}
	return &k.ProjectKey, nil
}

// DeleteProjectKey Delete the given key from the given project
func DeleteProjectKey(db gorp.SqlExecutor, projectID int64, keyName string) error {
	_, err := db.Exec("DELETE FROM project_key WHERE project_id = $1 AND name = $2", projectID, keyName)
	return sdk.WrapError(err, "Cannot delete key %s", keyName)
}

func loadBuildinKey(db gorp.SqlExecutor, projectID int64) (*sdk.ProjectKey, error) {
	query := gorpmapping.NewQuery(`
	SELECT * 
	FROM project_key
	WHERE id = $1 
	AND builtin = true 
	AND name = 'builtin'
	`).Args(projectID)
	var k dbProjectKey
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
		log.Error(context.Background(), "project.LoadKey> project key %d data corrupted", k.ID)
		return nil, sdk.WithStack(sdk.ErrNotFound)
	}
	return &k.ProjectKey, nil
}
