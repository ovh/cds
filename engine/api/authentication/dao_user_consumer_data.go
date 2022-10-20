package authentication

import (
	"context"

	"github.com/go-gorp/gorp"
	"github.com/rockbears/log"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
)

func loadConsumerUserDataByConsumerID(ctx context.Context, db gorp.SqlExecutor, ac *sdk.AuthUserConsumer) error {
	query := gorpmapping.NewQuery("SELECT * FROM auth_consumer_user WHERE auth_consumer_id = $1").Args(ac.ID)
	var dbAuthConsumerUserData authConsumerUserData
	found, err := gorpmapping.Get(ctx, db, query, &dbAuthConsumerUserData)
	if err != nil {
		return sdk.WrapError(err, "cannot get auth consumer user")
	}
	if !found {
		return nil
	}

	isValid, err := gorpmapping.CheckSignature(dbAuthConsumerUserData, dbAuthConsumerUserData.Signature)
	if err != nil {
		return err
	}
	if !isValid {
		log.Error(ctx, "authentication.loadConsumerUserDataByConsumerID> auth consumer user from au consumer %s data corrupted", ac.ID)
		return sdk.WithStack(sdk.ErrNotFound)
	}
	ac.AuthConsumerUser = dbAuthConsumerUserData.AuthUserConsumerData
	return nil
}

// insertConsumerUserData in database.
func insertConsumerUserData(ctx context.Context, db gorpmapper.SqlExecutorWithTx, acu *sdk.AuthUserConsumerData) error {
	// Because we need to create consumers before CDS first start with the init token, the consumer id can be set.
	// In this case we don't want to create a new UUID.
	if acu.ID == "" {
		acu.ID = sdk.UUID()
	}
	c := authConsumerUserData{AuthUserConsumerData: *acu}
	if err := gorpmapping.InsertAndSign(ctx, db, &c); err != nil {
		return sdk.WrapError(err, "unable to insert auth consumer user")
	}
	*acu = c.AuthUserConsumerData
	return nil
}

// updateConsumerUserData in database.
func updateConsumerUserData(ctx context.Context, db gorpmapper.SqlExecutorWithTx, acu *sdk.AuthUserConsumerData) error {
	c := authConsumerUserData{AuthUserConsumerData: *acu}
	if err := gorpmapping.UpdateAndSign(ctx, db, &c); err != nil {
		return sdk.WrapError(err, "unable to update auth consumer with id: %s", acu.ID)
	}
	*acu = c.AuthUserConsumerData
	return nil
}

func getAllConsumerUserData(ctx context.Context, db gorp.SqlExecutor, q gorpmapping.Query) ([]sdk.AuthUserConsumerData, error) {
	var dbAuthConsumerUsers []authConsumerUserData
	if err := gorpmapping.GetAll(ctx, db, q, &dbAuthConsumerUsers); err != nil {
		return nil, sdk.WrapError(err, "cannot get auth consumer users")
	}

	authConsumerUsers := make([]sdk.AuthUserConsumerData, 0, len(dbAuthConsumerUsers))
	for _, cu := range dbAuthConsumerUsers {
		isValid, err := gorpmapping.CheckSignature(cu, cu.Signature)
		if err != nil {
			return nil, err
		}
		if !isValid {
			log.Error(ctx, "authentication.loadConsumerUserDataByConsumerID> auth consumer user %s data corrupted", cu.ID)
			continue
		}
		authConsumerUsers = append(authConsumerUsers, cu.AuthUserConsumerData)
	}
	return authConsumerUsers, nil
}

func loadConsumerUserDataByUserID(ctx context.Context, db gorp.SqlExecutor, userID string) ([]sdk.AuthUserConsumerData, error) {
	query := gorpmapping.NewQuery("SELECT * FROM auth_consumer_user WHERE user_id = $1").Args(userID)
	return getAllConsumerUserData(ctx, db, query)
}

func loadConsumerUsersDataByGroupID(ctx context.Context, db gorp.SqlExecutor, groupID int64) ([]sdk.AuthUserConsumerData, error) {
	query := gorpmapping.NewQuery("SELECT * FROM auth_consumer_user WHERE group_ids @> $1 OR invalid_group_ids @> $1").Args(groupID)
	return getAllConsumerUserData(ctx, db, query)
}

func loadConsumerUserDataByUserExternalID(ctx context.Context, db gorp.SqlExecutor, userExternalID string) ([]sdk.AuthUserConsumerData, error) {
	query := gorpmapping.NewQuery("SELECT * FROM auth_consumer_user WHERE (data->>'external_id')::text = $1").Args(userExternalID)
	return getAllConsumerUserData(ctx, db, query)
}
