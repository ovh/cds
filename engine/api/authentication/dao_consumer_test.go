package authentication_test

import (
	"context"
	"testing"

	"github.com/ovh/cds/engine/api/authentication"
	"github.com/ovh/cds/engine/api/bootstrap"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/user"
	"github.com/ovh/cds/sdk"
)

func TestLoadConsumer(t *testing.T) {
	db, _, end := test.SetupPG(t, bootstrap.InitiliazeDB)
	defer end()

	u := sdk.AuthentifiedUser{
		Username: sdk.RandomString(10),
	}
	test.NoError(t, user.Insert(db, &u))

	c := sdk.AuthConsumer{
		Name:               sdk.RandomString(10),
		Description:        sdk.RandomString(10),
		Type:               sdk.ConsumerLocal,
		Scopes:             []string{sdk.AccessTokenScopeALL},
		AuthentifiedUserID: u.ID,
	}
	test.NoError(t, authentication.InsertConsumer(db, &c))

	// LoadConsumerByID
	res, err := authentication.LoadConsumerByID(context.TODO(), db, sdk.RandomString(10))
	test.NoError(t, err)
	test.Nil(t, res)
	res, err = authentication.LoadConsumerByID(context.TODO(), db, c.ID)
	test.NoError(t, err)
	test.NotNil(t, res)
	test.Equal(t, c, res)

	// LoadConsumerByTypeAndUserID
	res, err = authentication.LoadConsumerByTypeAndUserID(context.TODO(), db, sdk.ConsumerLDAP, sdk.RandomString(10))
	test.NoError(t, err)
	test.Nil(t, res)
	res, err = authentication.LoadConsumerByTypeAndUserID(context.TODO(), db, sdk.ConsumerLocal, u.ID)
	test.NoError(t, err)
	test.NotNil(t, res)
	test.Equal(t, c, res)
}

func TestInsertConsumer(t *testing.T) {
	db, _, end := test.SetupPG(t, bootstrap.InitiliazeDB)
	defer end()

	u := sdk.AuthentifiedUser{
		Username: sdk.RandomString(10),
	}
	test.NoError(t, user.Insert(db, &u))

	c := sdk.AuthConsumer{
		Name:               sdk.RandomString(10),
		AuthentifiedUserID: u.ID,
	}
	test.NoError(t, authentication.InsertConsumer(db, &c))

	res, err := authentication.LoadConsumerByID(context.TODO(), db, c.ID)
	test.NoError(t, err)
	test.NotNil(t, res)
	test.Equal(t, c, res)
}

func TestUpdateConsumer(t *testing.T) {
	db, _, end := test.SetupPG(t, bootstrap.InitiliazeDB)
	defer end()

	u := sdk.AuthentifiedUser{
		Username: sdk.RandomString(10),
	}
	test.NoError(t, user.Insert(db, &u))

	c := sdk.AuthConsumer{
		Name:               sdk.RandomString(10),
		AuthentifiedUserID: u.ID,
	}
	test.NoError(t, authentication.InsertConsumer(db, &c))

	c.Description = sdk.RandomString(10)
	test.NoError(t, authentication.UpdateConsumer(db, &c))

	res, err := authentication.LoadConsumerByID(context.TODO(), db, c.ID)
	test.NoError(t, err)
	test.NotNil(t, res)
	test.Equal(t, c, res)
}

func TestDeleteConsumer(t *testing.T) {
	db, _, end := test.SetupPG(t, bootstrap.InitiliazeDB)
	defer end()

	u := sdk.AuthentifiedUser{
		Username: sdk.RandomString(10),
	}
	test.NoError(t, user.Insert(db, &u))

	c := sdk.AuthConsumer{
		Name:               sdk.RandomString(10),
		AuthentifiedUserID: u.ID,
	}
	test.NoError(t, authentication.InsertConsumer(db, &c))

	res, err := authentication.LoadConsumerByID(context.TODO(), db, c.ID)
	test.NoError(t, err)
	test.NotNil(t, res)

	test.NoError(t, authentication.DeleteConsumerByID(db, c.ID))

	res, err = authentication.LoadConsumerByID(context.TODO(), db, c.ID)
	test.NoError(t, err)
	test.Nil(t, res)
}
