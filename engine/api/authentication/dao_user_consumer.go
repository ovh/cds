package authentication

import (
	"context"
	"github.com/go-gorp/gorp"
	"github.com/lib/pq"
	"github.com/rockbears/log"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
)

func getUserConsumers(ctx context.Context, db gorp.SqlExecutor, q gorpmapping.Query, opts ...LoadUserConsumerOptionFunc) (sdk.AuthUserConsumers, error) {
	consumers, err := getConsumers(ctx, db, q)
	if err != nil {
		return nil, err
	}
	userConsumers := make([]*sdk.AuthUserConsumer, 0, len(consumers))

	for i := range consumers {
		uc := sdk.AuthUserConsumer{
			AuthConsumer: consumers[i],
		}
		if err := loadConsumerUserDataByConsumerID(ctx, db, &uc); err != nil {
			return nil, err
		}
		userConsumers = append(userConsumers, &uc)
	}

	if len(userConsumers) > 0 {
		for i := range opts {
			if err := opts[i](ctx, db, userConsumers...); err != nil {
				return nil, err
			}
		}
	}

	fullUserConsumers := make([]sdk.AuthUserConsumer, 0, len(userConsumers))
	for i := range userConsumers {
		fullUserConsumers = append(fullUserConsumers, *userConsumers[i])
	}

	return fullUserConsumers, nil
}

func getUserConsumer(ctx context.Context, db gorp.SqlExecutor, q gorpmapping.Query, opts ...LoadUserConsumerOptionFunc) (*sdk.AuthUserConsumer, error) {
	c, err := getConsumer(ctx, db, q)
	if err != nil {
		return nil, err
	}
	userConsumer := sdk.AuthUserConsumer{
		AuthConsumer: *c,
	}
	if err := loadConsumerUserDataByConsumerID(ctx, db, &userConsumer); err != nil {
		return nil, err
	}
	for i := range opts {
		if err := opts[i](ctx, db, &userConsumer); err != nil {
			return nil, err
		}
	}
	userConsumer.ValidityPeriods.Sort()
	return &userConsumer, nil
}

// LoadUserConsumersByUserID returns auth consumers from database for given user id.
func LoadUserConsumersByUserID(ctx context.Context, db gorp.SqlExecutor, id string, opts ...LoadUserConsumerOptionFunc) (sdk.AuthUserConsumers, error) {
	consumerUsers, err := loadConsumerUserDataByUserID(ctx, db, id)
	if err != nil {
		return nil, err
	}
	consumerIDs := make([]string, 0, len(consumerUsers))
	for _, cu := range consumerUsers {
		consumerIDs = append(consumerIDs, cu.AuthConsumerID)
	}

	query := gorpmapping.NewQuery("SELECT * FROM auth_consumer WHERE id = ANY($1) ORDER BY created ASC").Args(pq.StringArray(consumerIDs))
	consumers, err := getUserConsumers(ctx, db, query, opts...)
	if err != nil {
		return nil, err
	}
	return consumers, nil
}

// LoadUserConsumersByGroupID returns all consumers from database that refer to given group id.
func LoadUserConsumersByGroupID(ctx context.Context, db gorp.SqlExecutor, groupID int64, opts ...LoadUserConsumerOptionFunc) (sdk.AuthUserConsumers, error) {
	consumerUsers, err := loadConsumerUsersDataByGroupID(ctx, db, groupID)
	if err != nil {
		return nil, err
	}
	consumerIDs := make([]string, 0, len(consumerUsers))
	for _, cu := range consumerUsers {
		consumerIDs = append(consumerIDs, cu.AuthConsumerID)
	}

	query := gorpmapping.NewQuery("SELECT * FROM auth_consumer WHERE id = ANY($1) ORDER BY created ASC").Args(pq.StringArray(consumerIDs))
	consumers, err := getUserConsumers(ctx, db, query, opts...)
	if err != nil {
		return nil, err
	}
	return consumers, nil
}

// LoadUserConsumerByID returns an auth consumer from database.
func LoadUserConsumerByID(ctx context.Context, db gorp.SqlExecutor, id string, opts ...LoadUserConsumerOptionFunc) (*sdk.AuthUserConsumer, error) {
	query := gorpmapping.NewQuery("SELECT * FROM auth_consumer WHERE id = $1").Args(id)
	return getUserConsumer(ctx, db, query, opts...)
}

// LoadUserConsumerByTypeAndUserID returns an auth consumer from database for given type and user id.
func LoadUserConsumerByTypeAndUserID(ctx context.Context, db gorp.SqlExecutor, consumerType sdk.AuthConsumerType, userID string, opts ...LoadUserConsumerOptionFunc) (*sdk.AuthUserConsumer, error) {
	consumerUsers, err := loadConsumerUserDataByUserID(ctx, db, userID)
	if err != nil {
		return nil, err
	}
	consumerIDs := make([]string, 0, len(consumerUsers))
	for _, cu := range consumerUsers {
		consumerIDs = append(consumerIDs, cu.AuthConsumerID)
	}
	query := gorpmapping.NewQuery("SELECT * FROM auth_consumer WHERE type = $1 AND id = ANY($2)").Args(consumerType, pq.StringArray(consumerIDs))
	consumer, err := getUserConsumer(ctx, db, query, opts...)
	if err != nil {
		return nil, err
	}
	return consumer, nil
}

// LoadUserConsumerByTypeAndUserExternalID returns an auth consumer from database for given type and user id.
func LoadUserConsumerByTypeAndUserExternalID(ctx context.Context, db gorp.SqlExecutor, consumerType sdk.AuthConsumerType, userExternalID string, opts ...LoadUserConsumerOptionFunc) (*sdk.AuthUserConsumer, error) {
	consumerUsers, err := loadConsumerUserDataByUserExternalID(ctx, db, userExternalID)
	if err != nil {
		return nil, err
	}
	consumerIDs := make([]string, 0, len(consumerUsers))
	for _, cu := range consumerUsers {
		consumerIDs = append(consumerIDs, cu.AuthConsumerID)
	}
	query := gorpmapping.NewQuery("SELECT * FROM auth_consumer WHERE type = $1 AND id = ANY($2)").Args(consumerType, pq.StringArray(consumerIDs))
	consumer, err := getUserConsumer(ctx, db, query, opts...)
	if err != nil {
		return nil, err
	}
	return consumer, nil
}

// InsertUserConsumer in database.
func InsertUserConsumer(ctx context.Context, db gorpmapper.SqlExecutorWithTx, ac *sdk.AuthUserConsumer) error {
	if err := inserConsumer(ctx, db, &ac.AuthConsumer); err != nil {
		return sdk.WrapError(err, "unable to insert auth consumer")
	}
	ac.AuthConsumerUser.AuthConsumerID = ac.ID
	return insertConsumerUserData(ctx, db, &ac.AuthConsumerUser)
}

// UpdateUserConsumer in database.
func UpdateUserConsumer(ctx context.Context, db gorpmapper.SqlExecutorWithTx, ac *sdk.AuthUserConsumer) error {
	ac.ValidityPeriods.Sort()
	if err := updateConsumer(ctx, db, &ac.AuthConsumer); err != nil {
		return err
	}
	ac.AuthConsumerUser.AuthConsumerID = ac.ID
	return updateConsumerUserData(ctx, db, &ac.AuthConsumerUser)

}

// DEPRECATED - load old consumers, only use for migration
func LoadOldConsumers(ctx context.Context, db gorp.SqlExecutor) ([]AuthConsumerOld, error) {
	query := gorpmapping.NewQuery("SELECT * FROM auth_consumer_old order by created ASC")
	var consumers []AuthConsumerOld

	if err := gorpmapping.GetAll(ctx, db, query, &consumers); err != nil {
		return nil, sdk.WrapError(err, "cannot get old auth consumers")
	}

	// Check signature of data, if invalid do not return it
	for _, c := range consumers {
		isValid, err := gorpmapping.CheckSignature(c, c.Signature)
		if err != nil {
			return nil, err
		}
		if !isValid {
			log.Error(ctx, "authentication.getConsumers> auth consumer %s data corrupted", c.ID)
			continue
		}
	}
	return consumers, nil
}
