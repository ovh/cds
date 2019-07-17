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
		Scopes: []sdk.AuthConsumerScope{
			sdk.AuthConsumerScopeWorker,
			sdk.AuthConsumerScopeWorkerModel,
			sdk.AuthConsumerScopeRun,
			sdk.AuthConsumerScopeRunExecution,
		},
	}

	if err := InsertConsumer(db, &c); err != nil {
		return nil, err
	}

	return &c, nil
}

// NewConsumerExternal returns a new local consumer for given data.
func NewConsumerExternal(db gorp.SqlExecutor, userID string, consumerType sdk.AuthConsumerType, userInfo sdk.AuthDriverUserInfo) (*sdk.AuthConsumer, error) {
	c := sdk.AuthConsumer{
		Name:               string(consumerType),
		AuthentifiedUserID: userID,
		Type:               consumerType,
		Data: map[string]string{
			"external_id": userInfo.ExternalID,
			"fullname":    userInfo.Fullname,
			"username":    userInfo.Username,
			"email":       userInfo.Email,
		},
	}

	if err := InsertConsumer(db, &c); err != nil {
		return nil, err
	}

	return &c, nil
}
