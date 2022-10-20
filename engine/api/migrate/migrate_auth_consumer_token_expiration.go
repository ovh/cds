package migrate

import (
	"context"
	"database/sql"
	"time"

	"github.com/go-gorp/gorp"
	"github.com/ovh/cds/engine/api/authentication"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
	"github.com/rockbears/log"
)

func AuthConsumerTokenExpiration(ctx context.Context, dbFunc func() *gorp.DbMap, duration time.Duration) error {
	log.Info(ctx, "starting auth consumer token expiration migration")
	defer log.Info(ctx, "ending authconsumer token expiration migration")

	var authConsumerIDs []string
	_, err := dbFunc().Select(&authConsumerIDs, "select id from auth_consumer where validity_periods is null")
	if err != nil && err != sql.ErrNoRows {
		return sdk.WrapError(err, "unable to load auth_consumer.id")
	}

	for _, id := range authConsumerIDs {
		tx, err := dbFunc().Begin()
		if err != nil {
			ctx := sdk.ContextWithStacktrace(ctx, err)
			log.Error(ctx, "unable to start transaction")
			continue
		}
		if err := authConsumerTokenExpirationPerID(ctx, tx, id, duration); err != nil {
			ctx := sdk.ContextWithStacktrace(ctx, err)
			log.Error(ctx, "%v", err)
			tx.Rollback() // nolint
			continue
		}
		if err := tx.Commit(); err != nil {
			ctx := sdk.ContextWithStacktrace(ctx, err)
			log.Error(ctx, "unable to commit transaction")
			continue
		}
	}

	return nil
}

func authConsumerTokenExpirationPerID(ctx context.Context, tx gorpmapper.SqlExecutorWithTx, id string, duration time.Duration) error {
	// Lock the row
	id, err := tx.SelectStr("select id from auth_consumer where id=$1 and validity_periods is null for update skip locked", id)
	if err == sql.ErrNoRows {
		return nil
	}
	if err != nil {
		return err
	}
	if id == "" {
		return nil
	}

	log.Info(ctx, "migrating consumer %s", id)

	// Load the consumer
	consumer, err := authentication.LoadUserConsumerByID(ctx, tx, id)
	if err != nil {
		return err
	}

	if len(consumer.ValidityPeriods) > 0 {
		return nil
	}

	consumer.ValidityPeriods = sdk.NewAuthConsumerValidityPeriod(consumer.DeprecatedIssuedAt, duration)
	log.Info(ctx, "consumer %q IAT=%v Expiration=%v", consumer.ID, consumer.ValidityPeriods.Latest().IssuedAt, consumer.ValidityPeriods.Latest().IssuedAt.Add(consumer.ValidityPeriods.Latest().Duration))

	return authentication.UpdateUserConsumer(ctx, tx, consumer)
}
