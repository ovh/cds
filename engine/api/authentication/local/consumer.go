package local

import (
	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/authentication"
	"github.com/ovh/cds/sdk"
)

// NewConsumer returns a new local consumer for given data.
func NewConsumer(db gorp.SqlExecutor, userID string) (*sdk.AuthConsumer, error) {
	return newConsumerWithData(db, userID, nil)
}

// NewConsumerWithPassword returns a new local consumer with given password.
func NewConsumerWithPassword(db gorp.SqlExecutor, userID, password string) (*sdk.AuthConsumer, error) {
	// Generate password hash to store in consumer
	hash, err := HashPassword(password)
	if err != nil {
		return nil, err
	}

	return newConsumerWithData(db, userID, map[string]string{
		"hash": string(hash),
	})
}

func newConsumerWithData(db gorp.SqlExecutor, userID string, data map[string]string) (*sdk.AuthConsumer, error) {
	c := sdk.AuthConsumer{
		Name:               string(sdk.ConsumerLocal),
		AuthentifiedUserID: userID,
		Type:               sdk.ConsumerLocal,
		Data: map[string]string{
			"verified": sdk.FalseString,
		},
	}

	for k, v := range data {
		if _, ok := c.Data[k]; !ok {
			c.Data[k] = v
		}
	}

	if err := authentication.InsertConsumer(db, &c); err != nil {
		return nil, err
	}

	return &c, nil
}
