package authentication

import (
	"context"

	"github.com/go-gorp/gorp"
	"github.com/rockbears/log"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
)

// InsertHatcheryConsumer in database.
func InsertHatcheryConsumer(ctx context.Context, db gorpmapper.SqlExecutorWithTx, ac *sdk.AuthHatcheryConsumer) error {
	if err := insertConsumer(ctx, db, &ac.AuthConsumer); err != nil {
		return sdk.WrapError(err, "unable to insert auth consumer")
	}
	ac.AuthConsumerHatchery.AuthConsumerID = ac.ID
	return insertConsumerHatcheryData(ctx, db, &ac.AuthConsumerHatchery)
}

// InsertConsumerHatchery in database.
func insertConsumerHatcheryData(ctx context.Context, db gorpmapper.SqlExecutorWithTx, ach *sdk.AuthConsumerHatcheryData) error {
	if ach.ID == "" {
		ach.ID = sdk.UUID()
	}
	c := authConsumerHatcheryData{AuthConsumerHatcheryData: *ach}
	if err := gorpmapping.InsertAndSign(ctx, db, &c); err != nil {
		return sdk.WrapError(err, "unable to insert auth consumer user")
	}
	*ach = c.AuthConsumerHatcheryData
	return nil
}

func LoadHatcheryConsumerByID(ctx context.Context, db gorp.SqlExecutor, consumerID string) (*sdk.AuthHatcheryConsumer, error) {
	c, err := LoadConsumerByID(ctx, db, consumerID)
	if err != nil {
		return nil, err
	}
	q := gorpmapping.NewQuery("SELECT * from auth_consumer_hatchery WHERE auth_consumer_id = $1").Args(c.ID)
	hatcheryData, err := getAuthConsumerHatchery(ctx, db, q)
	if err != nil {
		return nil, err
	}
	hc := sdk.AuthHatcheryConsumer{
		AuthConsumer:         *c,
		AuthConsumerHatchery: *hatcheryData,
	}
	return &hc, nil
}

func LoadHatcheryConsumerByName(ctx context.Context, db gorp.SqlExecutor, hatcheryName string) (*sdk.AuthHatcheryConsumer, error) {
	q := gorpmapping.NewQuery("SELECT * FROM auth_consumer WHERE type = $1 AND name = $2").Args(sdk.ConsumerHatchery, hatcheryName)
	c, err := getConsumer(ctx, db, q)
	if err != nil {

	}
	queryhatch := gorpmapping.NewQuery("SELECT * from auth_consumer_hatchery WHERE auth_consumer_id = $1").Args(c.ID)
	hatcheryData, err := getAuthConsumerHatchery(ctx, db, queryhatch)
	if err != nil {
		return nil, err
	}
	hc := sdk.AuthHatcheryConsumer{
		AuthConsumer:         *c,
		AuthConsumerHatchery: *hatcheryData,
	}
	return &hc, nil
}

func LoadHatcheryConsumerByHatcheryID(ctx context.Context, db gorp.SqlExecutor, hatcheryID string) (*sdk.AuthHatcheryConsumer, error) {
	q := gorpmapping.NewQuery("SELECT * from auth_consumer_hatchery WHERE hatchery_id = $1").Args(hatcheryID)
	hatcheryData, err := getAuthConsumerHatchery(ctx, db, q)
	if err != nil {
		return nil, err
	}

	c, err := LoadConsumerByID(ctx, db, hatcheryData.AuthConsumerID)
	if err != nil {
		return nil, err
	}
	hc := sdk.AuthHatcheryConsumer{
		AuthConsumer:         *c,
		AuthConsumerHatchery: *hatcheryData,
	}
	return &hc, nil
}

func getAuthConsumerHatchery(ctx context.Context, db gorp.SqlExecutor, q gorpmapping.Query) (*sdk.AuthConsumerHatcheryData, error) {
	var dbAuthConsumerHatchery authConsumerHatcheryData
	_, err := gorpmapping.Get(ctx, db, q, &dbAuthConsumerHatchery)
	if err != nil {
		return nil, err
	}
	isValid, err := gorpmapping.CheckSignature(dbAuthConsumerHatchery, dbAuthConsumerHatchery.Signature)
	if err != nil {
		return nil, err
	}
	if !isValid {
		log.Error(ctx, "authentication.getAllAuthConsumerHatcheries> auth consumer hatchery %s data corrupted", dbAuthConsumerHatchery.ID)
		return nil, sdk.WithStack(sdk.ErrNotFound)
	}
	return &dbAuthConsumerHatchery.AuthConsumerHatcheryData, nil
}
