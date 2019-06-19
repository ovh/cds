package api

import (
	"context"
	"testing"

	"github.com/ovh/cds/engine/api/permission"
	"github.com/stretchr/testify/assert"

	"github.com/ovh/cds/engine/api/authentication"
	"github.com/ovh/cds/sdk"
)

func TestAPI_checkWorkflowPermissions(t *testing.T) {
	api, _, _, end := newTestAPI(t)
	defer end()

	wctx := testRunWorkflow(t, api, api.Router)

	consumer, err := authentication.NewConsumerBuiltin(api.mustDB(), "Test consumer for user "+wctx.user.Username, "", wctx.user.ID, nil, []string{sdk.AccessTokenScopeALL})
	if err != nil {
		t.Fatal(err)
	}
	consumer.AuthentifiedUser = wctx.user

	ctx := context.Background()

	// test case: has enough permission
	ctx = context.WithValue(ctx, contextAPIConsumer, consumer)
	err = api.checkWorkflowPermissions(ctx, wctx.workflow.Name, permission.PermissionReadWriteExecute, map[string]string{
		"key":              wctx.project.Key,
		"permWorkflowName": wctx.workflow.Name,
	})
	assert.NoError(t, err, "should be granted because has permission (max permission = 7) ")

	// test case: is Admin
	consumer.GroupIDs = nil
	consumer.AuthentifiedUser.OldUserStruct.Groups = nil
	ctx = context.WithValue(ctx, contextAPIConsumer, consumer)
	err = api.checkWorkflowPermissions(ctx, wctx.workflow.Name, permission.PermissionReadWriteExecute, map[string]string{
		"key":              wctx.project.Key,
		"permWorkflowName": wctx.workflow.Name,
	})
	assert.NoError(t, err, "should be granted because because is admin ")

	// test case: is Admin
	consumer.GroupIDs = nil
	consumer.AuthentifiedUser.OldUserStruct.Groups = nil
	ctx = context.WithValue(ctx, contextAPIConsumer, consumer)
	err = api.checkWorkflowPermissions(ctx, wctx.workflow.Name, permission.PermissionRead, map[string]string{
		"key":              wctx.project.Key,
		"permWorkflowName": wctx.workflow.Name,
	})
	assert.NoError(t, err, "should be granted because because is maintainer ")

	// test case: forbidden
	consumer.GroupIDs = nil
	consumer.AuthentifiedUser.OldUserStruct.Groups = nil
	consumer.AuthentifiedUser.Ring = ""
	ctx = context.WithValue(ctx, contextAPIConsumer, consumer)
	err = api.checkWorkflowPermissions(ctx, wctx.workflow.Name, permission.PermissionRead, map[string]string{
		"key":              wctx.project.Key,
		"permWorkflowName": wctx.workflow.Name,
	})
	assert.Error(t, err, "should not be granted ")
}
