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

func TestLoadAll(t *testing.T) {
	db, _, end := test.SetupPG(t, bootstrap.InitiliazeDB)
	defer end()

	u, _ := assets.InsertLambdaUser(t, db)
	groupName := sdk.RandomString(10)

	err := group.Create(context.TODO(), db, &sdk.Group{
		Name: groupName,
	}, u.ID)
	require.NoError(t, err)
	grp, err := group.LoadByName(context.TODO(), db, groupName, group.LoadOptions.WithMembers)
	require.NoError(t, err)
	require.NotNil(t, grp)

	grps, err := group.LoadAll(context.TODO(), db, group.LoadOptions.WithMembers)
	require.NoError(t, err)
	require.NotEmpty(t, grps)

	grps2, err := group.LoadAllByIDs(context.TODO(), db, grps.ToIDs(), group.LoadOptions.WithMembers)
	require.NoError(t, err)
	assert.Len(t, grps2, len(grps))

	grps, err = group.LoadAllByUserID(context.TODO(), db, u.ID)
	require.NoError(t, err)
	require.Len(t, grps, 1)

	grp, err = group.LoadByID(context.TODO(), db, grp.ID, group.LoadOptions.WithMembers)
	require.NoError(t, err)
	require.Len(t, grp.Members, 1)

	require.NoError(t, group.Update(context.TODO(), db, grp))

	groupNameDefault := sdk.RandomString(10)
	require.NoError(t, group.CreateDefaultGroup(db, groupNameDefault))
	group.DefaultGroup, _ = group.LoadByName(context.TODO(), db, groupNameDefault)

	require.NoError(t, group.CheckUserInDefaultGroup(context.TODO(), db, u.ID))
}
