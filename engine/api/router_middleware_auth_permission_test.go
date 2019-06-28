package api

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ovh/cds/engine/api/authentication/local"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/permission"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/sdk"
)

func TestAPI_checkWorkflowPermissions(t *testing.T) {
	api, _, _, end := newTestAPI(t)
	defer end()

	wctx := testRunWorkflow(t, api, api.Router)

	consumer, err := local.NewConsumer(api.mustDB(), wctx.user.ID, sdk.RandomString(20))
	require.NoError(t, err)

	consumer.AuthentifiedUser = wctx.user

	ctx := context.Background()

	// test case: has enough permission
	ctx = context.WithValue(ctx, contextAPIConsumer, consumer)
	err = api.checkWorkflowPermissions(ctx, wctx.workflow.Name, permission.PermissionReadWriteExecute, map[string]string{
		"key":              wctx.project.Key,
		"permWorkflowName": wctx.workflow.Name,
	})
	assert.NoError(t, err, "should be granted because has permission (max permission = 7)")

	// test case: is Admin
	consumer.GroupIDs = nil
	consumer.AuthentifiedUser.OldUserStruct.Groups = nil
	ctx = context.WithValue(ctx, contextAPIConsumer, consumer)
	err = api.checkWorkflowPermissions(ctx, wctx.workflow.Name, permission.PermissionReadWriteExecute, map[string]string{
		"key":              wctx.project.Key,
		"permWorkflowName": wctx.workflow.Name,
	})
	assert.NoError(t, err, "should be granted because because is admin")

	// test case: is Maintaner
	consumer.GroupIDs = nil
	consumer.AuthentifiedUser.OldUserStruct.Groups = nil
	ctx = context.WithValue(ctx, contextAPIConsumer, consumer)
	err = api.checkWorkflowPermissions(ctx, wctx.workflow.Name, permission.PermissionRead, map[string]string{
		"key":              wctx.project.Key,
		"permWorkflowName": wctx.workflow.Name,
	})
	assert.NoError(t, err, "should be granted because because is maintainer")

	// test case: forbidden
	consumer.GroupIDs = nil
	consumer.AuthentifiedUser.OldUserStruct.Groups = nil
	consumer.AuthentifiedUser.Ring = ""
	ctx = context.WithValue(ctx, contextAPIConsumer, consumer)
	err = api.checkWorkflowPermissions(ctx, wctx.workflow.Name, permission.PermissionRead, map[string]string{
		"key":              wctx.project.Key,
		"permWorkflowName": wctx.workflow.Name,
	})
	assert.Error(t, err, "should not be granted")
}

func TestAPI_checkProjectPermissions(t *testing.T) {
	api, _, _, end := newTestAPI(t)
	defer end()

	g := assets.InsertGroup(t, api.mustDB())
	authUser, _ := assets.InsertLambdaUser(api.mustDB(), g)

	p := assets.InsertTestProject(t, api.mustDB(), api.Cache, sdk.RandomString(10), sdk.RandomString(10), authUser)

	require.NoError(t, group.InsertUserInGroup(api.mustDB(), p.ProjectGroups[0].Group.ID, authUser.OldUserStruct.ID, false))

	// Reload the groups for the user
	groups, err := group.LoadGroupByUser(api.mustDB(), authUser.OldUserStruct.ID)
	require.NoError(t, err)
	authUser.OldUserStruct.Groups = groups

	var consumer sdk.AuthConsumer
	consumer.AuthentifiedUser = authUser
	ctx := context.Background()

	// test case: has enough permission
	ctx = context.WithValue(ctx, contextAPIConsumer, consumer)
	err = api.checkProjectPermissions(ctx, p.Key, permission.PermissionReadWriteExecute, nil)
	assert.NoError(t, err, "should be granted because has permission (max permission = 7)")

	// test case: is Admin
	consumer.AuthentifiedUser.Ring = sdk.UserRingAdmin
	consumer.GroupIDs = nil
	consumer.AuthentifiedUser.OldUserStruct.Groups = nil
	ctx = context.WithValue(ctx, contextAPIConsumer, consumer)
	err = api.checkProjectPermissions(ctx, p.Key, permission.PermissionReadWriteExecute, nil)
	assert.NoError(t, err, "should be granted because because is admin")

	// test case: is Maintainer
	consumer.GroupIDs = nil
	consumer.AuthentifiedUser.OldUserStruct.Groups = nil
	ctx = context.WithValue(ctx, contextAPIConsumer, consumer)
	err = api.checkProjectPermissions(ctx, p.Key, permission.PermissionRead, nil)
	assert.NoError(t, err, "should be granted because because is maintainer")

	// test case: forbidden
	consumer.GroupIDs = nil
	consumer.AuthentifiedUser.OldUserStruct.Groups = nil
	consumer.AuthentifiedUser.Ring = ""
	ctx = context.WithValue(ctx, contextAPIConsumer, consumer)
	err = api.checkProjectPermissions(ctx, p.Key, permission.PermissionRead, nil)
	assert.Error(t, err, "should not be granted")
}

func TestAPI_checkUserPermissions(t *testing.T) {
	api, db, _, end := newTestAPI(t)
	defer end()

	authUser, _ := assets.InsertLambdaUser(db)
	authUserAdmin, _ := assets.InsertAdminUser(db)

	ctx := context.WithValue(context.TODO(), contextAPIConsumer, &sdk.AuthConsumer{
		AuthentifiedUser: authUser,
	})
	err := api.checkUserPermissions(ctx, authUser.Username, permission.PermissionReadWriteExecute, nil)
	assert.NoError(t, err, "should be granted")

	ctx = context.WithValue(context.TODO(), contextAPIConsumer, &sdk.AuthConsumer{
		AuthentifiedUser: authUser,
	})
	err = api.checkUserPermissions(ctx, authUserAdmin.Username, permission.PermissionRead, nil)
	assert.Error(t, err, "should not be granted")

	ctx = context.WithValue(context.TODO(), contextAPIConsumer, &sdk.AuthConsumer{
		AuthentifiedUser: authUserAdmin,
	})
	err = api.checkUserPermissions(ctx, authUser.Username, permission.PermissionRead, nil)
	assert.NoError(t, err, "should be granted")

	ctx = context.WithValue(context.TODO(), contextAPIConsumer, &sdk.AuthConsumer{
		AuthentifiedUser: authUserAdmin,
	})
	err = api.checkUserPermissions(ctx, authUser.Username, permission.PermissionReadWriteExecute, nil)
	assert.Error(t, err, "should not be granted")

	ctx = context.WithValue(context.TODO(), contextAPIConsumer, &sdk.AuthConsumer{
		AuthentifiedUser: authUserAdmin,
	})
	err = api.checkUserPermissions(ctx, authUserAdmin.Username, permission.PermissionReadWriteExecute, nil)
	assert.NoError(t, err, "should be granted")
}
