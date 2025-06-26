package project

import (
	"context"
	"time"

	"github.com/go-gorp/gorp"
	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
)

func InsertRunRetention(ctx context.Context, db gorpmapper.SqlExecutorWithTx, retention *sdk.ProjectRunRetention) error {
	retention.LastModified = time.Now()
	retention.ID = sdk.UUID()
	dbData := dbProjectRunRetention{ProjectRunRetention: *retention}
	if err := gorpmapping.Insert(db, &dbData); err != nil {
		return err
	}
	*retention = dbData.ProjectRunRetention
	return nil
}

func UpdateRunRetention(ctx context.Context, db gorpmapper.SqlExecutorWithTx, retention *sdk.ProjectRunRetention) error {
	retention.LastModified = time.Now()
	dbData := dbProjectRunRetention{ProjectRunRetention: *retention}
	if err := gorpmapping.Update(db, &dbData); err != nil {
		return err
	}
	*retention = dbData.ProjectRunRetention
	return nil
}

func getRunRetention(ctx context.Context, db gorp.SqlExecutor, query gorpmapping.Query) (*sdk.ProjectRunRetention, error) {
	var res dbProjectRunRetention
	found, err := gorpmapping.Get(ctx, db, query, &res)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, sdk.WithStack(sdk.ErrNotFound)
	}
	return &res.ProjectRunRetention, nil
}

func LoadRunRetentionByProjectKey(ctx context.Context, db gorp.SqlExecutor, projKey string) (*sdk.ProjectRunRetention, error) {
	query := gorpmapping.NewQuery(`SELECT project_run_retention.* FROM project_run_retention WHERE project_key = $1`).Args(projKey)
	return getRunRetention(ctx, db, query)
}
