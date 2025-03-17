package project

import (
	"context"
	"time"

	"github.com/go-gorp/gorp"
	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
)

func InsertRepositoryHook(ctx context.Context, db gorpmapper.SqlExecutorWithTx, h *sdk.ProjectRepositoryHook) error {
	h.Created = time.Now()

	dbData := &dbProjectRepositoryHook{ProjectRepositoryHook: *h}
	if err := gorpmapping.Insert(db, dbData); err != nil {
		return err
	}
	*h = dbData.ProjectRepositoryHook
	return nil
}

func DeleteRepsitoryHook(db gorpmapper.SqlExecutorWithTx, projectKey string, hookUUID string) error {
	_, err := db.Exec("DELETE FROM project_repository_hook WHERE id = $1 AND project_key = $2", hookUUID, projectKey)
	return sdk.WrapError(err, "cannot delete project_repository_hook %s / %s", projectKey, hookUUID)
}

func getRepositoryHook(ctx context.Context, db gorp.SqlExecutor, query gorpmapping.Query) (*sdk.ProjectRepositoryHook, error) {
	var res dbProjectRepositoryHook
	found, err := gorpmapping.Get(ctx, db, query, &res)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, sdk.NewErrorFrom(sdk.ErrNotFound, "unable to found repository hook")
	}
	return &res.ProjectRepositoryHook, nil
}

func getAllRepositoryHooks(ctx context.Context, db gorp.SqlExecutor, query gorpmapping.Query) ([]sdk.ProjectRepositoryHook, error) {
	var res []dbProjectRepositoryHook
	if err := gorpmapping.GetAll(ctx, db, query, &res); err != nil {
		return nil, err
	}

	hooks := make([]sdk.ProjectRepositoryHook, 0, len(res))
	for _, r := range res {
		hooks = append(hooks, r.ProjectRepositoryHook)
	}

	return hooks, nil
}

func LoadAllRepositoryHooks(ctx context.Context, db gorp.SqlExecutor, projKey string) ([]sdk.ProjectRepositoryHook, error) {
	query := gorpmapping.NewQuery(`SELECT project_repository_hook.* FROM project_repository_hook WHERE project_key = $1`).Args(projKey)
	return getAllRepositoryHooks(ctx, db, query)
}

func LoadRepositoryHookByID(ctx context.Context, db gorp.SqlExecutor, projKey string, uuid string) (*sdk.ProjectRepositoryHook, error) {
	query := gorpmapping.NewQuery(`SELECT project_repository_hook.* FROM project_repository_hook WHERE project_key = $1 AND id = $2`).Args(projKey, uuid)
	return getRepositoryHook(ctx, db, query)
}
