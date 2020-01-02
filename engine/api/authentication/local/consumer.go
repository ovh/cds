package local

import (
	"context"
	"time"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/authentication"
	"github.com/ovh/cds/sdk"
)

// NewConsumer returns a new local consumer for given data.
func NewConsumer(ctx context.Context, db gorp.SqlExecutor, userID string) (*sdk.AuthConsumer, error) {
	return newConsumerWithData(ctx, db, userID, nil)
}

// NewConsumerWithHash returns a new local consumer with given hash.
func NewConsumerWithHash(ctx context.Context, db gorp.SqlExecutor, userID, hash string) (*sdk.AuthConsumer, error) {
	return newConsumerWithData(ctx, db, userID, map[string]string{
		"hash": hash,
	})
}

func newConsumerWithData(ctx context.Context, db gorp.SqlExecutor, userID string, data map[string]string) (*sdk.AuthConsumer, error) {
	c := sdk.AuthConsumer{
		Name:               string(sdk.ConsumerLocal),
		AuthentifiedUserID: userID,
		Type:               sdk.ConsumerLocal,
		Data: map[string]string{
			"verified": sdk.FalseString,
		},
		IssuedAt: time.Now(),
	}

	for k, v := range data {
		if _, ok := c.Data[k]; !ok {
			c.Data[k] = v
		}
	}

	if err := authentication.InsertConsumer(ctx, db, &c); err != nil {
		return nil, sdk.WrapError(err, "unable to insert consumer for user %s", userID)
	}

	return &c, nil
}
