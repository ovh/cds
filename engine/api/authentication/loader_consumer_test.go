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

func TestWithAuthentifiedUser(t *testing.T) {
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

	res, err := authentication.LoadConsumerByID(context.TODO(), db, c.ID,
		authentication.LoadConsumerOptions.WithAuthentifiedUser)
	test.NoError(t, err)
	test.NotNil(t, res)
	test.NotNil(t, res.AuthentifiedUser)
	test.Equal(t, u.Username, res.AuthentifiedUser.Username)
}
