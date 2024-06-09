package notification

import (
	"context"
	"testing"

	"github.com/ovh/cds/engine/api/bootstrap"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/sdk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test_projectPermissionUserIDs test the usernames selected to send notifications
func Test_projectPermissionUserIDs(t *testing.T) {
	db, cache := test.SetupPG(t, bootstrap.InitializeDB)

	g1 := assets.InsertTestGroup(t, db, sdk.RandomString(10))
	g2 := assets.InsertTestGroup(t, db, sdk.RandomString(10))
	g3 := assets.InsertTestGroup(t, db, sdk.RandomString(10))

	_, _ = assets.InsertLambdaUser(t, db, g1)
	u2, _ := assets.InsertLambdaUser(t, db, g2)

	pkey := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, cache, pkey, pkey)

	require.NoError(t, group.InsertLinkGroupProject(context.TODO(), db, &group.LinkGroupProject{
		GroupID:   g2.ID,
		ProjectID: proj.ID,
		Role:      sdk.PermissionReadWriteExecute,
	}))

	require.NoError(t, group.InsertLinkGroupProject(context.TODO(), db, &group.LinkGroupProject{
		GroupID:   g3.ID,
		ProjectID: proj.ID,
		Role:      sdk.PermissionReadWriteExecute,
	}))

	require.NoError(t, group.InsertLinkGroupUser(context.TODO(), db, &group.LinkGroupUser{
		GroupID:            g3.ID,
		AuthentifiedUserID: u2.ID,
		Admin:              false,
	}), "unable to insert user in group")

	assert.NoError(t, project.Update(db, proj))

	group.DefaultGroup = g1

	userList, err := projectPermissionUserIDs(context.Background(), db, cache, proj.ID, sdk.PermissionRead)
	assert.NoError(t, err)
	assert.NotEmpty(t, userList)
	assert.Equal(t, 1, len(userList))
	assert.Equal(t, u2.ID, userList[0], "Only user 2 have to be here. u1, in the default group only should not be here.")
}
