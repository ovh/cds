package group_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ovh/cds/engine/api/bootstrap"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/sdk"
)

func TestDAO_LinkGroupUser(t *testing.T) {
	db, _ := test.SetupPG(t, bootstrap.InitializeDB)

	u1, _ := assets.InsertLambdaUser(t, db)
	u2, _ := assets.InsertLambdaUser(t, db)

	groupName := sdk.RandomString(10)

	require.NoError(t, group.Create(context.TODO(), db, &sdk.Group{
		Name: groupName,
	}, u1))

	grp, err := group.LoadByName(context.TODO(), db, groupName, group.LoadOptions.WithMembers)
	require.NoError(t, err)
	assert.Len(t, grp.Members, 1)

	link := &group.LinkGroupUser{
		GroupID:            grp.ID,
		AuthentifiedUserID: u2.ID,
	}
	require.NoError(t, group.InsertLinkGroupUser(context.TODO(), db, link))

	grp, err = group.LoadByName(context.TODO(), db, groupName, group.LoadOptions.WithMembers)
	require.NoError(t, err)
	assert.Len(t, grp.Members, 2)

	var m1, m2 *sdk.GroupMember
	for i := range grp.Members {
		if grp.Members[i].ID == u1.ID {
			m1 = &grp.Members[i]
		}
		if grp.Members[i].ID == u2.ID {
			m2 = &grp.Members[i]
		}
	}
	require.NotNil(t, m1)
	require.True(t, m1.Admin)
	require.NotNil(t, m2)
	require.False(t, m2.Admin)

	link.Admin = true
	require.NoError(t, group.UpdateLinkGroupUser(context.TODO(), db, link))

	grp, err = group.LoadByName(context.TODO(), db, groupName, group.LoadOptions.WithMembers)
	require.NoError(t, err)
	assert.Len(t, grp.Members, 2)

	m1, m2 = nil, nil
	for i := range grp.Members {
		if grp.Members[i].ID == u1.ID {
			m1 = &grp.Members[i]
		}
		if grp.Members[i].ID == u2.ID {
			m2 = &grp.Members[i]
		}
	}
	require.NotNil(t, m1)
	require.True(t, m1.Admin)
	require.NotNil(t, m2)
	require.True(t, m2.Admin)

	links, err := group.LoadLinksGroupUserForUserIDs(context.TODO(), db, []string{u1.ID, u2.ID})
	require.NoError(t, err)
	assert.Len(t, links, 2)
}
