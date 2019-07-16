package api

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ovh/cds/engine/api/authentication"
	"github.com/ovh/cds/engine/api/authentication/local"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/sdk"
)

func TestAPI_checkWorkflowPermissions(t *testing.T) {
	api, _, _, end := newTestAPI(t)
	defer end()

	wctx := testRunWorkflow(t, api, api.Router)

	consumer, err := local.NewConsumer(api.mustDB(), wctx.user.ID)
	require.NoError(t, err)

	consumer.AuthentifiedUser = wctx.user

	ctx := context.Background()

	// test case: has enough permission
	ctx = context.WithValue(ctx, contextAPIConsumer, consumer)
	err = api.checkWorkflowPermissions(ctx, wctx.workflow.Name, sdk.PermissionReadWriteExecute, map[string]string{
		"key":              wctx.project.Key,
		"permWorkflowName": wctx.workflow.Name,
	})
	assert.NoError(t, err, "should be granted because has permission (max permission = 7)")

	// test case: is Admin
	consumer.GroupIDs = nil
	consumer.AuthentifiedUser.OldUserStruct.Groups = nil
	ctx = context.WithValue(ctx, contextAPIConsumer, consumer)
	err = api.checkWorkflowPermissions(ctx, wctx.workflow.Name, sdk.PermissionReadWriteExecute, map[string]string{
		"key":              wctx.project.Key,
		"permWorkflowName": wctx.workflow.Name,
	})
	assert.NoError(t, err, "should be granted because because is admin")

	// test case: is Maintaner
	consumer.GroupIDs = nil
	consumer.AuthentifiedUser.OldUserStruct.Groups = nil
	ctx = context.WithValue(ctx, contextAPIConsumer, consumer)
	err = api.checkWorkflowPermissions(ctx, wctx.workflow.Name, sdk.PermissionRead, map[string]string{
		"key":              wctx.project.Key,
		"permWorkflowName": wctx.workflow.Name,
	})
	assert.NoError(t, err, "should be granted because because is maintainer")

	// test case: forbidden
	consumer.GroupIDs = nil
	consumer.AuthentifiedUser.OldUserStruct.Groups = nil
	consumer.AuthentifiedUser.Ring = ""
	ctx = context.WithValue(ctx, contextAPIConsumer, consumer)
	err = api.checkWorkflowPermissions(ctx, wctx.workflow.Name, sdk.PermissionRead, map[string]string{
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
	ctx = context.WithValue(ctx, contextAPIConsumer, &consumer)
	err = api.checkProjectPermissions(ctx, p.Key, sdk.PermissionReadWriteExecute, nil)
	assert.NoError(t, err, "should be granted because has permission (max permission = 7)")

	// test case: is Admin
	consumer.AuthentifiedUser.Ring = sdk.UserRingAdmin
	consumer.GroupIDs = nil
	consumer.AuthentifiedUser.OldUserStruct.Groups = nil
	ctx = context.WithValue(ctx, contextAPIConsumer, &consumer)
	err = api.checkProjectPermissions(ctx, p.Key, sdk.PermissionReadWriteExecute, nil)
	assert.NoError(t, err, "should be granted because because is admin")

	// test case: is Maintainer
	consumer.GroupIDs = nil
	consumer.AuthentifiedUser.OldUserStruct.Groups = nil
	ctx = context.WithValue(ctx, contextAPIConsumer, &consumer)
	err = api.checkProjectPermissions(ctx, p.Key, sdk.PermissionRead, nil)
	assert.NoError(t, err, "should be granted because because is maintainer")

	// test case: forbidden
	consumer.GroupIDs = nil
	consumer.AuthentifiedUser.OldUserStruct.Groups = nil
	consumer.AuthentifiedUser.Ring = ""
	ctx = context.WithValue(ctx, contextAPIConsumer, &consumer)
	err = api.checkProjectPermissions(ctx, p.Key, sdk.PermissionRead, nil)
	assert.Error(t, err, "should not be granted")
}

func TestAPI_checkUserPermissions(t *testing.T) {
	api, db, _, end := newTestAPI(t)
	defer end()

	authUser, _ := assets.InsertLambdaUser(db)
	authUserAdmin, _ := assets.InsertAdminUser(db)

	ctx := context.WithValue(context.TODO(), contextAPIConsumer, &sdk.AuthConsumer{
		AuthentifiedUserID: authUser.ID,
		AuthentifiedUser:   authUser,
	})
	err := api.checkUserPermissions(ctx, authUser.Username, sdk.PermissionReadWriteExecute, nil)
	assert.NoError(t, err, "should be granted")

	ctx = context.WithValue(context.TODO(), contextAPIConsumer, &sdk.AuthConsumer{
		AuthentifiedUser: authUser,
	})
	err = api.checkUserPermissions(ctx, authUserAdmin.Username, sdk.PermissionRead, nil)
	assert.Error(t, err, "should not be granted")

	ctx = context.WithValue(context.TODO(), contextAPIConsumer, &sdk.AuthConsumer{
		AuthentifiedUserID: authUserAdmin.ID,
		AuthentifiedUser:   authUserAdmin,
	})
	err = api.checkUserPermissions(ctx, authUser.Username, sdk.PermissionRead, nil)
	assert.NoError(t, err, "should be granted")

	ctx = context.WithValue(context.TODO(), contextAPIConsumer, &sdk.AuthConsumer{
		AuthentifiedUserID: authUserAdmin.ID,
		AuthentifiedUser:   authUserAdmin,
	})
	err = api.checkUserPermissions(ctx, authUser.Username, sdk.PermissionReadWriteExecute, nil)
	assert.Error(t, err, "should not be granted")

	ctx = context.WithValue(context.TODO(), contextAPIConsumer, &sdk.AuthConsumer{
		AuthentifiedUserID: authUserAdmin.ID,
		AuthentifiedUser:   authUserAdmin,
	})
	err = api.checkUserPermissions(ctx, authUserAdmin.Username, sdk.PermissionReadWriteExecute, nil)
	assert.NoError(t, err, "should be granted")
}

func TestAPI_checkConsumerPermissions(t *testing.T) {
	api, db, _, end := newTestAPI(t)
	defer end()

	authUser, _ := assets.InsertLambdaUser(db)
	localConsumer, err := authentication.LoadConsumerByTypeAndUserID(context.TODO(), db, sdk.ConsumerLocal, authUser.ID)
	require.NoError(t, err)

	authUserAdmin, _ := assets.InsertAdminUser(db)
	localConsumerAdmin, err := authentication.LoadConsumerByTypeAndUserID(context.TODO(), db, sdk.ConsumerLocal, authUserAdmin.ID)
	require.NoError(t, err)

	ctx := context.WithValue(context.TODO(), contextAPIConsumer, localConsumer)
	err = api.checkConsumerPermissions(ctx, localConsumer.ID, sdk.PermissionReadWriteExecute, nil)
	assert.NoError(t, err, "should be granted")

	ctx = context.WithValue(context.TODO(), contextAPIConsumer, localConsumer)
	err = api.checkConsumerPermissions(ctx, authUserAdmin.ID, sdk.PermissionRead, nil)
	assert.Error(t, err, "should not be granted")

	ctx = context.WithValue(context.TODO(), contextAPIConsumer, localConsumerAdmin)
	err = api.checkConsumerPermissions(ctx, localConsumer.ID, sdk.PermissionRead, nil)
	assert.Error(t, err, "should not be granted")
}

func TestAPI_checkSessionPermissions(t *testing.T) {
	api, db, _, end := newTestAPI(t)
	defer end()

	authUser, _ := assets.InsertLambdaUser(db)
	localConsumer, err := authentication.LoadConsumerByTypeAndUserID(context.TODO(), db, sdk.ConsumerLocal, authUser.ID)
	require.NoError(t, err)
	localSession, err := authentication.NewSession(db, localConsumer, time.Second)
	require.NoError(t, err)

	authUserAdmin, _ := assets.InsertAdminUser(db)
	localConsumerAdmin, err := authentication.LoadConsumerByTypeAndUserID(context.TODO(), db, sdk.ConsumerLocal, authUserAdmin.ID)
	require.NoError(t, err)
	localSessionAdmin, err := authentication.NewSession(db, localConsumerAdmin, time.Second)
	require.NoError(t, err)

	ctx := context.WithValue(context.TODO(), contextAPIConsumer, localConsumer)
	err = api.checkSessionPermissions(ctx, localSession.ID, sdk.PermissionReadWriteExecute, nil)
	assert.NoError(t, err, "should be granted")

	ctx = context.WithValue(context.TODO(), contextAPIConsumer, localConsumer)
	err = api.checkSessionPermissions(ctx, localSessionAdmin.ID, sdk.PermissionRead, nil)
	assert.Error(t, err, "should not be granted")

	ctx = context.WithValue(context.TODO(), contextAPIConsumer, localConsumerAdmin)
	err = api.checkSessionPermissions(ctx, localSession.ID, sdk.PermissionRead, nil)
	assert.Error(t, err, "should not be granted")
}

func Test_checkWorkerModelPermissionsByUser(t *testing.T) {
	_, _, _, end := newTestAPI(t)
	defer end()

	type args struct {
		m *sdk.Model
		u *sdk.User
		p int
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "Should return true for admin user",
			args: args{
				m: &sdk.Model{
					GroupID: 1,
				},
				u: &sdk.User{
					Admin: true,
				},
				p: 7,
			},
			want: true,
		},
		{
			name: "Should return true for user who has the right group for getting the model",
			args: args{
				m: &sdk.Model{
					GroupID: 1,
				},
				u: &sdk.User{
					Admin: false,
					Groups: []sdk.Group{
						{
							ID: 1,
						},
					},
				},
				p: 4,
			},
			want: true,
		},
		{
			name: "Should return false for user who has not the right group for updating the model",
			args: args{
				m: &sdk.Model{
					GroupID: 1,
				},
				u: &sdk.User{
					Admin: false,
					Groups: []sdk.Group{
						{
							ID: 1,
						},
					},
				},
				p: 7,
			},
			want: false,
		},
		{
			name: "Should return false for user who has not the right group",
			args: args{
				m: &sdk.Model{
					GroupID: 666,
				},
				u: &sdk.User{
					Admin: false,
					Groups: []sdk.Group{
						{
							ID: 1,
						},
					},
				},
				p: 7,
			},
			want: false,
		},
		{
			name: "Should return true for user who has the right group as admin for updating the model",
			args: args{
				m: &sdk.Model{
					GroupID: 1,
				},
				u: &sdk.User{
					ID:    1,
					Admin: false,
					Groups: []sdk.Group{
						{
							ID: 1,
							Members: []sdk.User{
								{
									ID:         1,
									GroupAdmin: true,
								},
							},
						},
					},
				},
				p: 7,
			},
			want: true,
		},
	}
	for _, tt := range tests {
		authUser := &sdk.AuthentifiedUser{OldUserStruct: tt.args.u}
		if tt.args.u.Admin {
			authUser.Ring = sdk.UserRingAdmin
		}
		// TODO
		//got := api.checkWorkerModelPermissionsByUser(tt.args.m, authUser, tt.args.p)
		//if !reflect.DeepEqual(got, tt.want) {
		//	t.Errorf("%q. checkWorkerModelPermissionsByUser() = %v, want %v", tt.name, got, tt.want)
		//}
	}
}

func Test_checkWorkflowPermissionsByUser(t *testing.T) {
	api, _, _, end := newTestAPI(t)
	defer end()

	type setup struct {
		UserAdmin                bool
		UserGroupNames           []string
		ProjGroupPermissions     map[string]int
		WorkflowGroupPermissions map[string]int
	}
	type args struct {
		wName           string
		pKey            string
		permissionLevel int
	}
	tests := []struct {
		name  string
		setup setup
		args  args
		want  bool
	}{
		{
			name: "Should return true for user [read permission]",
			setup: setup{
				UserGroupNames: []string{"Test_checkWorkflowPermissionsByUser"},
				ProjGroupPermissions: map[string]int{
					"Test_checkWorkflowPermissionsByUser": sdk.PermissionRead,
				},
				WorkflowGroupPermissions: map[string]int{
					"Test_checkWorkflowPermissionsByUser": sdk.PermissionRead,
				},
			},
			args: args{
				wName:           "workflow1",
				pKey:            "key1",
				permissionLevel: 4,
			},
			want: true,
		}, {
			name: "Should return false for user [read permission]",
			setup: setup{
				UserGroupNames:           []string{"Test_checkWorkflowPermissionsByUser"},
				ProjGroupPermissions:     map[string]int{},
				WorkflowGroupPermissions: map[string]int{},
			},
			args: args{
				wName:           "workflow1",
				pKey:            "key2",
				permissionLevel: 4,
			},
			want: false,
		},
		{
			name: "Should return true for user [write permission]",
			setup: setup{
				UserGroupNames: []string{"Test_checkWorkflowPermissionsByUser"},
				ProjGroupPermissions: map[string]int{
					"Test_checkWorkflowPermissionsByUser": sdk.PermissionRead,
				},
				WorkflowGroupPermissions: map[string]int{
					"Test_checkWorkflowPermissionsByUser": sdk.PermissionReadWriteExecute,
				},
			},
			args: args{
				wName:           "workflow1",
				pKey:            "key1",
				permissionLevel: 7,
			},
			want: true,
		},
		{
			name: "Should return false for user [wrong workflow]",
			setup: setup{
				UserGroupNames: []string{"Test_checkWorkflowPermissionsByUser"},
				ProjGroupPermissions: map[string]int{
					"Test_checkWorkflowPermissionsByUser": sdk.PermissionRead,
				},
				WorkflowGroupPermissions: map[string]int{},
			},
			args: args{
				wName:           "workflow1",
				pKey:            "key1",
				permissionLevel: 7,
			},
			want: false,
		},
		{
			name: "Should return false for user [wrong permission]",
			setup: setup{
				UserGroupNames: []string{"Test_checkWorkflowPermissionsByUser"},
				ProjGroupPermissions: map[string]int{
					"Test_checkWorkflowPermissionsByUser": sdk.PermissionRead,
				},
				WorkflowGroupPermissions: map[string]int{
					"Test_checkWorkflowPermissionsByUser": sdk.PermissionRead,
				},
			},
			args: args{
				wName:           "workflow1",
				pKey:            "key1",
				permissionLevel: 7,
			},
			want: false,
		},
		{
			name: "Should return true for user [execution]",
			setup: setup{
				UserGroupNames: []string{"Test_checkWorkflowPermissionsByUser"},
				ProjGroupPermissions: map[string]int{
					"Test_checkWorkflowPermissionsByUser": sdk.PermissionReadExecute,
				},
				WorkflowGroupPermissions: map[string]int{
					"Test_checkWorkflowPermissionsByUser": sdk.PermissionReadExecute,
				},
			},
			args: args{

				wName:           "workflow1",
				pKey:            "key1",
				permissionLevel: 5,
			},
			want: true,
		},
		{
			name: "Should return false for user [execution]",
			setup: setup{
				UserGroupNames: []string{"Test_checkWorkflowPermissionsByUser"},
				ProjGroupPermissions: map[string]int{
					"Test_checkWorkflowPermissionsByUser": sdk.PermissionRead,
				},
				WorkflowGroupPermissions: map[string]int{
					"Test_checkWorkflowPermissionsByUser": sdk.PermissionRead,
				},
			},
			args: args{
				wName:           "workflow1",
				pKey:            "key1",
				permissionLevel: 5,
			},
			want: false,
		},
	}
	for _, tt := range tests {
		var suffix = "-" + sdk.RandomString(10)
		var groups []*sdk.Group
		for _, s := range tt.setup.UserGroupNames {
			groups = append(groups, &sdk.Group{Name: s + suffix})
		}
		var usr *sdk.AuthentifiedUser
		if tt.setup.UserAdmin {
			usr, _ = assets.InsertAdminUser(api.mustDB())
		} else {
			usr, _ = assets.InsertLambdaUser(api.mustDB(), groups...)
		}

		proj := assets.InsertTestProject(t, api.mustDB(), api.Cache, tt.args.pKey, tt.args.pKey, nil)
		wrkflw := assets.InsertTestWorkflow(t, api.mustDB(), api.Cache, proj, tt.args.wName)

		for groupName, permLevel := range tt.setup.ProjGroupPermissions {
			g, err := group.LoadByName(context.TODO(), api.mustDB(), groupName+suffix, group.LoadOptions.WithMembers)
			require.NoError(t, err)

			require.NoError(t, group.InsertGroupInProject(api.mustDB(), proj.ID, g.ID, permLevel))
		}

		for groupName, permLevel := range tt.setup.WorkflowGroupPermissions {
			g, err := group.LoadByName(context.TODO(), api.mustDB(), groupName+suffix, group.LoadOptions.WithMembers)
			require.NoError(t, err)

			require.NoError(t, group.AddWorkflowGroup(api.mustDB(), wrkflw, sdk.GroupPermission{
				Group:      *g,
				Permission: permLevel,
			}))
		}

		cons, err := authentication.LoadConsumerByTypeAndUserID(context.TODO(), api.mustDB(), sdk.ConsumerLocal, usr.ID, authentication.LoadConsumerOptions.WithAuthentifiedUser)
		require.NoError(t, err)

		ctx := context.WithValue(context.TODO(), contextAPIConsumer, cons)

		m := map[string]string{}
		m["key"] = tt.args.pKey
		err = api.checkWorkflowPermissions(ctx, tt.args.wName, tt.args.permissionLevel, m)
		got := err == nil
		if !reflect.DeepEqual(got, tt.want) {
			t.Errorf("%q. checkWorkerModelPermissionsByUser() = %v, want %v", tt.name, got, tt.want)
		}
	}
}

func Test_checkJobIDPermissions(t *testing.T) {
	// TODO
}
