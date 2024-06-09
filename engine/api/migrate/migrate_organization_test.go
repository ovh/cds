package migrate

import (
	"context"
	"fmt"
	"testing"

	"github.com/ovh/cds/engine/api/bootstrap"
	"github.com/ovh/cds/engine/api/organization"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/user"
	"github.com/ovh/cds/sdk"
	"github.com/stretchr/testify/require"
)

func TestGetOrganizationUsersToMigrate_UserWithoutOrga(t *testing.T) {
	db, _ := test.SetupPG(t, bootstrap.InitializeDB)

	u := sdk.AuthentifiedUser{
		Username: sdk.RandomString(10),
	}
	require.NoError(t, user.Insert(context.TODO(), db, &u))

	_, err := GetOrganizationUsersToMigrate(context.TODO(), db.DbMap)
	require.NotNil(t, err)
	require.Contains(t, err.Error(), "You must assign organization to all users before upgrading CDS to the new version.")
}

func TestGetOrganizationUsersToMigrate_OK(t *testing.T) {

	db, _ := test.SetupPG(t, bootstrap.InitializeDB)

	_, err := db.Exec("DELETE FROM organization")
	require.NoError(t, err)
	_, err = db.Exec("DELETE FROM authentified_user")
	require.NoError(t, err)

	orga1 := sdk.Organization{Name: "orga1"}
	require.NoError(t, organization.Insert(context.TODO(), db, &orga1))

	orga2 := sdk.Organization{Name: "orga2"}
	require.NoError(t, organization.Insert(context.TODO(), db, &orga2))

	for i := 0; i < 10; i++ {
		u := sdk.AuthentifiedUser{
			Username: fmt.Sprintf("user-%d-%s", i, sdk.RandomString(10)),
		}
		require.NoError(t, user.Insert(context.TODO(), db, &u))
		destOrga := orga1.Name
		if i > 4 {
			destOrga = orga2.Name
		}
		require.NoError(t, user.InsertOldUserOrganisation(context.TODO(), db, u.ID, destOrga))
	}

	users, err := GetOrganizationUsersToMigrate(context.TODO(), db.DbMap)
	require.NoError(t, err)
	for _, u := range users {
		t.Logf("%s - %s", u.User.Username, u.OrganizationName)
	}
}
