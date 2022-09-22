package user_test

import (
	"context"
	"github.com/ovh/cds/engine/api/organization"
	"github.com/ovh/cds/sdk"
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

	o := sdk.Organization{Name: "myorg"}
	require.NoError(t, organization.Insert(context.TODO(), db, &o))

	require.NoError(t, user.InsertUserOrganization(context.TODO(), db, &user.UserOrganization{
		AuthentifiedUserID: u.ID,
		OrganizationID:     o.ID,
	}))

	userDB, err := user.LoadByID(context.TODO(), db, u.ID, user.LoadOptions.WithOrganization)
	require.NoError(t, err)
	require.Equal(t, "one", userDB.Organization)
}
