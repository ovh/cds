package authentication

import (
	"context"
	"github.com/go-gorp/gorp"
	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
	"github.com/rockbears/log"
	"time"
)

func getConsumers(ctx context.Context, db gorp.SqlExecutor, q gorpmapping.Query) ([]sdk.AuthConsumer, error) {
	cs := []authConsumer{}

	if err := gorpmapping.GetAll(ctx, db, q, &cs); err != nil {
		return nil, sdk.WrapError(err, "cannot get auth consumers")
	}

	// Check signature of data, if invalid do not return it
	verifiedConsumers := make([]sdk.AuthConsumer, 0, len(cs))
	for i := range cs {
		isValid, err := gorpmapping.CheckSignature(cs[i], cs[i].Signature)
		if err != nil {
			return nil, err
		}
		if !isValid {
			log.Error(ctx, "authentication.getConsumers> auth consumer %s data corrupted", cs[i].ID)
			continue
		}
		verifiedConsumers = append(verifiedConsumers, cs[i].AuthConsumer)
	}

	return verifiedConsumers, nil
}

func getConsumer(ctx context.Context, db gorp.SqlExecutor, q gorpmapping.Query) (*sdk.AuthConsumer, error) {
	var consumer authConsumer

	found, err := gorpmapping.Get(ctx, db, q, &consumer)
	if err != nil {
		return nil, sdk.WrapError(err, "cannot get auth consumer")
	}
	if !found {
		return nil, sdk.WithStack(sdk.ErrNotFound)
	}

	isValid, err := gorpmapping.CheckSignature(consumer, consumer.Signature)
	if err != nil {
		return nil, err
	}
	if !isValid {
		log.Error(ctx, "authentication.getConsumer> auth consumer %s data corrupted", consumer.ID)
		return nil, sdk.WithStack(sdk.ErrNotFound)
	}
	c := consumer.AuthConsumer
	c.ValidityPeriods.Sort()
	return &c, nil
}

func insertConsumer(ctx context.Context, db gorpmapper.SqlExecutorWithTx, ac *sdk.AuthConsumer) error {
	// Because we need to create consumers before CDS first start with the init token, the consumer id can be set.
	// In this case we don't want to create a new UUID.
	ac.ValidityPeriods.Sort()
	if ac.ID == "" {
		ac.ID = sdk.UUID()
	}
	ac.Created = time.Now()
	ac.ValidityPeriods.Sort()
	c := authConsumer{AuthConsumer: *ac}
	if err := gorpmapping.InsertAndSign(ctx, db, &c); err != nil {
		return sdk.WrapError(err, "unable to insert auth consumer")
	}
	*ac = c.AuthConsumer
	return nil
}

func updateConsumer(ctx context.Context, db gorpmapper.SqlExecutorWithTx, ac *sdk.AuthConsumer) error {
	ac.ValidityPeriods.Sort()
	c := authConsumer{AuthConsumer: *ac}
	if err := gorpmapping.UpdateAndSign(ctx, db, &c); err != nil {
		return sdk.WrapError(err, "unable to update auth consumer with id: %s", ac.ID)
	}
	*ac = c.AuthConsumer
	return nil
}

// DeleteConsumerByID removes a auth consumer in database for given id.
func DeleteConsumerByID(db gorp.SqlExecutor, id string) error {
	_, err := db.Exec("DELETE FROM auth_consumer WHERE id = $1", id)
	return sdk.WrapError(err, "unable to delete auth consumer with id %s", id)
}

// UpdateConsumerLastAuthentication updates only the column last_authentication
func UpdateConsumerLastAuthentication(ctx context.Context, db gorp.SqlExecutor, ac *sdk.AuthConsumer) error {
	c := authConsumer{AuthConsumer: *ac}
	err := gorpmapping.UpdateColumns(db, &c, func(cm *gorp.ColumnMap) bool {
		return cm.ColumnName == "last_authentication"
	})
	*ac = c.AuthConsumer
	return sdk.WrapError(err, "unable to update last_authentication auth consumer with id %s", ac.ID)
}

func loadConsumerByID(ctx context.Context, db gorp.SqlExecutor, consumerID string) (*sdk.AuthConsumer, error) {
	query := gorpmapping.NewQuery("SELECT * from auth_consumer WHERE id = $1").Args(consumerID)
	return getConsumer(ctx, db, query)
}
