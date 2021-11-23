package user_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ovh/cds/engine/api/bootstrap"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/engine/api/user"
)

func TestDAO_AuthentifiedUserOrganization(t *testing.T) {
	db, _ := test.SetupPG(t, bootstrap.InitiliazeDB)

	u, _ := assets.InsertLambdaUser(t, db)

	_, err := user.LoadOrganizationByUserID(context.TODO(), db, u.ID)
	require.Error(t, err)

	require.NoError(t, user.InsertOrganization(context.TODO(), db, &user.Organization{
		AuthentifiedUserID: u.ID,
		Organization:       "one",
	}))

	org, err := user.LoadOrganizationByUserID(context.TODO(), db, u.ID)
	require.NoError(t, err)
	require.NotNil(t, org)
	require.Equal(t, "one", org.Organization)
}
