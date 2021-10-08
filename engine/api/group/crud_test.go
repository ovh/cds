package group_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ovh/cds/engine/api/bootstrap"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/engine/api/user"
	"github.com/ovh/cds/sdk"
)

func Test_CreateUpdateDelete(t *testing.T) {
	db, _ := test.SetupPG(t, bootstrap.InitiliazeDB)

	u1, _ := assets.InsertLambdaUser(t, db)
	u1.Organization = "one"
	require.NoError(t, user.InsertOrganization(context.TODO(), db, &user.Organization{
		AuthentifiedUserID: u1.ID,
		Organization:       u1.Organization,
	}))
	u2, _ := assets.InsertLambdaUser(t, db)
	u3, _ := assets.InsertLambdaUser(t, db)
	u3.Organization = "two"
	require.NoError(t, user.InsertOrganization(context.TODO(), db, &user.Organization{
		AuthentifiedUserID: u3.ID,
		Organization:       u3.Organization,
	}))

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
	grp, err = group.LoadByName(context.TODO(), db, groupName, group.LoadOptions.Default)
	require.NoError(t, err)
	require.Len(t, grp.Members, 2)
	require.Equal(t, "one", grp.Organization)

	require.NoError(t, group.Upsert(context.TODO(), db, grp, &sdk.Group{
		ID:   grp.ID,
		Name: grp.Name,
		Members: []sdk.GroupMember{
			{ID: u2.ID, Admin: true, Organization: u2.Organization},
			{ID: u3.ID, Organization: u3.Organization},
		},
	}))
	grp, err = group.LoadByName(context.TODO(), db, groupName, group.LoadOptions.Default)
	require.NoError(t, err)
	require.Len(t, grp.Members, 2)
	require.Equal(t, "one", grp.Organization)

	require.NoError(t, group.EnsureOrganization(context.TODO(), db, grp))
	require.Equal(t, "two", grp.Organization)

	require.NoError(t, group.Upsert(context.TODO(), db, grp, &sdk.Group{
		ID:   grp.ID,
		Name: grp.Name,
		Members: []sdk.GroupMember{
			{ID: u2.ID, Admin: true, Organization: u2.Organization},
		},
	}))
	grp, err = group.LoadByName(context.TODO(), db, groupName, group.LoadOptions.Default)
	require.NoError(t, err)
	require.Len(t, grp.Members, 1)
	require.Equal(t, "two", grp.Organization)

	require.NoError(t, group.EnsureOrganization(context.TODO(), db, grp))
	require.Equal(t, "", grp.Organization)

	// Missing group admin should raise an error
	err = group.Upsert(context.TODO(), db, grp, &sdk.Group{
		ID:   grp.ID,
		Name: grp.Name,
		Members: []sdk.GroupMember{
			{ID: u1.ID, Organization: u1.Organization},
			{ID: u2.ID, Organization: u2.Organization},
		},
	})
	require.Error(t, err)
	require.Equal(t, "Cannot validate given data (from: invalid given group members, at least one admin required)", err.Error())

	// Conflict user orgs should raise an error
	require.NoError(t, group.Upsert(context.TODO(), db, grp, &sdk.Group{
		ID:   grp.ID,
		Name: grp.Name,
		Members: []sdk.GroupMember{
			{ID: u1.ID, Admin: true, Organization: u1.Organization},
			{ID: u3.ID, Organization: u3.Organization},
		},
	}))
	err = group.EnsureOrganization(context.TODO(), db, grp)
	require.Error(t, err)
	require.Equal(t, "Cannot validate given data (from: group members organization conflict \"one\" and \"two\")", err.Error())

	// Delete the group
	require.NoError(t, group.Delete(context.TODO(), db, grp))
	_, err = group.LoadByName(context.TODO(), db, groupName)
	require.Error(t, err)
}
