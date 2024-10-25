package authentication_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ovh/cds/engine/api/authentication"
	"github.com/ovh/cds/engine/api/bootstrap"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/engine/api/user"
	"github.com/ovh/cds/sdk"
)

func TestLoadConsumer(t *testing.T) {
	db, _ := test.SetupPG(t, bootstrap.InitiliazeDB)

	assets.DeleteConsumers(t, db)

	u := sdk.AuthentifiedUser{
		Username: sdk.RandomString(10),
	}
	require.NoError(t, user.Insert(context.TODO(), db, &u))

	c1 := sdk.AuthUserConsumer{
		AuthConsumer: sdk.AuthConsumer{
			Name:            sdk.RandomString(10),
			Description:     sdk.RandomString(10),
			Type:            sdk.ConsumerLocal,
			ValidityPeriods: sdk.NewAuthConsumerValidityPeriod(time.Now(), 0),
		},
		AuthConsumerUser: sdk.AuthUserConsumerData{
			ScopeDetails:       sdk.NewAuthConsumerScopeDetails(sdk.AuthConsumerScopeAdmin),
			GroupIDs:           []int64{5, 10},
			AuthentifiedUserID: u.ID,
		},
	}
	require.NoError(t, authentication.InsertUserConsumer(context.TODO(), db, &c1))

	c2 := sdk.AuthUserConsumer{
		AuthConsumer: sdk.AuthConsumer{
			Name:            sdk.RandomString(10),
			Description:     sdk.RandomString(10),
			Type:            sdk.ConsumerBuiltin,
			ValidityPeriods: sdk.NewAuthConsumerValidityPeriod(time.Now(), 0),
		},
		AuthConsumerUser: sdk.AuthUserConsumerData{
			ScopeDetails:       sdk.NewAuthConsumerScopeDetails(sdk.AuthConsumerScopeAdmin),
			GroupIDs:           []int64{10, 15},
			AuthentifiedUserID: u.ID,
		},
	}
	require.NoError(t, authentication.InsertUserConsumer(context.TODO(), db, &c2))

	// LoadUserConsumerByID
	res, err := authentication.LoadUserConsumerByID(context.TODO(), db, sdk.RandomString(10))
	assert.Error(t, err)
	res, err = authentication.LoadUserConsumerByID(context.TODO(), db, c1.ID)
	assert.NoError(t, err)
	test.Equal(t, c1, res)

	// LoadUserConsumerByTypeAndUserID
	res, err = authentication.LoadUserConsumerByTypeAndUserID(context.TODO(), db, sdk.ConsumerLDAP, sdk.RandomString(10))
	assert.Error(t, err)
	res, err = authentication.LoadUserConsumerByTypeAndUserID(context.TODO(), db, sdk.ConsumerLocal, u.ID)
	assert.NoError(t, err)
	test.Equal(t, c1, res)

	// LoadUserConsumersByUserID
	cs, err := authentication.LoadUserConsumersByUserID(context.TODO(), db, sdk.RandomString(10))
	assert.NoError(t, err)
	assert.Len(t, cs, 0)
	cs, err = authentication.LoadUserConsumersByUserID(context.TODO(), db, u.ID)
	assert.NoError(t, err)
	require.Len(t, cs, 2)
	test.Equal(t, c1, cs[0])
	test.Equal(t, c2, cs[1])

	// LoadUserConsumersByGroupID
	cs, err = authentication.LoadUserConsumersByGroupID(context.TODO(), db, 10)
	require.NoError(t, err)
	require.Len(t, cs, 2)
	assert.Equal(t, c1.ID, cs[0].ID)
	assert.Equal(t, c2.ID, cs[1].ID)
	cs, err = authentication.LoadUserConsumersByGroupID(context.TODO(), db, 5)
	require.NoError(t, err)
	require.Len(t, cs, 1)
	assert.Equal(t, c1.ID, cs[0].ID)
	cs, err = authentication.LoadUserConsumersByGroupID(context.TODO(), db, 0)
	require.NoError(t, err)
	require.Len(t, cs, 0)
}

func TestInsertConsumer(t *testing.T) {
	db, _ := test.SetupPG(t, bootstrap.InitiliazeDB)

	u := sdk.AuthentifiedUser{
		Username: sdk.RandomString(10),
	}
	require.NoError(t, user.Insert(context.TODO(), db, &u))

	c := sdk.AuthUserConsumer{
		AuthConsumer: sdk.AuthConsumer{
			Name:            sdk.RandomString(10),
			ValidityPeriods: sdk.NewAuthConsumerValidityPeriod(time.Now(), 0),
		},
		AuthConsumerUser: sdk.AuthUserConsumerData{
			AuthentifiedUserID: u.ID,
		},
	}
	require.NoError(t, authentication.InsertUserConsumer(context.TODO(), db, &c))

	res, err := authentication.LoadUserConsumerByID(context.TODO(), db, c.ID)
	require.NoError(t, err)
	test.NotNil(t, res)
	test.Equal(t, c, res)
}

func TestUpdateConsumer(t *testing.T) {
	db, _ := test.SetupPG(t, bootstrap.InitiliazeDB)

	u := sdk.AuthentifiedUser{
		Username: sdk.RandomString(10),
	}
	require.NoError(t, user.Insert(context.TODO(), db, &u))

	c := sdk.AuthUserConsumer{
		AuthConsumer: sdk.AuthConsumer{
			Name:            sdk.RandomString(10),
			ValidityPeriods: sdk.NewAuthConsumerValidityPeriod(time.Now(), 0),
		},
		AuthConsumerUser: sdk.AuthUserConsumerData{
			AuthentifiedUserID: u.ID,
		},
	}
	require.NoError(t, authentication.InsertUserConsumer(context.TODO(), db, &c))

	c.Description = sdk.RandomString(10)
	assert.NoError(t, authentication.UpdateUserConsumer(context.TODO(), db, &c))

	res, err := authentication.LoadUserConsumerByID(context.TODO(), db, c.ID)
	assert.NoError(t, err)
	test.Equal(t, c, res)
}

func TestDeleteConsumer(t *testing.T) {
	db, _ := test.SetupPG(t, bootstrap.InitiliazeDB)

	u := sdk.AuthentifiedUser{
		Username: sdk.RandomString(10),
	}
	require.NoError(t, user.Insert(context.TODO(), db, &u))

	c := sdk.AuthUserConsumer{
		AuthConsumer: sdk.AuthConsumer{
			Name:            sdk.RandomString(10),
			ValidityPeriods: sdk.NewAuthConsumerValidityPeriod(time.Now(), 0),
		},
		AuthConsumerUser: sdk.AuthUserConsumerData{
			AuthentifiedUserID: u.ID,
		},
	}
	require.NoError(t, authentication.InsertUserConsumer(context.TODO(), db, &c))

	_, err := authentication.LoadUserConsumerByID(context.TODO(), db, c.ID)
	assert.NoError(t, err)

	assert.NoError(t, authentication.DeleteConsumerByID(db, c.ID))

	_, err = authentication.LoadUserConsumerByID(context.TODO(), db, c.ID)
	assert.Error(t, err)
}
