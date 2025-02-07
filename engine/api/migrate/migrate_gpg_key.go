package migrate

import (
	"context"
	"strings"

	"github.com/go-gorp/gorp"
	"github.com/ovh/cds/engine/api/keys"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
	"github.com/rockbears/log"
)

func MigrateAllProjectGPGKeys(ctx context.Context, db *gorp.DbMap, store cache.Store) error {
	projects, err := project.LoadAll(ctx, db, store)
	if err != nil {
		return err
	}
	for _, p := range projects {
		tx, err := db.Begin()
		if err != nil {
			return sdk.WithStack(err)
		}
		if err := MigrateProjectGPGKeys(ctx, tx, store, p.ID); err != nil {
			_ = tx.Rollback()
			return err
		}
		if err := tx.Commit(); err != nil {
			return sdk.WithStack(err)
		}
	}
	return nil
}

func MigrateProjectGPGKeys(ctx context.Context, db gorpmapper.SqlExecutorWithTx, store cache.Store, projectID int64) error {
	pkeys, err := project.LoadAllKeysWithPrivateContent(ctx, db, projectID)
	if err != nil {
		return err
	}
	for i := range pkeys {
		k := &pkeys[i]
		if k.Type != sdk.KeyTypePGP {
			continue
		}
		pgpEntity, err := keys.GetOpenPGPEntity(strings.NewReader(k.Private))
		if err != nil {
			return err
		}
		log.Info(ctx, "handling key %d %s %s %s", k.ID, k.Name, pgpEntity.PrimaryKey.KeyIdShortString(), pgpEntity.PrimaryKey.KeyIdString())
		k.LongKeyID = pgpEntity.PrimaryKey.KeyIdString()
		if err := project.UpdateKey(db, k); err != nil {
			return err
		}
	}
	return nil
}
