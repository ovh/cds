package group_test

import (
	"context"
	"testing"

	"github.com/ovh/cds/engine/api/organization"
	"github.com/ovh/cds/engine/api/test/assets"

	"github.com/stretchr/testify/require"

	"github.com/ovh/cds/engine/api/bootstrap"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/sdk"
)

func TestDAO_GroupOrganization(t *testing.T) {
	db, _ := test.SetupPG(t, bootstrap.InitializeDB)

	_, err := db.Exec("DELETE FROM organization")
	require.NoError(t, err)

	g := assets.InsertTestGroupInOrganization(t, db, sdk.RandomString(10), "one")

	orgaTwo := &sdk.Organization{Name: "two"}
	require.NoError(t, organization.Insert(context.TODO(), db, orgaTwo))

	grp, err := group.LoadByID(context.TODO(), db, g.ID, group.LoadOptions.WithOrganization)
	require.NoError(t, err)
	require.NotNil(t, grp)
	require.Equal(t, "one", grp.Organization)

	grpOrga, err := group.LoadGroupOrganizationByGroupID(context.TODO(), db, grp.ID)
	require.NoError(t, err)
	grpOrga.OrganizationID = orgaTwo.ID
	require.NoError(t, group.UpdateGroupOrganization(context.TODO(), db, grpOrga))

	grp, err = group.LoadByID(context.TODO(), db, g.ID, group.LoadOptions.WithOrganization)
	require.NoError(t, err)
	require.NotNil(t, grp)
	require.Equal(t, "two", grp.Organization)
}
