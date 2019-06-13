package authentication

import (
	"context"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/user"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

// LoadConsumerOptionFunc for auth consumer.
type LoadConsumerOptionFunc func(context.Context, gorp.SqlExecutor, ...*sdk.AuthConsumer) error

// LoadConsumerOptions provides all options on auth consumer loads functions.
var LoadConsumerOptions = struct {
	WithAuthentifiedUser LoadConsumerOptionFunc
}{
	WithAuthentifiedUser: loadAuthentifiedUser,
}

func loadAuthentifiedUser(ctx context.Context, db gorp.SqlExecutor, cs ...*sdk.AuthConsumer) error {
	// Load all users for given access tokens
	users, err := user.LoadAllByIDs(ctx, db, sdk.AuthConsumersToAuthentifiedUserIDs(cs), user.LoadOptions.WithDeprecatedUser)
	if err != nil {
		return err
	}

	log.Debug("loadAuthentifiedUser> users: %v", users)

	mUsers := make(map[string]sdk.AuthentifiedUser)
	for i := range users {
		mUsers[users[i].ID] = users[i]
	}

	for i := range cs {
		if user, ok := mUsers[cs[i].AuthentifiedUserID]; ok {
			cs[i].AuthentifiedUser = &user
		}
	}

	return nil
}
