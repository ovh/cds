package authentication

import (
	"context"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/sdk"
)

func getConsumers(ctx context.Context, db gorp.SqlExecutor, q gorpmapping.Query, opts ...LoadConsumerOptionFunc) ([]sdk.AuthConsumer, error) {
	pConsumers := []*sdk.AuthConsumer{}

	if err := gorpmapping.GetAll(ctx, db, q, &pConsumers); err != nil {
		return nil, sdk.WrapError(err, "cannot get auth consumers")
	}
	if len(pConsumers) > 0 {
		for i := range opts {
			if err := opts[i](ctx, db, pConsumers...); err != nil {
				return nil, err
			}
		}
	}

	consumers := make([]sdk.AuthConsumer, len(pConsumers))
	for i := range pConsumers {
		consumers[i] = *pConsumers[i]
	}

	return consumers, nil
}

func getConsumer(ctx context.Context, db gorp.SqlExecutor, q gorpmapping.Query, opts ...LoadConsumerOptionFunc) (*sdk.AuthConsumer, error) {
	var consumer sdk.AuthConsumer

	found, err := gorpmapping.Get(ctx, db, q, &consumer)
	if err != nil {
		return nil, sdk.WrapError(err, "cannot get auth consumer")
	}
	if !found {
		return nil, nil
	}

	for i := range opts {
		if err := opts[i](ctx, db, &consumer); err != nil {
			return nil, err
		}
	}

	return &consumer, nil
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
func InsertConsumer(db gorp.SqlExecutor, c *sdk.AuthConsumer) error {
	if err := gorpmapping.Insert(db, c); err != nil {
		return sdk.WrapError(err, "unable to insert auth consumer")
	}
	return nil
}

// UpdateConsumer in database.
func UpdateConsumer(db gorp.SqlExecutor, c *sdk.AuthConsumer) error {
	if err := gorpmapping.Update(db, c); err != nil {
		return sdk.WrapError(err, "unable to update auth consumer with id: %s", c.ID)
	}
	return nil
}

// DeleteConsumerByID removes a auth consumer in database for given id.
func DeleteConsumerByID(db gorp.SqlExecutor, id string) error {
	_, err := db.Exec("DELETE FROM auth_consumer WHERE id = $1", id)
	return sdk.WrapError(err, "unable to delete auth consumer with id %s", id)
}
