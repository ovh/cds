package group_test

import (
	"context"
	"github.com/ovh/cds/engine/api/organization"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ovh/cds/engine/api/bootstrap"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/sdk"
)

func TestDAO_GroupOrganization(t *testing.T) {
	db, _ := test.SetupPG(t, bootstrap.InitiliazeDB)

	g := &sdk.Group{Name: sdk.RandomString(10)}
	require.NoError(t, group.Insert(context.TODO(), db, g))

	_, err := group.LoadGroupOrganizationByGroupID(context.TODO(), db, g.ID)
	require.Error(t, err)

	orgaOne := sdk.Organization{
		Name: "one",
	}
	require.NoError(t, organization.Insert(context.TODO(), db, &orgaOne))

	orgaTwo := sdk.Organization{
		Name: "two",
	}
	require.NoError(t, organization.Insert(context.TODO(), db, &orgaTwo))

	grpOrga := &group.Organization{
		GroupID:        g.ID,
		OrganizationID: orgaOne.ID,
	}
	require.NoError(t, group.InsertGroupOrganization(context.TODO(), db, grpOrga))

	grp, err := group.LoadByID(context.TODO(), db, g.ID, group.LoadOptions.WithOrganization)
	require.NoError(t, err)
	require.NotNil(t, grp)
	require.Equal(t, "one", grp.Organization)

	grpOrga.OrganizationID = orgaTwo.ID
	require.NoError(t, group.UpdateGroupOrganization(context.TODO(), db, grpOrga))

	grp, err = group.LoadByID(context.TODO(), db, g.ID, group.LoadOptions.WithOrganization)
	require.NoError(t, err)
	require.NotNil(t, grp)
	require.Equal(t, "two", grp.Organization)
}
