package authentication

import (
	"context"
	"time"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func getConsumers(ctx context.Context, db gorp.SqlExecutor, q gorpmapping.Query, opts ...LoadConsumerOptionFunc) (sdk.AuthConsumers, error) {
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
			log.Error(ctx, "authentication.getConsumers> auth consumer %s data corrupted", cs[i].ID)
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

	ac := consumer.AuthConsumer
	for i := range opts {
		if err := opts[i](ctx, db, &ac); err != nil {
			return nil, err
		}
	}

	return &ac, nil
}

// LoadConsumersByUserID returns auth consumers from database for given user id.
func LoadConsumersByUserID(ctx context.Context, db gorp.SqlExecutor, id string, opts ...LoadConsumerOptionFunc) (sdk.AuthConsumers, error) {
	query := gorpmapping.NewQuery("SELECT * FROM auth_consumer WHERE user_id = $1 ORDER BY created ASC").Args(id)
	return getConsumers(ctx, db, query, opts...)
}

// LoadConsumersByGroupID returns all consumers from database that refer to given group id.
func LoadConsumersByGroupID(ctx context.Context, db gorp.SqlExecutor, groupID int64, opts ...LoadConsumerOptionFunc) (sdk.AuthConsumers, error) {
	query := gorpmapping.NewQuery("SELECT * FROM auth_consumer WHERE group_ids @> $1 OR invalid_group_ids @> $1 ORDER BY created ASC").Args(groupID)
	return getConsumers(ctx, db, query, opts...)
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

// LoadConsumerByTypeAndUserExternalID returns an auth consumer from database for given type and user id.
func LoadConsumerByTypeAndUserExternalID(ctx context.Context, db gorp.SqlExecutor, consumerType sdk.AuthConsumerType, userExternalID string, opts ...LoadConsumerOptionFunc) (*sdk.AuthConsumer, error) {
	query := gorpmapping.NewQuery("SELECT * FROM auth_consumer WHERE type = $1 AND (data->>'external_id')::text = $2").Args(consumerType, userExternalID)
	return getConsumer(ctx, db, query, opts...)
}

// InsertConsumer in database.
func InsertConsumer(ctx context.Context, db gorpmapper.SqlExecutorWithTx, ac *sdk.AuthConsumer) error {
	// Because we need to create consumers before CDS first start with the init token, the consumer id can be set.
	// In this case we don't want to create a new UUID.
	if ac.ID == "" {
		ac.ID = sdk.UUID()
	}
	ac.Created = time.Now()
	c := authConsumer{AuthConsumer: *ac}
	if err := gorpmapping.InsertAndSign(ctx, db, &c); err != nil {
		return sdk.WrapError(err, "unable to insert auth consumer")
	}
	*ac = c.AuthConsumer
	return nil
}

// UpdateConsumer in database.
func UpdateConsumer(ctx context.Context, db gorpmapper.SqlExecutorWithTx, ac *sdk.AuthConsumer) error {
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
