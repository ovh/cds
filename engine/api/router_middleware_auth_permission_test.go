package api

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ovh/cds/engine/api/action"
	"github.com/ovh/cds/engine/api/authentication"
	"github.com/ovh/cds/engine/api/authentication/local"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/engine/api/user"
	"github.com/ovh/cds/engine/api/workermodel"
	"github.com/ovh/cds/engine/api/workflowtemplate"
	"github.com/ovh/cds/sdk"
)

func Test_checkWorkflowPermissions(t *testing.T) {
	api, _, _, end := newTestAPI(t)
	defer end()

	wctx := testRunWorkflow(t, api, api.Router)

	consumer, err := local.NewConsumer(context.TODO(), api.mustDB(), wctx.user.ID)
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
	consumer.AuthentifiedUser.Groups = nil
	ctx = context.WithValue(ctx, contextAPIConsumer, consumer)
	err = api.checkWorkflowPermissions(ctx, wctx.workflow.Name, sdk.PermissionReadWriteExecute, map[string]string{
		"key":              wctx.project.Key,
		"permWorkflowName": wctx.workflow.Name,
	})
	assert.NoError(t, err, "should be granted because because is admin")

	// test case: is Maintaner
	consumer.GroupIDs = nil
	consumer.AuthentifiedUser.Groups = nil
	ctx = context.WithValue(ctx, contextAPIConsumer, consumer)
	err = api.checkWorkflowPermissions(ctx, wctx.workflow.Name, sdk.PermissionRead, map[string]string{
		"key":              wctx.project.Key,
		"permWorkflowName": wctx.workflow.Name,
	})
	assert.NoError(t, err, "should be granted because because is maintainer")

	// test case: forbidden
	consumer.GroupIDs = nil
	consumer.AuthentifiedUser.Groups = nil
	consumer.AuthentifiedUser.Ring = ""
	ctx = context.WithValue(ctx, contextAPIConsumer, consumer)
	err = api.checkWorkflowPermissions(ctx, wctx.workflow.Name, sdk.PermissionRead, map[string]string{
		"key":              wctx.project.Key,
		"permWorkflowName": wctx.workflow.Name,
	})
	assert.Error(t, err, "should not be granted")
}

func Test_checkProjectPermissions(t *testing.T) {
	api, _, _, end := newTestAPI(t)
	defer end()

	g := assets.InsertGroup(t, api.mustDB())
	authUser, _ := assets.InsertLambdaUser(t, api.mustDB(), g)

	p := assets.InsertTestProject(t, api.mustDB(), api.Cache, sdk.RandomString(10), sdk.RandomString(10))

	require.NoError(t, group.InsertLinkGroupUser(context.TODO(), api.mustDB(), &group.LinkGroupUser{
		GroupID:            p.ProjectGroups[0].Group.ID,
		AuthentifiedUserID: authUser.ID,
		Admin:              false,
	}))

	// Reload the groups for the user
	groups, err := group.LoadAllByUserID(context.TODO(), api.mustDB(), authUser.ID)
	require.NoError(t, err)
	authUser.Groups = groups

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
	consumer.AuthentifiedUser.Groups = nil
	ctx = context.WithValue(ctx, contextAPIConsumer, &consumer)
	err = api.checkProjectPermissions(ctx, p.Key, sdk.PermissionReadWriteExecute, nil)
	assert.NoError(t, err, "should be granted because because is admin")

	// test case: is Maintainer
	consumer.GroupIDs = nil
	consumer.AuthentifiedUser.Groups = nil
	ctx = context.WithValue(ctx, contextAPIConsumer, &consumer)
	err = api.checkProjectPermissions(ctx, p.Key, sdk.PermissionRead, nil)
	assert.NoError(t, err, "should be granted because because is maintainer")

	// test case: forbidden
	consumer.GroupIDs = nil
	consumer.AuthentifiedUser.Groups = nil
	consumer.AuthentifiedUser.Ring = ""
	ctx = context.WithValue(ctx, contextAPIConsumer, &consumer)
	err = api.checkProjectPermissions(ctx, p.Key, sdk.PermissionRead, nil)
	assert.Error(t, err, "should not be granted")
}

func Test_checkUserPermissions(t *testing.T) {
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
				IssuedAt:           time.Now(),
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

func Test_checkUserPublicPermissions(t *testing.T) {
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
				IssuedAt:           time.Now(),
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

func Test_checkConsumerPermissions(t *testing.T) {
	api, db, _, end := newTestAPI(t)
	defer end()

	authUser, _ := assets.InsertLambdaUser(t, db)
	authUserConsumer, err := authentication.LoadConsumerByTypeAndUserID(context.TODO(), db, sdk.ConsumerLocal, authUser.ID)
	require.NoError(t, err)
	authUserMaintainer, _ := assets.InsertMaintainerUser(t, db)
	authUserMaintainerConsumer, err := authentication.LoadConsumerByTypeAndUserID(context.TODO(), db, sdk.ConsumerLocal, authUserMaintainer.ID)
	require.NoError(t, err)
	authUserAdmin, _ := assets.InsertAdminUser(t, db)

	cases := []struct {
		Name                     string
		ConsumerAuthentifiedUser *sdk.AuthentifiedUser
		TargetConsumer           *sdk.AuthConsumer
		Permission               int
		Granted                  bool
	}{
		{
			Name:                     "RW on himself",
			ConsumerAuthentifiedUser: authUser,
			TargetConsumer:           authUserConsumer,
			Permission:               sdk.PermissionReadWriteExecute,
			Granted:                  true,
		},
		{
			Name:                     "R on other by lambda user",
			ConsumerAuthentifiedUser: authUser,
			TargetConsumer:           authUserMaintainerConsumer,
			Permission:               sdk.PermissionRead,
			Granted:                  false,
		},
		{
			Name:                     "R on other by admin user",
			ConsumerAuthentifiedUser: authUserAdmin,
			TargetConsumer:           authUserConsumer,
			Permission:               sdk.PermissionRead,
			Granted:                  true,
		},
		{
			Name:                     "R on other by maintainer user",
			ConsumerAuthentifiedUser: authUserMaintainer,
			TargetConsumer:           authUserConsumer,
			Permission:               sdk.PermissionRead,
			Granted:                  true,
		},
		{
			Name:                     "RW on other by lambda user",
			ConsumerAuthentifiedUser: authUser,
			TargetConsumer:           authUserMaintainerConsumer,
			Permission:               sdk.PermissionReadWriteExecute,
			Granted:                  false,
		},
		{
			Name:                     "RW on other by maintainer user",
			ConsumerAuthentifiedUser: authUserMaintainer,
			TargetConsumer:           authUserConsumer,
			Permission:               sdk.PermissionReadWriteExecute,
			Granted:                  false,
		},
		{
			Name:                     "RW on other by admin user",
			ConsumerAuthentifiedUser: authUserAdmin,
			TargetConsumer:           authUserConsumer,
			Permission:               sdk.PermissionReadWriteExecute,
			Granted:                  true,
		},
	}

	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			ctx := context.WithValue(context.TODO(), contextAPIConsumer, &sdk.AuthConsumer{
				AuthentifiedUserID: c.ConsumerAuthentifiedUser.ID,
				AuthentifiedUser:   c.ConsumerAuthentifiedUser,
				IssuedAt:           time.Now(),
			})
			err := api.checkConsumerPermissions(ctx, c.TargetConsumer.ID, c.Permission, nil)
			if c.Granted {
				assert.NoError(t, err, "should be granted")
			} else {
				assert.Error(t, err, "should not be granted")
			}
		})
	}
}

func Test_checkSessionPermissions(t *testing.T) {
	api, db, _, end := newTestAPI(t)
	defer end()

	authUser, _ := assets.InsertLambdaUser(t, db)
	authUserConsumer, err := authentication.LoadConsumerByTypeAndUserID(context.TODO(), db, sdk.ConsumerLocal, authUser.ID)
	require.NoError(t, err)
	authUserSession, err := authentication.NewSession(context.TODO(), db, authUserConsumer, 10*time.Second, false)
	require.NoError(t, err)
	authUserMaintainer, _ := assets.InsertMaintainerUser(t, db)
	authUserMaintainerConsumer, err := authentication.LoadConsumerByTypeAndUserID(context.TODO(), db, sdk.ConsumerLocal, authUserMaintainer.ID)
	require.NoError(t, err)
	authUserMaintainerSession, err := authentication.NewSession(context.TODO(), db, authUserMaintainerConsumer, 10*time.Second, false)
	require.NoError(t, err)
	authUserAdmin, _ := assets.InsertAdminUser(t, db)

	cases := []struct {
		Name                     string
		ConsumerAuthentifiedUser *sdk.AuthentifiedUser
		TargetSession            *sdk.AuthSession
		Permission               int
		Granted                  bool
	}{
		{
			Name:                     "RW on himself",
			ConsumerAuthentifiedUser: authUser,
			TargetSession:            authUserSession,
			Permission:               sdk.PermissionReadWriteExecute,
			Granted:                  true,
		},
		{
			Name:                     "R on other by lambda user",
			ConsumerAuthentifiedUser: authUser,
			TargetSession:            authUserMaintainerSession,
			Permission:               sdk.PermissionRead,
			Granted:                  false,
		},
		{
			Name:                     "R on other by admin user",
			ConsumerAuthentifiedUser: authUserAdmin,
			TargetSession:            authUserSession,
			Permission:               sdk.PermissionRead,
			Granted:                  true,
		},
		{
			Name:                     "R on other by maintainer user",
			ConsumerAuthentifiedUser: authUserMaintainer,
			TargetSession:            authUserSession,
			Permission:               sdk.PermissionRead,
			Granted:                  true,
		},
		{
			Name:                     "RW on other by lambda user",
			ConsumerAuthentifiedUser: authUser,
			TargetSession:            authUserMaintainerSession,
			Permission:               sdk.PermissionReadWriteExecute,
			Granted:                  false,
		},
		{
			Name:                     "RW on other by maintainer user",
			ConsumerAuthentifiedUser: authUserMaintainer,
			TargetSession:            authUserSession,
			Permission:               sdk.PermissionReadWriteExecute,
			Granted:                  false,
		},
		{
			Name:                     "RW on other by admin user",
			ConsumerAuthentifiedUser: authUserAdmin,
			TargetSession:            authUserSession,
			Permission:               sdk.PermissionReadWriteExecute,
			Granted:                  true,
		},
	}

	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			ctx := context.WithValue(context.TODO(), contextAPIConsumer, &sdk.AuthConsumer{
				AuthentifiedUserID: c.ConsumerAuthentifiedUser.ID,
				AuthentifiedUser:   c.ConsumerAuthentifiedUser,
				IssuedAt:           time.Now(),
			})
			err := api.checkSessionPermissions(ctx, c.TargetSession.ID, c.Permission, nil)
			if c.Granted {
				assert.NoError(t, err, "should be granted")
			} else {
				assert.Error(t, err, "should not be granted")
			}
		})
	}
}

func Test_checkWorkerModelPermissions(t *testing.T) {
	api, db, _, end := newTestAPI(t)
	defer end()

	g := assets.InsertTestGroup(t, db, sdk.RandomString(10))
	defer func() {
		assets.DeleteTestGroup(t, db, g)
	}()

	m := sdk.Model{
		Name: sdk.RandomString(10),
		Type: sdk.Docker,
		ModelDocker: sdk.ModelDocker{
			Image: "foo/bar:3.4",
		},
		GroupID: g.ID,
	}
	require.NoError(t, workermodel.Insert(db, &m))
	defer func() {
		require.NoError(t, workermodel.Delete(db, m.ID))
	}()

	assert.Error(t, api.checkWorkerModelPermissions(context.TODO(), m.Name, sdk.PermissionRead, map[string]string{
		"permGroupName": sdk.RandomString(10),
	}), "error should be returned for random worker model name")
	assert.Error(t, api.checkWorkerModelPermissions(context.TODO(), sdk.RandomString(10), sdk.PermissionRead, map[string]string{
		"permGroupName": g.Name,
	}), "error should be returned for random worker model name")
	assert.NoError(t, api.checkWorkerModelPermissions(context.TODO(), m.Name, sdk.PermissionRead, map[string]string{
		"permGroupName": g.Name,
	}), "no error should be returned for the right group an worker model names")
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

			require.NoError(t, group.InsertLinkGroupProject(context.TODO(), api.mustDB(), &group.LinkGroupProject{
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
			t.Errorf("%q. Test_checkWorkflowPermissionsByUser() = %v, want %v", tt.name, got, tt.want)
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

			require.NoError(t, user.Insert(context.TODO(), api.mustDB(), &currentUser))

			groupAdmin := &sdk.AuthentifiedUser{
				Username: prefix + "auto-group-admin",
			}
			require.NoError(t, user.Insert(context.TODO(), api.mustDB(), groupAdmin))

			var err error
			groupAdmin, err = user.LoadByID(context.TODO(), api.mustDB(), groupAdmin.ID)
			require.NoError(t, err)

			tt.args.groupName = prefix + tt.args.groupName

			g := sdk.Group{
				Name: tt.args.groupName,
			}

			require.NoError(t, group.Create(context.TODO(), api.mustDB(), &g, groupAdmin.ID))

			for _, adm := range tt.setup.groupAdmins {
				adm = prefix + adm
				uAdm, _ := user.LoadByUsername(context.TODO(), api.mustDB(), adm)
				if uAdm == nil {
					uAdm = &sdk.AuthentifiedUser{
						Username: adm,
						Ring:     sdk.UserRingUser,
					}
					require.NoError(t, user.Insert(context.TODO(), api.mustDB(), uAdm))
					defer assert.NoError(t, user.DeleteByID(api.mustDB(), uAdm.ID))

				}
				uAdm, _ = user.LoadByID(context.TODO(), api.mustDB(), uAdm.ID)

				require.NoError(t, group.InsertLinkGroupUser(context.TODO(), api.mustDB(), &group.LinkGroupUser{
					Admin:              true,
					GroupID:            g.ID,
					AuthentifiedUserID: uAdm.ID,
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
					require.NoError(t, user.Insert(context.TODO(), api.mustDB(), uMember))
					defer assert.NoError(t, user.DeleteByID(api.mustDB(), uMember.ID))

				}
				uMember, _ = user.LoadByID(context.TODO(), api.mustDB(), uMember.ID)

				require.NoError(t, group.InsertLinkGroupUser(context.TODO(), api.mustDB(), &group.LinkGroupUser{
					Admin:              false,
					GroupID:            g.ID,
					AuthentifiedUserID: uMember.ID,
				}))
			}

			consumer, err := local.NewConsumer(context.TODO(), api.mustDB(), currentUser.ID)
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

func Test_checkTemplateSlugPermissions(t *testing.T) {
	api, _, _, end := newTestAPI(t)
	defer end()

	type setup struct {
		groupName    string
		templateSlug string
	}
	type args struct {
		groupName    string
		templateSlug string
	}
	tests := []struct {
		name    string
		setup   setup
		args    args
		wantErr bool
	}{
		{
			name:    "invalid workflow template",
			wantErr: true,
		},
		{
			name:    "wrong group",
			wantErr: true,
			setup: setup{
				groupName:    "group",
				templateSlug: "template",
			},
			args: args{
				groupName:    "wronggroup",
				templateSlug: "template",
			},
		},
		{
			name:    "wrong template",
			wantErr: true,
			setup: setup{
				groupName:    "group",
				templateSlug: "template",
			},
			args: args{
				groupName:    "group",
				templateSlug: "wrongtemplate",
			},
		},
		{
			name:    "rignt group and template",
			wantErr: false,
			setup: setup{
				groupName:    "group",
				templateSlug: "template",
			},
			args: args{
				groupName:    "group",
				templateSlug: "template",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prefix := sdk.RandomString(10) + "."

			if tt.setup.groupName != "" {
				groupAdmin := &sdk.AuthentifiedUser{
					Username: prefix + "auto-group-admin",
				}
				require.NoError(t, user.Insert(context.TODO(), api.mustDB(), groupAdmin))

				var err error
				groupAdmin, err = user.LoadByID(context.TODO(), api.mustDB(), groupAdmin.ID)
				require.NoError(t, err)
				tt.setup.groupName = prefix + tt.setup.groupName
				g := sdk.Group{
					Name: tt.setup.groupName,
				}
				require.NoError(t, group.Create(context.TODO(), api.mustDB(), &g, groupAdmin.ID))
				t.Logf("group %s created", g.Name)

				if tt.setup.templateSlug != "" {
					tt.setup.templateSlug = prefix + tt.setup.templateSlug

					template := sdk.WorkflowTemplate{
						GroupID: g.ID,
						Name:    tt.setup.templateSlug,
						Slug:    tt.setup.templateSlug,
					}
					require.NoError(t, workflowtemplate.Insert(api.mustDB(), &template))
					t.Logf("template %s created", template.Name)
				}
			}

			ctx := context.TODO()
			err := api.checkTemplateSlugPermissions(ctx, prefix+tt.args.templateSlug, sdk.PermissionRead, map[string]string{"permGroupName": prefix + tt.args.groupName})
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func Test_checkActionPermissions(t *testing.T) {
	api, db, _, end := newTestAPI(t)
	defer end()

	g := assets.InsertTestGroup(t, db, sdk.RandomString(10))
	defer func() {
		assets.DeleteTestGroup(t, db, g)
	}()

	a := sdk.Action{
		GroupID: &g.ID,
		Type:    sdk.DefaultAction,
		Name:    sdk.RandomString(10),
	}
	require.NoError(t, action.Insert(db, &a))
	defer func() {
		require.NoError(t, action.Delete(db, &a))
	}()

	assert.Error(t, api.checkActionPermissions(context.TODO(), a.Name, sdk.PermissionRead, map[string]string{
		"permGroupName": sdk.RandomString(10),
	}), "error should be returned for random group name")
	assert.Error(t, api.checkActionPermissions(context.TODO(), sdk.RandomString(10), sdk.PermissionRead, map[string]string{
		"permGroupName": g.Name,
	}), "error should be returned for random action name")
	assert.NoError(t, api.checkActionPermissions(context.TODO(), a.Name, sdk.PermissionRead, map[string]string{
		"permGroupName": g.Name,
	}), "no error should be returned for the right group an action names")
}

func Test_checkActionBuiltinPermissions(t *testing.T) {
	api, db, _, end := newTestAPI(t)
	defer end()

	scriptAction := assets.GetBuiltinOrPluginActionByName(t, db, "Script")

	assert.Error(t, api.checkActionBuiltinPermissions(context.TODO(), sdk.RandomString(10), sdk.PermissionRead, nil), "error should be returned for random action name")
	assert.NoError(t, api.checkActionBuiltinPermissions(context.TODO(), scriptAction.Name, sdk.PermissionRead, nil), "no error should be returned for valid action name")
}
