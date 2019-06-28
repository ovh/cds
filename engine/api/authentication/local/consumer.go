package local

import (
	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/authentication"
	"github.com/ovh/cds/sdk"
)

// NewConsumer returns a new local consumer for given data.
func NewConsumer(db gorp.SqlExecutor, userID, password string) (*sdk.AuthConsumer, error) {
	// Generate password hash to store in consumer
	hash, err := HashPassword(password)
	if err != nil {
		return nil, err
	}

	c := sdk.AuthConsumer{
		Name:               string(sdk.ConsumerLocal),
		AuthentifiedUserID: userID,
		Type:               sdk.ConsumerLocal,
		Data: map[string]string{
			"hash":     string(hash),
			"verified": sdk.FalseString,
		},
	}

	if err := authentication.InsertConsumer(db, &c); err != nil {
		return nil, err
	}

	return &c, nil
}
