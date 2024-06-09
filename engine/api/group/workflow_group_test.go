package group_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ovh/cds/engine/api/authentication"
	"github.com/ovh/cds/engine/api/bootstrap"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/sdk"
)

func TestCheckWorkflowGroups_UserShouldBeGroupAdminForRWAndRWX(t *testing.T) {
	db, cache := test.SetupPG(t, bootstrap.InitializeDB)

	proj := assets.InsertTestProject(t, db, cache, sdk.RandomString(10), sdk.RandomString(10))

	g1 := proj.ProjectGroups[0].Group
	g2 := assets.InsertTestGroup(t, db, sdk.RandomString(10))
	g3 := assets.InsertTestGroup(t, db, sdk.RandomString(10))

	u, _ := assets.InsertLambdaUser(t, db, &g1, g2)
	assets.SetUserGroupAdmin(t, db, g1.ID, u.ID)
	assets.SetUserGroupAdmin(t, db, g2.ID, u.ID)

	localConsumer, err := authentication.LoadUserConsumerByTypeAndUserID(context.TODO(), db, sdk.ConsumerLocal, u.ID, authentication.LoadUserConsumerOptions.WithAuthentifiedUser)
	require.NoError(t, err)

	// Insert workflow that will inherit from project permission
	w := assets.InsertTestWorkflow(t, db, cache, proj, sdk.RandomString(10))

	// Set g2 and g3 on project
	require.NoError(t, group.InsertLinkGroupProject(context.TODO(), db, &group.LinkGroupProject{
		GroupID:   g2.ID,
		ProjectID: proj.ID,
		Role:      sdk.PermissionReadWriteExecute,
	}))
	require.NoError(t, group.InsertLinkGroupProject(context.TODO(), db, &group.LinkGroupProject{
		GroupID:   g3.ID,
		ProjectID: proj.ID,
		Role:      sdk.PermissionRead,
	}))

	// User cannot add RX permission for g3 on workflow because not admin of g3
	w.Groups = append(w.Groups, sdk.GroupPermission{
		Permission: sdk.PermissionReadExecute,
		Group:      *g3,
	})
	err = group.CheckWorkflowGroups(context.TODO(), db, proj, w, localConsumer)
	require.Error(t, err)
	require.Equal(t, "User is not a group's admin (from: cannot set permission with level 5 for group \""+g3.Name+"\")", err.Error())

	// User can add RX permission for g2 on workflow because admin of g2
	w.Groups = w.Groups[0 : len(w.Groups)-2]
	w.Groups = append(w.Groups, sdk.GroupPermission{
		Permission: sdk.PermissionReadWriteExecute,
		Group:      *g2,
	})
	require.NoError(t, group.CheckWorkflowGroups(context.TODO(), db, proj, w, localConsumer))

	// User can add R permission for g3 on workflow
	w.Groups = append(w.Groups, sdk.GroupPermission{
		Permission: sdk.PermissionRead,
		Group:      *g3,
	})
	require.NoError(t, group.CheckWorkflowGroups(context.TODO(), db, proj, w, localConsumer))
}

func TestCheckWorkflowGroups_OnlyReadForDifferentOrganization(t *testing.T) {
	db, cache := test.SetupPG(t, bootstrap.InitializeDB)

	proj := assets.InsertTestProject(t, db, cache, sdk.RandomString(10), sdk.RandomString(10))

	g2 := assets.InsertTestGroupInOrganization(t, db, sdk.RandomString(10), "two")

	u, _ := assets.InsertAdminUser(t, db)

	localConsumer, err := authentication.LoadUserConsumerByTypeAndUserID(context.TODO(), db, sdk.ConsumerLocal, u.ID, authentication.LoadUserConsumerOptions.WithAuthentifiedUser)
	require.NoError(t, err)

	// Insert workflow that will inherit from project permission
	w := assets.InsertTestWorkflow(t, db, cache, proj, sdk.RandomString(10))

	// Set g2 on project
	require.NoError(t, group.InsertLinkGroupProject(context.TODO(), db, &group.LinkGroupProject{
		GroupID:   g2.ID,
		ProjectID: proj.ID,
		Role:      sdk.PermissionRead,
	}))

	// Cannot add RX permission for g2 on workflow because organization is not the same as project's one
	w.Groups = sdk.GroupPermissions{{
		Permission: sdk.PermissionReadExecute,
		Group:      *g2,
	}}
	err = group.CheckWorkflowGroups(context.TODO(), db, proj, w, localConsumer)
	require.Error(t, err)
	require.Equal(t, "forbidden (from: given group with organization \"two\" don't match project organization \"default\")", err.Error(), err)

	// Can add R permission for g2 on workflow
	w.Groups = sdk.GroupPermissions{{
		Permission: sdk.PermissionRead,
		Group:      *g2,
	}}
	require.NoError(t, group.CheckWorkflowGroups(context.TODO(), db, proj, w, localConsumer))
}

func TestCheckWorkflowGroups_UserShouldBeGroupAdminForRWAndRWX_Node(t *testing.T) {
	db, cache := test.SetupPG(t, bootstrap.InitializeDB)

	proj := assets.InsertTestProject(t, db, cache, sdk.RandomString(10), sdk.RandomString(10))

	g1 := proj.ProjectGroups[0].Group
	g2 := assets.InsertTestGroup(t, db, sdk.RandomString(10))
	g3 := assets.InsertTestGroup(t, db, sdk.RandomString(10))

	u, _ := assets.InsertLambdaUser(t, db, &g1, g2)
	assets.SetUserGroupAdmin(t, db, g1.ID, u.ID)
	assets.SetUserGroupAdmin(t, db, g2.ID, u.ID)

	localConsumer, err := authentication.LoadUserConsumerByTypeAndUserID(context.TODO(), db, sdk.ConsumerLocal, u.ID, authentication.LoadUserConsumerOptions.WithAuthentifiedUser)
	require.NoError(t, err)

	// Insert workflow that will inherit from project permission
	w := assets.InsertTestWorkflow(t, db, cache, proj, sdk.RandomString(10))

	// Set g2 and g3 on project
	require.NoError(t, group.InsertLinkGroupProject(context.TODO(), db, &group.LinkGroupProject{
		GroupID:   g2.ID,
		ProjectID: proj.ID,
		Role:      sdk.PermissionReadWriteExecute,
	}))
	require.NoError(t, group.InsertLinkGroupProject(context.TODO(), db, &group.LinkGroupProject{
		GroupID:   g3.ID,
		ProjectID: proj.ID,
		Role:      sdk.PermissionRead,
	}))

	// User cannot add RX permission for g3 on workflow node because not admin of g3
	w.Groups = nil
	w.WorkflowData.Node.Groups = sdk.GroupPermissions{{
		Permission: sdk.PermissionReadExecute,
		Group:      *g3,
	}}
	err = group.CheckWorkflowGroups(context.TODO(), db, proj, w, localConsumer)
	require.Error(t, err)
	require.Equal(t, "User is not a group's admin (from: cannot set permission with level 5 for group \""+g3.Name+"\")", err.Error())

	// User can add RX permission for g2 on workflow node because admin of g2
	w.Groups = nil
	w.WorkflowData.Node.Groups = sdk.GroupPermissions{{
		Permission: sdk.PermissionReadExecute,
		Group:      *g2,
	}}
	require.NoError(t, group.CheckWorkflowGroups(context.TODO(), db, proj, w, localConsumer))

	// User can add R permission for g3 on workflow node
	w.Groups = nil
	w.WorkflowData.Node.Groups = sdk.GroupPermissions{{
		Permission: sdk.PermissionRead,
		Group:      *g3,
	}}
	require.NoError(t, group.CheckWorkflowGroups(context.TODO(), db, proj, w, localConsumer))
}

func TestCheckWorkflowGroups_OnlyReadForDifferentOrganization_Node(t *testing.T) {
	db, cache := test.SetupPG(t, bootstrap.InitializeDB)

	proj := assets.InsertTestProject(t, db, cache, sdk.RandomString(10), sdk.RandomString(10))

	g2 := assets.InsertTestGroupInOrganization(t, db, sdk.RandomString(10), "two")

	u, _ := assets.InsertAdminUser(t, db)

	localConsumer, err := authentication.LoadUserConsumerByTypeAndUserID(context.TODO(), db, sdk.ConsumerLocal, u.ID, authentication.LoadUserConsumerOptions.WithAuthentifiedUser)
	require.NoError(t, err)

	// Insert workflow that will inherit from project permission
	w := assets.InsertTestWorkflow(t, db, cache, proj, sdk.RandomString(10))

	// Set g2 on project
	require.NoError(t, group.InsertLinkGroupProject(context.TODO(), db, &group.LinkGroupProject{
		GroupID:   g2.ID,
		ProjectID: proj.ID,
		Role:      sdk.PermissionRead,
	}))

	// Cannot add RX permission for g2 on workflow node because organization is not the same as project's one
	w.Groups = nil
	w.WorkflowData.Node.Groups = sdk.GroupPermissions{{
		Permission: sdk.PermissionReadExecute,
		Group:      *g2,
	}}
	err = group.CheckWorkflowGroups(context.TODO(), db, proj, w, localConsumer)
	require.Error(t, err)
	require.Equal(t, "forbidden (from: given group with organization \"two\" don't match project organization \"default\")", err.Error(), err)

	// Can add R permission for g2 on workflow node
	w.Groups = nil
	w.WorkflowData.Node.Groups = sdk.GroupPermissions{{
		Permission: sdk.PermissionRead,
		Group:      *g2,
	}}
	require.NoError(t, group.CheckWorkflowGroups(context.TODO(), db, proj, w, localConsumer))
}
