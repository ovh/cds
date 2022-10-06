package authentication

import (
	"context"

	"github.com/go-gorp/gorp"
	"github.com/rockbears/log"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
)

func loadConsumerUser(ctx context.Context, db gorp.SqlExecutor, ac *sdk.AuthConsumer) error {
	query := gorpmapping.NewQuery("SELECT * FROM auth_consumer_user WHERE auth_consumer_id = $1").Args(ac.ID)
	var dbAuthConsumerUser authConsumerUser
	found, err := gorpmapping.Get(ctx, db, query, &dbAuthConsumerUser)
	if err != nil {
		return sdk.WrapError(err, "cannot get auth consumer user")
	}
	if !found {
		return nil
	}

	isValid, err := gorpmapping.CheckSignature(dbAuthConsumerUser, dbAuthConsumerUser.Signature)
	if err != nil {
		return err
	}
	if !isValid {
		log.Error(ctx, "authentication.loadConsumerUser> auth consumer user from au consumer %s data corrupted", ac.ID)
		return sdk.WithStack(sdk.ErrNotFound)
	}
	ac.AuthConsumerUser = &dbAuthConsumerUser.AuthConsumerUser
	return nil
}

// InsertConsumerUser in database.
func InsertConsumerUser(ctx context.Context, db gorpmapper.SqlExecutorWithTx, acu *sdk.AuthConsumerUser) error {
	// Because we need to create consumers before CDS first start with the init token, the consumer id can be set.
	// In this case we don't want to create a new UUID.
	if acu.ID == "" {
		acu.ID = sdk.UUID()
	}
	c := authConsumerUser{AuthConsumerUser: *acu}
	if err := gorpmapping.InsertAndSign(ctx, db, &c); err != nil {
		return sdk.WrapError(err, "unable to insert auth consumer user")
	}
	*acu = c.AuthConsumerUser
	return nil
}

// UpdateConsumerUser in database.
func UpdateConsumerUser(ctx context.Context, db gorpmapper.SqlExecutorWithTx, acu *sdk.AuthConsumerUser) error {
	c := authConsumerUser{AuthConsumerUser: *acu}
	if err := gorpmapping.UpdateAndSign(ctx, db, &c); err != nil {
		return sdk.WrapError(err, "unable to update auth consumer with id: %s", acu.ID)
	}
	*acu = c.AuthConsumerUser
	return nil
}
