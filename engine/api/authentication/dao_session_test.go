package authentication_test

import (
	"context"
	"testing"
	"time"

	"github.com/ovh/cds/engine/api/authentication"
	"github.com/ovh/cds/engine/api/bootstrap"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/engine/api/user"
	"github.com/ovh/cds/sdk"
)

func TestLoadSession(t *testing.T) {
	db, _, end := test.SetupPG(t, bootstrap.InitiliazeDB)
	defer end()

	u := sdk.AuthentifiedUser{
		Username: sdk.RandomString(10),
	}
	test.NoError(t, user.Insert(db, &u))

	c, err := builtin.NewConsumer(db, sdk.RandomString(10), "", u.ID, nil, nil)
	test.NoError(t, err)

	s, err := authentication.NewSession(db, c, time.Second)
	test.NoError(t, err)

	// LoadSessionByID
	res, err := authentication.LoadSessionByID(context.TODO(), db, sdk.RandomString(10))
	test.NoError(t, err)
	test.Nil(t, res)
	res, err = authentication.LoadSessionByID(context.TODO(), db, s.ID)
	test.NoError(t, err)
	test.NotNil(t, res)
	test.Equal(t, s, res)
}

func TestInsertSession(t *testing.T) {
	db, _, end := test.SetupPG(t, bootstrap.InitiliazeDB)
	defer end()

	u := sdk.AuthentifiedUser{
		Username: sdk.RandomString(10),
	}
	test.NoError(t, user.Insert(db, &u))

	c, err := builtin.NewConsumer(db, sdk.RandomString(10), "", u.ID, nil, nil)
	test.NoError(t, err)

	s, err := authentication.NewSession(db, c, time.Second)
	test.NoError(t, err)

	res, err := authentication.LoadSessionByID(context.TODO(), db, s.ID)
	test.NoError(t, err)
	test.NotNil(t, res)
	test.Equal(t, s, res)
}

func TestUpdateSession(t *testing.T) {
	db, _, end := test.SetupPG(t, bootstrap.InitiliazeDB)
	defer end()

	g := assets.InsertGroup(t, db)

	u := sdk.AuthentifiedUser{
		Username: sdk.RandomString(10),
	}
	test.NoError(t, user.Insert(db, &u))

	c, err := builtin.NewConsumer(db, sdk.RandomString(10), "", u.ID, nil, nil)
	test.NoError(t, err)

	s, err := authentication.NewSession(db, c, time.Second)
	test.NoError(t, err)

	s.GroupIDs = []int64{g.ID}
	test.NoError(t, authentication.UpdateSession(db, s))

	res, err := authentication.LoadSessionByID(context.TODO(), db, s.ID)
	test.NoError(t, err)
	test.NotNil(t, res)
	test.Equal(t, s, res)
}

func TestDeleteSession(t *testing.T) {
	db, _, end := test.SetupPG(t, bootstrap.InitiliazeDB)
	defer end()

	u := sdk.AuthentifiedUser{
		Username: sdk.RandomString(10),
	}
	test.NoError(t, user.Insert(db, &u))

	c, err := builtin.NewConsumer(db, sdk.RandomString(10), "", u.ID, nil, nil)
	test.NoError(t, err)

	s, err := authentication.NewSession(db, c, time.Second)
	test.NoError(t, err)

	res, err := authentication.LoadSessionByID(context.TODO(), db, s.ID)
	test.NoError(t, err)
	test.NotNil(t, res)

	test.NoError(t, authentication.DeleteSessionByID(db, s.ID))

	res, err = authentication.LoadSessionByID(context.TODO(), db, s.ID)
	test.NoError(t, err)
	test.Nil(t, res)
}
