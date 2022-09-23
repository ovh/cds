package group_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ovh/cds/engine/api/bootstrap"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/sdk"
)

func Test_CreateUpdateDelete(t *testing.T) {
	db, _ := test.SetupPG(t, bootstrap.InitiliazeDB)

	u1, _ := assets.InsertLambdaUserInOrganization(t, db, "one")
	u2, _ := assets.InsertLambdaUser(t, db)
	u3, _ := assets.InsertLambdaUserInOrganization(t, db, "one")

	// Create the group
	groupName := sdk.RandomString(10)
	require.NoError(t, group.Create(context.TODO(), db, &sdk.Group{Name: groupName}, u1))
	grp, err := group.LoadByName(context.TODO(), db, groupName, group.LoadOptions.Default)
	require.NoError(t, err)
	require.Len(t, grp.Members, 1)
	require.Equal(t, "one", grp.Organization)

	// Update members
	require.NoError(t, group.Upsert(context.TODO(), db, grp, &sdk.Group{
		ID:   grp.ID,
		Name: grp.Name,
		Members: []sdk.GroupMember{
			{ID: u1.ID, Admin: true, Organization: u1.Organization},
			{ID: u2.ID, Organization: u2.Organization},
		},
	}))
	err = group.EnsureOrganization(context.TODO(), db, grp)
	require.Error(t, err)
	require.Equal(t, "Cannot validate given data (from: group members organization conflict \"one\" and \"default\")", err.Error())

	require.NoError(t, group.Upsert(context.TODO(), db, grp, &sdk.Group{
		ID:   grp.ID,
		Name: grp.Name,
		Members: []sdk.GroupMember{
			{ID: u1.ID, Admin: true, Organization: u1.Organization},
			{ID: u3.ID, Organization: u3.Organization},
		},
	}))
	require.NoError(t, group.EnsureOrganization(context.TODO(), db, grp))

	// Delete the group
	require.NoError(t, group.Delete(context.TODO(), db, grp))
	_, err = group.LoadByName(context.TODO(), db, groupName)
	require.Error(t, err)
}
