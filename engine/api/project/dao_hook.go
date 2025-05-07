package project

import (
	"context"
	"time"

	"github.com/go-gorp/gorp"
	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
)

func InsertWebHook(ctx context.Context, db gorpmapper.SqlExecutorWithTx, h *sdk.ProjectWebHook) error {
	h.Created = time.Now()

	dbData := &dbProjectWebHook{ProjectWebHook: *h}
	if err := gorpmapping.Insert(db, dbData); err != nil {
		return err
	}
	*h = dbData.ProjectWebHook
	return nil
}

func DeleteWebHook(db gorpmapper.SqlExecutorWithTx, projectKey string, hookUUID string) error {
	_, err := db.Exec("DELETE FROM project_webhook WHERE id = $1 AND project_key = $2", hookUUID, projectKey)
	return sdk.WrapError(err, "cannot delete project_webhook %s / %s", projectKey, hookUUID)
}

func getWebHook(ctx context.Context, db gorp.SqlExecutor, query gorpmapping.Query) (*sdk.ProjectWebHook, error) {
	var res dbProjectWebHook
	found, err := gorpmapping.Get(ctx, db, query, &res)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, sdk.NewErrorFrom(sdk.ErrNotFound, "unable to found webhook")
	}
	return &res.ProjectWebHook, nil
}

func getAllWebHook(ctx context.Context, db gorp.SqlExecutor, query gorpmapping.Query) ([]sdk.ProjectWebHook, error) {
	var res []dbProjectWebHook
	if err := gorpmapping.GetAll(ctx, db, query, &res); err != nil {
		return nil, err
	}

	hooks := make([]sdk.ProjectWebHook, 0, len(res))
	for _, r := range res {
		hooks = append(hooks, r.ProjectWebHook)
	}

	return hooks, nil
}

func LoadAllWebHooks(ctx context.Context, db gorp.SqlExecutor, projKey string) ([]sdk.ProjectWebHook, error) {
	query := gorpmapping.NewQuery(`SELECT project_webhook.* FROM project_webhook WHERE project_key = $1`).Args(projKey)
	return getAllWebHook(ctx, db, query)
}

func LoadWebHookByID(ctx context.Context, db gorp.SqlExecutor, projKey string, uuid string) (*sdk.ProjectWebHook, error) {
	query := gorpmapping.NewQuery(`SELECT project_webhook.* FROM project_webhook WHERE project_key = $1 AND id = $2`).Args(projKey, uuid)
	return getWebHook(ctx, db, query)
}
