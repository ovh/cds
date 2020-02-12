package group_test

import (
	"context"
	"testing"

	"github.com/ovh/cds/engine/api/bootstrap"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/sdk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDAO_LinkGroupUser(t *testing.T) {
	db, _, end := test.SetupPG(t, bootstrap.InitiliazeDB)
	defer end()

	u1, _ := assets.InsertLambdaUser(t, db)
	u2, _ := assets.InsertLambdaUser(t, db)

	groupName := sdk.RandomString(10)

	err := group.Create(context.TODO(), db, &sdk.Group{
		Name: groupName,
	}, u1.ID)
	require.NoError(t, err)

	grp, err := group.LoadByName(context.TODO(), db, groupName, group.LoadOptions.WithMembers)
	require.NoError(t, err)
	assert.Len(t, grp.Members, 1)

	link := &group.LinkGroupUser{
		GroupID:            grp.ID,
		AuthentifiedUserID: u2.ID,
	}

	err = group.InsertLinkGroupUser(context.TODO(), db, link)
	require.NoError(t, err)

	grp, err = group.LoadByName(context.TODO(), db, groupName, group.LoadOptions.WithMembers)
	require.NoError(t, err)
	assert.Len(t, grp.Members, 2)

	err = group.UpdateLinkGroupUser(context.TODO(), db, link)
	require.NoError(t, err)

	grp, err = group.LoadByName(context.TODO(), db, groupName, group.LoadOptions.WithMembers)
	require.NoError(t, err)
	assert.Len(t, grp.Members, 2)

	err = group.DeleteUserFromGroup(context.TODO(), db, grp.ID, u1.ID)
	require.EqualError(t, err, "TestDAO_LinkGroupUser>DeleteUserFromGroup: not enough group admin left (caused by: not enough group admin left)")

	grp, err = group.LoadByName(context.TODO(), db, groupName, group.LoadOptions.WithMembers)
	require.NoError(t, err)
	assert.Len(t, grp.Members, 2)

	err = group.DeleteUserFromGroup(context.TODO(), db, grp.ID, u2.ID)
	require.NoError(t, err)

	grp, err = group.LoadByName(context.TODO(), db, groupName, group.LoadOptions.WithMembers)
	require.NoError(t, err)
	assert.Len(t, grp.Members, 1)

	links1, err := group.LoadLinksGroupUserForUserIDs(context.TODO(), db, []string{u1.ID})
	require.NoError(t, err)
	assert.Len(t, links1, 1)

	links2, err := group.LoadLinksGroupUserForUserIDs(context.TODO(), db, []string{u2.ID})
	require.NoError(t, err)
	assert.Len(t, links2, 0)

}
