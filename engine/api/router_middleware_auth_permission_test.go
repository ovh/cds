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
	"github.com/ovh/cds/engine/api/user"
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
	authUser, _ := assets.InsertLambdaUser(t, api.mustDB(), g)

	p := assets.InsertTestProject(t, api.mustDB(), api.Cache, sdk.RandomString(10), sdk.RandomString(10))

	require.NoError(t, group.InsertLinkGroupUser(api.mustDB(), &group.LinkGroupUser{
		GroupID: p.ProjectGroups[0].Group.ID,
		UserID:  authUser.OldUserStruct.ID,
		Admin:   false,
	}))

	// Reload the groups for the user
	groups, err := group.LoadAllByDeprecatedUserID(context.TODO(), api.mustDB(), authUser.OldUserStruct.ID)
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

	authUser, _ := assets.InsertLambdaUser(t, db)
	authUserMaintainer, _ := assets.InsertMaintainerUser(t, db)
	authUserAdmin, _ := assets.InsertAdminUser(t, db)

	cases := []struct {
		Name                     string
		ConsumerAuthentifiedUser *sdk.AuthentifiedUser
		TargetAuthentifiedUser   *sdk.AuthentifiedUser
		Permission               int
		Granted                  bool
	}{
		{
			Name:                     "RW on himself",
			ConsumerAuthentifiedUser: authUser,
			TargetAuthentifiedUser:   authUser,
			Permission:               sdk.PermissionReadWriteExecute,
			Granted:                  true,
		},
		{
			Name:                     "R on other by lambda user",
			ConsumerAuthentifiedUser: authUser,
			TargetAuthentifiedUser:   authUserAdmin,
			Permission:               sdk.PermissionRead,
			Granted:                  false,
		},
		{
			Name:                     "R on other by admin user",
			ConsumerAuthentifiedUser: authUserAdmin,
			TargetAuthentifiedUser:   authUser,
			Permission:               sdk.PermissionRead,
			Granted:                  true,
		},
		{
			Name:                     "R on other by maintainer user",
			ConsumerAuthentifiedUser: authUserMaintainer,
			TargetAuthentifiedUser:   authUser,
			Permission:               sdk.PermissionRead,
			Granted:                  true,
		},
		{
			Name:                     "RW on other by lambda user",
			ConsumerAuthentifiedUser: authUser,
			TargetAuthentifiedUser:   authUserAdmin,
			Permission:               sdk.PermissionReadWriteExecute,
			Granted:                  false,
		},
		{
			Name:                     "RW on other by maintainer user",
			ConsumerAuthentifiedUser: authUserMaintainer,
			TargetAuthentifiedUser:   authUser,
			Permission:               sdk.PermissionReadWriteExecute,
			Granted:                  false,
		},
		{
			Name:                     "RW on other by admin user",
			ConsumerAuthentifiedUser: authUserAdmin,
			TargetAuthentifiedUser:   authUser,
			Permission:               sdk.PermissionReadWriteExecute,
			Granted:                  true,
		},
	}

	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			ctx := context.WithValue(context.TODO(), contextAPIConsumer, &sdk.AuthConsumer{
				AuthentifiedUserID: c.ConsumerAuthentifiedUser.ID,
				AuthentifiedUser:   c.ConsumerAuthentifiedUser,
			})
			err := api.checkUserPermissions(ctx, c.TargetAuthentifiedUser.Username, c.Permission, nil)
			if c.Granted {
				assert.NoError(t, err, "should be granted")
			} else {
				assert.Error(t, err, "should not be granted")
			}
		})
	}
}

func TestAPI_checkUserPublicPermissions(t *testing.T) {
	api, db, _, end := newTestAPI(t)
	defer end()

	authUser, _ := assets.InsertLambdaUser(t, db)
	authUserMaintainer, _ := assets.InsertMaintainerUser(t, db)
	authUserAdmin, _ := assets.InsertAdminUser(t, db)

	cases := []struct {
		Name                     string
		ConsumerAuthentifiedUser *sdk.AuthentifiedUser
		TargetAuthentifiedUser   *sdk.AuthentifiedUser
		Permission               int
		Granted                  bool
	}{
		{
			Name:                     "RW on himself",
			ConsumerAuthentifiedUser: authUser,
			TargetAuthentifiedUser:   authUser,
			Permission:               sdk.PermissionReadWriteExecute,
			Granted:                  true,
		},
		{
			Name:                     "R on other by lambda user",
			ConsumerAuthentifiedUser: authUser,
			TargetAuthentifiedUser:   authUserAdmin,
			Permission:               sdk.PermissionRead,
			Granted:                  true,
		},
		{
			Name:                     "R on other by admin user",
			ConsumerAuthentifiedUser: authUserAdmin,
			TargetAuthentifiedUser:   authUser,
			Permission:               sdk.PermissionRead,
			Granted:                  true,
		},
		{
			Name:                     "R on other by maintainer user",
			ConsumerAuthentifiedUser: authUserMaintainer,
			TargetAuthentifiedUser:   authUser,
			Permission:               sdk.PermissionRead,
			Granted:                  true,
		},
		{
			Name:                     "RW on other by lambda user",
			ConsumerAuthentifiedUser: authUser,
			TargetAuthentifiedUser:   authUserAdmin,
			Permission:               sdk.PermissionReadWriteExecute,
			Granted:                  false,
		},
		{
			Name:                     "RW on other by maintainer user",
			ConsumerAuthentifiedUser: authUserMaintainer,
			TargetAuthentifiedUser:   authUser,
			Permission:               sdk.PermissionReadWriteExecute,
			Granted:                  false,
		},
		{
			Name:                     "RW on other by admin user",
			ConsumerAuthentifiedUser: authUserAdmin,
			TargetAuthentifiedUser:   authUser,
			Permission:               sdk.PermissionReadWriteExecute,
			Granted:                  true,
		},
	}

	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			ctx := context.WithValue(context.TODO(), contextAPIConsumer, &sdk.AuthConsumer{
				AuthentifiedUserID: c.ConsumerAuthentifiedUser.ID,
				AuthentifiedUser:   c.ConsumerAuthentifiedUser,
			})
			err := api.checkUserPublicPermissions(ctx, c.TargetAuthentifiedUser.Username, c.Permission, nil)
			if c.Granted {
				assert.NoError(t, err, "should be granted")
			} else {
				assert.Error(t, err, "should not be granted")
			}
		})
	}
}

func TestAPI_checkConsumerPermissions(t *testing.T) {
	api, db, _, end := newTestAPI(t)
	defer end()

	authUser, _ := assets.InsertLambdaUser(t, db)
	localConsumer, err := authentication.LoadConsumerByTypeAndUserID(context.TODO(), db, sdk.ConsumerLocal, authUser.ID)
	require.NoError(t, err)

	authUserAdmin, _ := assets.InsertAdminUser(t, db)
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

	authUser, _ := assets.InsertLambdaUser(t, db)
	localConsumer, err := authentication.LoadConsumerByTypeAndUserID(context.TODO(), db, sdk.ConsumerLocal, authUser.ID)
	require.NoError(t, err)
	localSession, err := authentication.NewSession(db, localConsumer, time.Second, false)
	require.NoError(t, err)

	authUserAdmin, _ := assets.InsertAdminUser(t, db)
	localConsumerAdmin, err := authentication.LoadConsumerByTypeAndUserID(context.TODO(), db, sdk.ConsumerLocal, authUserAdmin.ID)
	require.NoError(t, err)
	localSessionAdmin, err := authentication.NewSession(db, localConsumerAdmin, time.Second, false)
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
							Members: []sdk.GroupMember{
								{
									ID:    sdk.RandomString(10),
									Admin: true,
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
			usr, _ = assets.InsertAdminUser(t, api.mustDB())
		} else {
			usr, _ = assets.InsertLambdaUser(t, api.mustDB(), groups...)
		}

		proj := assets.InsertTestProject(t, api.mustDB(), api.Cache, tt.args.pKey, tt.args.pKey)
		wrkflw := assets.InsertTestWorkflow(t, api.mustDB(), api.Cache, proj, tt.args.wName)

		for groupName, permLevel := range tt.setup.ProjGroupPermissions {
			g, err := group.LoadByName(context.TODO(), api.mustDB(), groupName+suffix, group.LoadOptions.WithMembers)
			require.NoError(t, err)

			require.NoError(t, group.InsertLinkGroupProject(api.mustDB(), &group.LinkGroupProject{
				GroupID:   g.ID,
				ProjectID: proj.ID,
				Role:      permLevel,
			}))
		}

		for groupName, permLevel := range tt.setup.WorkflowGroupPermissions {
			g, err := group.LoadByName(context.TODO(), api.mustDB(), groupName+suffix, group.LoadOptions.WithMembers)
			require.NoError(t, err)

			require.NoError(t, group.AddWorkflowGroup(context.TODO(), api.mustDB(), wrkflw, sdk.GroupPermission{
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

func Test_checkGroupPermissions(t *testing.T) {
	api, _, _, end := newTestAPI(t)
	defer end()

	type setup struct {
		currentUser           string
		currenUserIsAdmin     bool
		currenUserIsMaitainer bool
		groupAdmins           []string
		groupMembers          []string
	}
	type args struct {
		groupName       string
		permissionLevel int
	}
	tests := []struct {
		name    string
		setup   setup
		args    args
		wantErr bool
	}{
		{
			name:    "invalid group name",
			wantErr: true,
		},
		{
			name:    "group does not exist",
			wantErr: true,
		},
		{
			name:    "admin can get group",
			wantErr: false,
			setup: setup{
				currentUser:       "admin",
				currenUserIsAdmin: true,
			},
			args: args{
				groupName:       "my_group",
				permissionLevel: sdk.PermissionRead,
			},
		},
		{
			name:    "maintainer can get group",
			wantErr: false,
			setup: setup{
				currentUser:           "maintainer",
				currenUserIsMaitainer: true,
			},
			args: args{
				groupName:       "my_group",
				permissionLevel: sdk.PermissionRead,
			},
		},
		{
			name:    "admin can update group",
			wantErr: false,
			setup: setup{
				currentUser:       "admin",
				currenUserIsAdmin: true,
			},
			args: args{
				groupName:       "my_group",
				permissionLevel: sdk.PermissionReadWriteExecute,
			},
		},
		{
			name:    "maintainer can't update group",
			wantErr: true,
			setup: setup{
				currentUser:           "maintainer",
				currenUserIsMaitainer: true,
			},
			args: args{
				groupName:       "my_group",
				permissionLevel: sdk.PermissionReadWriteExecute,
			},
		},
		{
			name:    "group admin can read group",
			wantErr: false,
			setup: setup{
				currentUser: "group_admin",
				groupAdmins: []string{"group_admin"},
			},
			args: args{
				groupName:       "my_group",
				permissionLevel: sdk.PermissionRead,
			},
		},
		{
			name:    "group member can read group",
			wantErr: false,
			setup: setup{
				currentUser:  "group_member",
				groupMembers: []string{"group_member"},
			},
			args: args{
				groupName:       "my_group",
				permissionLevel: sdk.PermissionRead,
			},
		},
		{
			name:    "group admin can update group",
			wantErr: false,
			setup: setup{
				currentUser: "group_admin",
				groupAdmins: []string{"group_admin"},
			},
			args: args{
				groupName:       "my_group",
				permissionLevel: sdk.PermissionReadWriteExecute,
			},
		},
		{
			name:    "group member can't update group",
			wantErr: true,
			setup: setup{
				currentUser:  "group_member",
				groupMembers: []string{"group_member"},
			},
			args: args{
				groupName:       "my_group",
				permissionLevel: sdk.PermissionReadWriteExecute,
			},
		},
		{
			name:    "lambda user can't get group",
			wantErr: true,
			setup: setup{
				currentUser: "user",
			},
			args: args{
				groupName:       "my_group",
				permissionLevel: sdk.PermissionRead,
			},
		},
		{
			name:    "lambda user can't update group",
			wantErr: true,
			setup: setup{
				currentUser: "user",
			},
			args: args{
				groupName:       "my_group",
				permissionLevel: sdk.PermissionReadWriteExecute,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prefix := sdk.RandomString(10) + "."
			currentUser := sdk.AuthentifiedUser{
				Username: prefix + tt.setup.currentUser,
			}
			if tt.setup.currenUserIsAdmin {
				currentUser.Ring = sdk.UserRingAdmin
			} else if tt.setup.currenUserIsMaitainer {
				currentUser.Ring = sdk.UserRingMaintainer
			} else {
				currentUser.Ring = sdk.UserRingUser
			}

			require.NoError(t, user.Insert(api.mustDB(), &currentUser))

			groupAdmin := &sdk.AuthentifiedUser{
				Username: prefix + "auto-group-admin",
			}
			require.NoError(t, user.Insert(api.mustDB(), groupAdmin))

			var err error
			groupAdmin, err = user.LoadByID(context.TODO(), api.mustDB(), groupAdmin.ID, user.LoadOptions.WithDeprecatedUser)
			require.NoError(t, err)

			tt.args.groupName = prefix + tt.args.groupName

			g := sdk.Group{
				Name: tt.args.groupName,
			}

			require.NoError(t, group.Create(api.mustDB(), &g, groupAdmin.OldUserStruct.ID))

			for _, adm := range tt.setup.groupAdmins {
				adm = prefix + adm
				uAdm, _ := user.LoadByUsername(context.TODO(), api.mustDB(), adm)
				if uAdm == nil {
					uAdm = &sdk.AuthentifiedUser{
						Username: adm,
						Ring:     sdk.UserRingUser,
					}
					require.NoError(t, user.Insert(api.mustDB(), uAdm))
					defer assert.NoError(t, user.DeleteByID(api.mustDB(), uAdm.ID))

				}
				uAdm, _ = user.LoadByID(context.TODO(), api.mustDB(), uAdm.ID, user.LoadOptions.WithDeprecatedUser)

				require.NoError(t, group.InsertLinkGroupUser(api.mustDB(), &group.LinkGroupUser{
					Admin:   true,
					GroupID: g.ID,
					UserID:  uAdm.OldUserStruct.ID,
				}))
			}

			for _, member := range tt.setup.groupMembers {
				member = prefix + member
				uMember, _ := user.LoadByUsername(context.TODO(), api.mustDB(), member)
				if uMember == nil {
					uMember = &sdk.AuthentifiedUser{
						Username: member,
						Ring:     sdk.UserRingUser,
					}
					require.NoError(t, user.Insert(api.mustDB(), uMember))
					defer assert.NoError(t, user.DeleteByID(api.mustDB(), uMember.ID))

				}
				uMember, _ = user.LoadByID(context.TODO(), api.mustDB(), uMember.ID, user.LoadOptions.WithDeprecatedUser)

				require.NoError(t, group.InsertLinkGroupUser(api.mustDB(), &group.LinkGroupUser{
					Admin:   false,
					GroupID: g.ID,
					UserID:  uMember.OldUserStruct.ID,
				}))
			}

			consumer, err := local.NewConsumer(api.mustDB(), currentUser.ID)
			require.NoError(t, err)
			consumer, err = authentication.LoadConsumerByID(context.TODO(), api.mustDB(), consumer.ID, authentication.LoadConsumerOptions.WithAuthentifiedUser)
			require.NoError(t, err)

			ctx := context.TODO()
			ctx = context.WithValue(ctx, contextAPIConsumer, consumer)

			err = api.checkGroupPermissions(ctx, tt.args.groupName, tt.args.permissionLevel, nil)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
