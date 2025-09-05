package migrate

import (
	"context"
	"time"

	"github.com/go-gorp/gorp"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/sdk"
)

func MigrateProjectRunRetention(ctx context.Context, db *gorp.DbMap, store cache.Store, nbCount, nbDays int64) error {
	pKeys, err := project.LoadAllProjectKeys(ctx, db, store)
	if err != nil {
		return err
	}

	for _, k := range pKeys {
		if err := addProjectRunRetention(ctx, db, store, k, nbCount, nbDays); err != nil {
			return err
		}
	}
	return nil
}

func addProjectRunRetention(ctx context.Context, db *gorp.DbMap, store cache.Store, pkey string, nbCount, nbDays int64) error {
	lockKey := cache.Key("migrate", "project", "run", "retention", pkey)
	b, err := store.Lock(lockKey, 2*time.Second, 250, 1)
	if err != nil {
		return err
	}
	if !b {
		return nil
	}
	defer store.Unlock(lockKey)

	_, err = project.LoadRunRetentionByProjectKey(ctx, db, pkey)
	if !sdk.ErrorIs(err, sdk.ErrNotFound) {
		return err
	}

	projectRunRetention := sdk.ProjectRunRetention{
		ProjectKey: pkey,
		Retentions: sdk.Retentions{
			DefaultRetention: sdk.RetentionRule{
				DurationInDays: nbDays,
				Count:          nbCount,
			},
		},
	}
	tx, err := db.Begin()
	if err != nil {
		return sdk.WithStack(err)
	}
	if err := project.InsertRunRetention(ctx, tx, &projectRunRetention); err != nil {
		return err
	}

	return sdk.WithStack(tx.Commit())
}
