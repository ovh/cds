package migrate

import (
	"context"
	"time"

	"github.com/go-gorp/gorp"
	"github.com/rockbears/log"

	"github.com/ovh/cds/engine/api/authentication"
	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/sdk"
)

func MigrateConsumers(ctx context.Context, db *gorp.DbMap, c cache.Store) error {
	b, err := c.Lock(cache.Key("migrate", "consumer", "lock"), 300*time.Second, -1, -1)
	if err != nil {
		log.ErrorWithStackTrace(ctx, sdk.WithStack(err))
		return err
	}
	if !b {
		log.Info(ctx, "MigrateConsumers> Lock is already taken")
		return nil
	}
	log.Info(ctx, "MigrateConsumers> Lock took")

	oldConsumers, err := authentication.LoadOldConsumers(ctx, db)
	if err != nil {
		return err
	}

	for _, oldC := range oldConsumers {
		// Check if consumer has been already migrated
		consumerExist, err := authentication.LoadConsumerByID(ctx, db, oldC.ID)
		if err == nil && consumerExist != nil {
			continue
		}

		newConsumer := sdk.AuthConsumer{
			ID:                 oldC.ID,
			Type:               oldC.Type,
			Name:               oldC.Name,
			Description:        oldC.Description,
			ParentID:           oldC.ParentID,
			Created:            oldC.Created,
			DeprecatedIssuedAt: oldC.DeprecatedIssuedAt,
			Disabled:           oldC.Disabled,
			LastAuthentication: oldC.LastAuthentication,
			ValidityPeriods:    oldC.ValidityPeriods,
			Warnings:           oldC.Warnings,
			AuthConsumerUser: &sdk.AuthConsumerUser{
				ID:                           sdk.UUID(),
				AuthConsumerID:               oldC.ID,
				ScopeDetails:                 oldC.ScopeDetails,
				AuthentifiedUserID:           oldC.AuthentifiedUserID,
				Data:                         oldC.Data,
				GroupIDs:                     oldC.GroupIDs,
				InvalidGroupIDs:              oldC.InvalidGroupIDs,
				ServiceIgnoreJobWithNoRegion: oldC.ServiceIgnoreJobWithNoRegion,
				ServiceName:                  oldC.ServiceName,
				ServiceType:                  oldC.ServiceType,
				ServiceRegion:                oldC.ServiceRegion,
			},
		}
		tx, err := db.Begin()
		if err != nil {
			return sdk.WithStack(err)
		}
		if err := authentication.InsertConsumer(ctx, tx, &newConsumer); err != nil {
			_ = tx.Rollback()
			return err
		}
		if err := tx.Commit(); err != nil {
			_ = tx.Rollback()
			return sdk.WithStack(err)
		}
	}
	return nil
}
