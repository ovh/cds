package authentication

import (
	"context"
	"time"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func getConsumers(ctx context.Context, db gorp.SqlExecutor, q gorpmapping.Query, opts ...LoadConsumerOptionFunc) ([]sdk.AuthConsumer, error) {
	cs := []authConsumer{}

	if err := gorpmapping.GetAll(ctx, db, q, &cs); err != nil {
		return nil, sdk.WrapError(err, "cannot get auth consumers")
	}

	// Check signature of data, if invalid do not return it
	verifiedConsumers := make([]*sdk.AuthConsumer, 0, len(cs))
	for i := range cs {
		isValid, err := gorpmapping.CheckSignature(cs[i], cs[i].Signature)
		if err != nil {
			return nil, err
		}
		if !isValid {
			log.Error("authentication.getConsumers> auth consumer %s data corrupted", cs[i].ID)
			continue
		}
		verifiedConsumers = append(verifiedConsumers, &cs[i].AuthConsumer)
	}

	if len(verifiedConsumers) > 0 {
		for i := range opts {
			if err := opts[i](ctx, db, verifiedConsumers...); err != nil {
				return nil, err
			}
		}
	}

	consumers := make([]sdk.AuthConsumer, len(verifiedConsumers))
	for i := range verifiedConsumers {
		consumers[i] = *verifiedConsumers[i]
	}

	return consumers, nil
}

func getConsumer(ctx context.Context, db gorp.SqlExecutor, q gorpmapping.Query, opts ...LoadConsumerOptionFunc) (*sdk.AuthConsumer, error) {
	var consumer authConsumer

	found, err := gorpmapping.Get(ctx, db, q, &consumer)
	if err != nil {
		return nil, sdk.WrapError(err, "cannot get auth consumer")
	}
	if !found {
		return nil, nil
	}

	isValid, err := gorpmapping.CheckSignature(consumer, consumer.Signature)
	if err != nil {
		return nil, err
	}
	if !isValid {
		log.Error("authentication.getConsumer> auth consumer %s data corrupted", consumer.ID)
		return nil, nil
	}

	ac := consumer.AuthConsumer

	for i := range opts {
		if err := opts[i](ctx, db, &ac); err != nil {
			return nil, err
		}
	}

	return &ac, nil
}

// LoadConsumerByID returns an auth consumer from database.
func LoadConsumerByID(ctx context.Context, db gorp.SqlExecutor, id string, opts ...LoadConsumerOptionFunc) (*sdk.AuthConsumer, error) {
	query := gorpmapping.NewQuery("SELECT * FROM auth_consumer WHERE id = $1").Args(id)
	return getConsumer(ctx, db, query, opts...)
}

// LoadConsumerByTypeAndUserID returns an auth consumer from database for given type and user id.
func LoadConsumerByTypeAndUserID(ctx context.Context, db gorp.SqlExecutor, consumerType sdk.AuthConsumerType, userID string, opts ...LoadConsumerOptionFunc) (*sdk.AuthConsumer, error) {
	query := gorpmapping.NewQuery("SELECT * FROM auth_consumer WHERE type = $1 AND user_id = $2").Args(consumerType, userID)
	return getConsumer(ctx, db, query, opts...)
}

// InsertConsumer in database.
func InsertConsumer(db gorp.SqlExecutor, ac *sdk.AuthConsumer) error {
	ac.ID = sdk.UUID()
	ac.Created = time.Now()
	c := authConsumer{AuthConsumer: *ac}
	if err := gorpmapping.InsertAndSign(db, &c); err != nil {
		return sdk.WrapError(err, "unable to insert auth consumer")
	}
	*ac = c.AuthConsumer
	return nil
}

// UpdateConsumer in database.
func UpdateConsumer(db gorp.SqlExecutor, ac *sdk.AuthConsumer) error {
	if err := gorpmapping.UpdatetAndSign(db, &authConsumer{AuthConsumer: *ac}); err != nil {
		return sdk.WrapError(err, "unable to update auth consumer with id: %s", ac.ID)
	}
	return nil
}

// DeleteConsumerByID removes a auth consumer in database for given id.
func DeleteConsumerByID(db gorp.SqlExecutor, id string) error {
	_, err := db.Exec("DELETE FROM auth_consumer WHERE id = $1", id)
	return sdk.WrapError(err, "unable to delete auth consumer with id %s", id)
}
