package authentication

import (
	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/sdk"
)

func NewConsumerWorker(db gorp.SqlExecutor, name string, hatcherySrv *sdk.Service, hatcheryConsumer *sdk.AuthConsumer, groupIDs []int64) (*sdk.AuthConsumer, error) {
	c := sdk.AuthConsumer{
		Name:               name,
		AuthentifiedUserID: hatcherySrv.Maintainer.ID,
		ParentID:           &hatcheryConsumer.ID,
		Type:               sdk.ConsumerBuiltin,
		Data:               map[string]string{},
		GroupIDs:           groupIDs,
		Scopes:             []string{sdk.AccessTokenScopeWorker, sdk.AccessTokenScopeRunExecution},
	}

	if err := InsertConsumer(db, &c); err != nil {
		return nil, err
	}

	return &c, nil
}

// NewConsumerBuiltin returns a new builtin consumer for given data.
func NewConsumerBuiltin(db gorp.SqlExecutor, name, description, userID string, groupIDs []int64, scopes []string) (*sdk.AuthConsumer, error) {
	c := sdk.AuthConsumer{
		Name:               name,
		Description:        description,
		AuthentifiedUserID: userID,
		Type:               sdk.ConsumerBuiltin,
		Data:               map[string]string{},
		GroupIDs:           groupIDs,
		Scopes:             scopes,
	}

	if err := InsertConsumer(db, &c); err != nil {
		return nil, err
	}

	return &c, nil
}

// NewConsumerLocal returns a new local consumer for given data.
func NewConsumerLocal(db gorp.SqlExecutor, userID string, hash []byte) (*sdk.AuthConsumer, error) {
	c := sdk.AuthConsumer{
		Name:               string(sdk.ConsumerLocal),
		AuthentifiedUserID: userID,
		Type:               sdk.ConsumerLocal,
		Data: map[string]string{
			"hash":     string(hash),
			"verified": sdk.FalseString,
		},
	}

	if err := InsertConsumer(db, &c); err != nil {
		return nil, err
	}

	return &c, nil
}
