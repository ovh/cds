package local_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ovh/cds/engine/api/authentication/local"
	"github.com/ovh/cds/engine/api/bootstrap"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/sdk"
)

func TestInsertRegistration(t *testing.T) {
	db, _ := test.SetupPG(t, bootstrap.InitializeDB)

	r1 := sdk.UserRegistration{
		Username: sdk.RandomString(10),
		Fullname: sdk.RandomString(10),
		Email:    sdk.RandomString(10),
		Hash:     sdk.RandomString(10),
	}
	require.NoError(t, local.InsertRegistration(context.TODO(), db, &r1))

	r2 := sdk.UserRegistration{
		Username: r1.Username,
		Fullname: sdk.RandomString(10),
		Email:    r1.Email,
		Hash:     sdk.RandomString(10),
	}
	require.Error(t, local.InsertRegistration(context.TODO(), db, &r2))

	res, err := local.LoadRegistrationByID(context.TODO(), db, r1.ID)
	require.NoError(t, err)
	test.NotNil(t, res)
	test.Equal(t, r1, res)

	require.NoError(t, local.DeleteRegistrationByID(db, r1.ID))
	_, err = local.LoadRegistrationByID(context.TODO(), db, r1.ID)
	assert.Error(t, err)
}
