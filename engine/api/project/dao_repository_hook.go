package project

import (
	"context"
	"time"

	"github.com/go-gorp/gorp"
	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
)

func InsertRepositoryHook(ctx context.Context, db gorpmapper.SqlExecutorWithTx, h *sdk.ProjectWebHook) error {
	h.Created = time.Now()

	dbData := &dbProjectRepositoryHook{ProjectWebHook: *h}
	if err := gorpmapping.Insert(db, dbData); err != nil {
		return err
	}
	*h = dbData.ProjectWebHook
	return nil
}

func DeleteRepsitoryHook(db gorpmapper.SqlExecutorWithTx, projectKey string, hookUUID string) error {
	_, err := db.Exec("DELETE FROM project_repository_hook WHERE id = $1 AND project_key = $2", hookUUID, projectKey)
	return sdk.WrapError(err, "cannot delete project_repository_hook %s / %s", projectKey, hookUUID)
}

func getRepositoryHook(ctx context.Context, db gorp.SqlExecutor, query gorpmapping.Query) (*sdk.ProjectWebHook, error) {
	var res dbProjectRepositoryHook
	found, err := gorpmapping.Get(ctx, db, query, &res)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, sdk.NewErrorFrom(sdk.ErrNotFound, "unable to found repository hook")
	}
	return &res.ProjectWebHook, nil
}

func getAllRepositoryHooks(ctx context.Context, db gorp.SqlExecutor, query gorpmapping.Query) ([]sdk.ProjectWebHook, error) {
	var res []dbProjectRepositoryHook
	if err := gorpmapping.GetAll(ctx, db, query, &res); err != nil {
		return nil, err
	}

	hooks := make([]sdk.ProjectWebHook, 0, len(res))
	for _, r := range res {
		hooks = append(hooks, r.ProjectWebHook)
	}

	return hooks, nil
}

func LoadAllRepositoryHooks(ctx context.Context, db gorp.SqlExecutor, projKey string) ([]sdk.ProjectWebHook, error) {
	query := gorpmapping.NewQuery(`SELECT project_repository_hook.* FROM project_repository_hook WHERE project_key = $1`).Args(projKey)
	return getAllRepositoryHooks(ctx, db, query)
}

func LoadRepositoryHookByID(ctx context.Context, db gorp.SqlExecutor, projKey string, uuid string) (*sdk.ProjectWebHook, error) {
	query := gorpmapping.NewQuery(`SELECT project_repository_hook.* FROM project_repository_hook WHERE project_key = $1 AND id = $2`).Args(projKey, uuid)
	return getRepositoryHook(ctx, db, query)
}
