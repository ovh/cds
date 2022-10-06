package authentication

import (
	"context"
	"time"

	"github.com/go-gorp/gorp"
	"github.com/lib/pq"
	"github.com/rockbears/log"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
)

func getConsumers(ctx context.Context, db gorp.SqlExecutor, q gorpmapping.Query, opts ...LoadConsumerOptionFunc) (sdk.AuthConsumers, error) {
	cs := []authConsumer{}

	if err := gorpmapping.GetAll(ctx, db, q, &cs); err != nil {
		return nil, sdk.WrapError(err, "cannot get auth consumers")
	}

	// Check signature of data, if invalid do not return it
	verifiedConsumers := make([]*sdk.AuthConsumer, 0, len(cs))
	for i := range cs {
		isValid, err := gorpmapping.CheckSignature(cs[i], cs[i].Signature)
		if err != nil {
			return nil, err
		}
		if !isValid {
			log.Error(ctx, "authentication.getConsumers> auth consumer %s data corrupted", cs[i].ID)
			continue
		}
		verifiedConsumers = append(verifiedConsumers, &cs[i].AuthConsumer)
	}

	consumers := make([]sdk.AuthConsumer, len(verifiedConsumers))
	for i := range verifiedConsumers {
		consumers[i] = *verifiedConsumers[i]
		if err := loadConsumerUser(ctx, db, &consumers[i]); err != nil {
			return nil, err
		}
	}

	if len(verifiedConsumers) > 0 {
		for i := range opts {
			if err := opts[i](ctx, db, verifiedConsumers...); err != nil {
				return nil, err
			}
		}
	}

	return consumers, nil
}

func getConsumer(ctx context.Context, db gorp.SqlExecutor, q gorpmapping.Query, opts ...LoadConsumerOptionFunc) (*sdk.AuthConsumer, error) {
	var consumer authConsumer

	found, err := gorpmapping.Get(ctx, db, q, &consumer)
	if err != nil {
		return nil, sdk.WrapError(err, "cannot get auth consumer")
	}
	if !found {
		return nil, sdk.WithStack(sdk.ErrNotFound)
	}

	isValid, err := gorpmapping.CheckSignature(consumer, consumer.Signature)
	if err != nil {
		return nil, err
	}
	if !isValid {
		log.Error(ctx, "authentication.getConsumer> auth consumer %s data corrupted", consumer.ID)
		return nil, sdk.WithStack(sdk.ErrNotFound)
	}

	ac := consumer.AuthConsumer

	if err := loadConsumerUser(ctx, db, &ac); err != nil {
		return nil, err
	}

	for i := range opts {
		if err := opts[i](ctx, db, &ac); err != nil {
			return nil, err
		}
	}
	ac.ValidityPeriods.Sort()
	return &ac, nil
}

// LoadConsumersByUserID returns auth consumers from database for given user id.
func LoadConsumersByUserID(ctx context.Context, db gorp.SqlExecutor, id string, opts ...LoadConsumerOptionFunc) (sdk.AuthConsumers, error) {
	consumerUsers, err := loadConsumerUserByID(ctx, db, id)
	if err != nil {
		return nil, err
	}
	consumerIDs := make([]string, 0, len(consumerUsers))
	for _, cu := range consumerUsers {
		consumerIDs = append(consumerIDs, cu.AuthConsumerID)
	}

	query := gorpmapping.NewQuery("SELECT * FROM auth_consumer WHERE id = ANY($1) ORDER BY created ASC").Args(pq.StringArray(consumerIDs))
	consumers, err := getConsumers(ctx, db, query, opts...)
	if err != nil {
		return nil, err
	}
	return consumers, nil
}

// LoadConsumersByGroupID returns all consumers from database that refer to given group id.
func LoadConsumersByGroupID(ctx context.Context, db gorp.SqlExecutor, groupID int64, opts ...LoadConsumerOptionFunc) (sdk.AuthConsumers, error) {
	consumerUsers, err := loadConsumerUsersByGroupID(ctx, db, groupID)
	if err != nil {
		return nil, err
	}
	consumerIDs := make([]string, 0, len(consumerUsers))
	for _, cu := range consumerUsers {
		consumerIDs = append(consumerIDs, cu.AuthConsumerID)
	}

	query := gorpmapping.NewQuery("SELECT * FROM auth_consumer WHERE id = ANY($1) ORDER BY created ASC").Args(pq.StringArray(consumerIDs))
	consumers, err := getConsumers(ctx, db, query, opts...)
	if err != nil {
		return nil, err
	}
	return consumers, nil
}

// LoadConsumerByID returns an auth consumer from database.
func LoadConsumerByID(ctx context.Context, db gorp.SqlExecutor, id string, opts ...LoadConsumerOptionFunc) (*sdk.AuthConsumer, error) {
	query := gorpmapping.NewQuery("SELECT * FROM auth_consumer WHERE id = $1").Args(id)
	return getConsumer(ctx, db, query, opts...)
}

// LoadConsumerByTypeAndUserID returns an auth consumer from database for given type and user id.
func LoadConsumerByTypeAndUserID(ctx context.Context, db gorp.SqlExecutor, consumerType sdk.AuthConsumerType, userID string, opts ...LoadConsumerOptionFunc) (*sdk.AuthConsumer, error) {
	consumerUsers, err := loadConsumerUserByID(ctx, db, userID)
	if err != nil {
		return nil, err
	}
	consumerIDs := make([]string, 0, len(consumerUsers))
	for _, cu := range consumerUsers {
		consumerIDs = append(consumerIDs, cu.AuthConsumerID)
	}
	query := gorpmapping.NewQuery("SELECT * FROM auth_consumer WHERE type = $1 AND id = ANY($2)").Args(consumerType, pq.StringArray(consumerIDs))
	consumer, err := getConsumer(ctx, db, query, opts...)
	if err != nil {
		return nil, err
	}
	return consumer, nil
}

// LoadConsumerByTypeAndUserExternalID returns an auth consumer from database for given type and user id.
func LoadConsumerByTypeAndUserExternalID(ctx context.Context, db gorp.SqlExecutor, consumerType sdk.AuthConsumerType, userExternalID string, opts ...LoadConsumerOptionFunc) (*sdk.AuthConsumer, error) {
	consumerUsers, err := loadConsumerByUserExternalID(ctx, db, userExternalID)
	if err != nil {
		return nil, err
	}
	consumerIDs := make([]string, 0, len(consumerUsers))
	for _, cu := range consumerUsers {
		consumerIDs = append(consumerIDs, cu.AuthConsumerID)
	}
	query := gorpmapping.NewQuery("SELECT * FROM auth_consumer WHERE type = $1 AND id = ANY($2)").Args(consumerType, pq.StringArray(consumerIDs))
	consumer, err := getConsumer(ctx, db, query, opts...)
	if err != nil {
		return nil, err
	}
	return consumer, nil
}

// InsertConsumer in database.
func InsertConsumer(ctx context.Context, db gorpmapper.SqlExecutorWithTx, ac *sdk.AuthConsumer) error {
	// Because we need to create consumers before CDS first start with the init token, the consumer id can be set.
	// In this case we don't want to create a new UUID.
	if ac.ID == "" {
		ac.ID = sdk.UUID()
	}
	ac.Created = time.Now()
	ac.ValidityPeriods.Sort()
	c := authConsumer{AuthConsumer: *ac}
	if err := gorpmapping.InsertAndSign(ctx, db, &c); err != nil {
		return sdk.WrapError(err, "unable to insert auth consumer")
	}
	*ac = c.AuthConsumer

	if ac.AuthConsumerUser != nil {
		ac.AuthConsumerUser.AuthConsumerID = ac.ID
		return InsertConsumerUser(ctx, db, ac.AuthConsumerUser)
	}

	return nil
}

// UpdateConsumer in database.
func UpdateConsumer(ctx context.Context, db gorpmapper.SqlExecutorWithTx, ac *sdk.AuthConsumer) error {
	ac.ValidityPeriods.Sort()
	c := authConsumer{AuthConsumer: *ac}
	if err := gorpmapping.UpdateAndSign(ctx, db, &c); err != nil {
		return sdk.WrapError(err, "unable to update auth consumer with id: %s", ac.ID)
	}
	*ac = c.AuthConsumer

	if ac.AuthConsumerUser != nil {
		ac.AuthConsumerUser.AuthConsumerID = ac.ID
		return UpdateConsumerUser(ctx, db, ac.AuthConsumerUser)
	}

	return nil
}

// DeleteConsumerByID removes a auth consumer in database for given id.
func DeleteConsumerByID(db gorp.SqlExecutor, id string) error {
	_, err := db.Exec("DELETE FROM auth_consumer WHERE id = $1", id)
	return sdk.WrapError(err, "unable to delete auth consumer with id %s", id)
}

// UpdateConsumerLastAuthentication updates only the column last_authentication
func UpdateConsumerLastAuthentication(ctx context.Context, db gorp.SqlExecutor, ac *sdk.AuthConsumer) error {
	c := authConsumer{AuthConsumer: *ac}
	err := gorpmapping.UpdateColumns(db, &c, func(cm *gorp.ColumnMap) bool {
		return cm.ColumnName == "last_authentication"
	})
	return sdk.WrapError(err, "unable to update last_authentication auth consumer with id %s", ac.ID)
}

// DEPRECATED - load old consumers, only use for migration
func LoadOldConsumers(ctx context.Context, db gorp.SqlExecutor) ([]AuthConsumerOld, error) {
	query := gorpmapping.NewQuery("SELECT * FROM auth_consumer")
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
