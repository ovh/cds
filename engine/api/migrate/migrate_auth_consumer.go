package migrate

import (
	"context"
	"github.com/ovh/cds/engine/api/authentication"
	"github.com/ovh/cds/sdk"

	"github.com/go-gorp/gorp"
)

func MigrateConsumers(ctx context.Context, db *gorp.DbMap) error {

	oldConsumers, err := authentication.LoadOldConsumers(ctx, db)
	if err != nil {
		return err
	}

	for _, oldC := range oldConsumers {
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
