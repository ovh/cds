package group_test

import (
	"context"
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

	_, err := group.LoadOrganizationByGroupID(context.TODO(), db, g.ID)
	require.Error(t, err)

	require.NoError(t, group.InsertOrganization(context.TODO(), db, &group.Organization{
		GroupID:      g.ID,
		Organization: "one",
	}))

	org, err := group.LoadOrganizationByGroupID(context.TODO(), db, g.ID)
	require.NoError(t, err)
	require.NotNil(t, org)
	require.Equal(t, "one", org.Organization)

	org.Organization = "two"
	require.NoError(t, group.UpdateOrganization(context.TODO(), db, org))

	org, err = group.LoadOrganizationByGroupID(context.TODO(), db, g.ID)
	require.NoError(t, err)
	require.NotNil(t, org)
	require.Equal(t, "two", org.Organization)
}
