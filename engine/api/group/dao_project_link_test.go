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

func Test_DAO_Project_Link(t *testing.T) {
	db, cache, end := test.SetupPG(t, bootstrap.InitiliazeDB)
	defer end()

	pkey := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, cache, pkey, pkey)

	u, _ := assets.InsertLambdaUser(t, db)
	groupName := sdk.RandomString(10)

	err := group.Create(context.TODO(), db, &sdk.Group{
		Name: groupName,
	}, u.ID)
	require.NoError(t, err)
	grp, err := group.LoadByName(context.TODO(), db, groupName, group.LoadOptions.WithMembers)
	require.NoError(t, err)

	require.NoError(t, group.InsertLinkGroupProject(context.TODO(), db, &group.LinkGroupProject{
		GroupID:   grp.ID,
		ProjectID: proj.ID,
		Role:      sdk.PermissionReadWriteExecute,
	}))

	links, err := group.LoadLinksGroupProjectForGroupID(context.TODO(), db, grp.ID)
	require.NoError(t, err)
	require.Len(t, links, 1)
	assert.Equal(t, sdk.PermissionReadWriteExecute, links[0].Role)

	l := links[0]
	l.Role = sdk.PermissionRead

	err = group.UpdateLinkGroupProject(db, &l)
	require.NoError(t, err)

	links, err = group.LoadLinksGroupProjectForProjectIDs(context.TODO(), db, []int64{proj.ID})
	require.NoError(t, err)
	require.Len(t, links, 2)

	link, err := group.LoadLinkGroupProjectForGroupIDAndProjectID(context.TODO(), db, grp.ID, proj.ID)
	require.NoError(t, err)

	assert.Equal(t, sdk.PermissionRead, link.Role)
	require.NoError(t, group.DeleteLinkGroupProject(db, link))
}
