package project

import (
	"context"

	"github.com/go-gorp/gorp"
	"github.com/lib/pq"
	"github.com/rockbears/log"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
)

// InsertKey a new project key in database
func InsertKey(db gorpmapper.SqlExecutorWithTx, key *sdk.ProjectKey) error {
	var dbProjKey = dbProjectKey{ProjectKey: *key}
	if err := gorpmapping.InsertAndSign(context.Background(), db, &dbProjKey); err != nil {
		return err
	}
	*key = dbProjKey.ProjectKey
	return nil
}

func getAllKeys(ctx context.Context, db gorp.SqlExecutor, query gorpmapping.Query) ([]sdk.ProjectKey, error) {
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

// LoadAllKeys load all keys for the given project
func LoadAllKeys(ctx context.Context, db gorp.SqlExecutor, projectID int64) ([]sdk.ProjectKey, error) {
	query := gorpmapping.NewQuery(`
		SELECT *
		FROM project_key
		WHERE project_id = $1
		AND builtin = false
	`).Args(projectID)

	return getAllKeys(ctx, db, query)
}

// LoadAllKeysWithPrivateContent load all keys for the given project
func LoadAllKeysWithPrivateContent(ctx context.Context, db gorp.SqlExecutor, appID int64) ([]sdk.ProjectKey, error) {
	keys, err := LoadAllKeys(ctx, db, appID)
	if err != nil {
		return nil, err
	}

	res := make([]sdk.ProjectKey, 0, len(keys))
	for _, k := range keys {
		x, err := LoadKey(ctx, db, k.ID, k.Name)
		if err != nil {
			return nil, err
		}
		res = append(res, *x)
	}

	return res, nil
}

func LoadKey(ctx context.Context, db gorp.SqlExecutor, id int64, keyName string) (*sdk.ProjectKey, error) {
	query := gorpmapping.NewQuery(`
	SELECT *
	FROM project_key
	WHERE id = $1
	AND name = $2
	AND builtin = false
	`).Args(id, keyName)
	var k dbProjectKey
	found, err := gorpmapping.Get(ctx, db, query, &k, gorpmapping.GetOptions.WithDecryption)
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
		log.Error(ctx, "project.LoadKey> project key %d data corrupted", k.ID)
		return nil, sdk.WithStack(sdk.ErrNotFound)
	}
	return &k.ProjectKey, nil
}

// DeleteProjectKey Delete the given key from the given project
func DeleteProjectKey(db gorp.SqlExecutor, projectID int64, keyName string) error {
	_, err := db.Exec("DELETE FROM project_key WHERE project_id = $1 AND name = $2", projectID, keyName)
	return sdk.WrapError(err, "Cannot delete key %s", keyName)
}

func loadBuiltinKey(ctx context.Context, db gorp.SqlExecutor, projectID int64) (*sdk.ProjectKey, error) {
	query := gorpmapping.NewQuery(`
	SELECT *
	FROM project_key
	WHERE project_id = $1
	AND builtin = true
	AND name = 'builtin'
	`).Args(projectID)
	var k dbProjectKey
	found, err := gorpmapping.Get(ctx, db, query, &k, gorpmapping.GetOptions.WithDecryption)
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
		log.Error(ctx, "project.LoadKey> project key %d data corrupted", k.ID)
		return nil, sdk.WithStack(sdk.ErrNotFound)
	}
	return &k.ProjectKey, nil
}

func LoadAllKeysForProjectsWithDecryption(ctx context.Context, db gorp.SqlExecutor, projIDs []int64) (map[int64][]sdk.ProjectKey, error) {
	return loadAllKeysForProjects(ctx, db, projIDs, gorpmapping.GetOptions.WithDecryption)
}

func loadAllKeysForProjects(ctx context.Context, db gorp.SqlExecutor, appsID []int64, opts ...gorpmapping.GetOptionFunc) (map[int64][]sdk.ProjectKey, error) {
	var res []dbProjectKey
	query := gorpmapping.NewQuery(`
		SELECT *
		FROM project_key
		WHERE project_id = ANY($1)
		AND builtin = false
		ORDER BY project_id
	`).Args(pq.Int64Array(appsID))
	if err := gorpmapping.GetAll(ctx, db, query, &res, opts...); err != nil {
		return nil, err
	}

	projsKeys := make(map[int64][]sdk.ProjectKey)

	for i := range res {
		dbProjKey := res[i]
		isValid, err := gorpmapping.CheckSignature(dbProjKey, dbProjKey.Signature)
		if err != nil {
			return nil, err
		}
		if !isValid {
			log.Error(ctx, "project.loadAllKeysForProjects> project key id %d data corrupted", dbProjKey.ID)
			continue
		}
		if _, ok := projsKeys[dbProjKey.ProjectID]; !ok {
			projsKeys[dbProjKey.ProjectID] = make([]sdk.ProjectKey, 0)
		}
		projsKeys[dbProjKey.ProjectID] = append(projsKeys[dbProjKey.ProjectID], dbProjKey.ProjectKey)
	}
	return projsKeys, nil
}
