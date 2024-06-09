package api

import (
	"context"
	"github.com/ovh/cds/engine/api/organization"
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
	api, db, _ := newTestAPI(t)

	wctx := testRunWorkflow(t, api, api.Router)
	user := wctx.user
	admin, _ := assets.InsertAdminUser(t, db)
	maintainer, _ := assets.InsertAdminUser(t, db)

	ctx := context.Background()

	ctx = context.WithValue(ctx, contextDriverManifest, &sdk.AuthDriverManifest{})

	consumer := &sdk.AuthUserConsumer{AuthConsumerUser: sdk.AuthUserConsumerData{}}

	// test case: has enough permission
	consumer.AuthConsumerUser.AuthentifiedUser = user
	ctx = context.WithValue(ctx, contextUserConsumer, consumer)
	err := api.checkWorkflowPermissions(ctx, &responseTracker{}, wctx.workflow.Name, sdk.PermissionReadWriteExecute, map[string]string{
		"key":              wctx.project.Key,
		"permWorkflowName": wctx.workflow.Name,
	})
	assert.NoError(t, err, "should be granted because has permission (max permission = 7)")

	// test case: is Admin
	consumer.AuthConsumerUser.GroupIDs = nil
	consumer.AuthConsumerUser.AuthentifiedUser.Groups = nil
	consumer.AuthConsumerUser.AuthentifiedUser = admin
	ctx = context.WithValue(ctx, contextUserConsumer, consumer)
	err = api.checkWorkflowPermissions(ctx, &responseTracker{}, wctx.workflow.Name, sdk.PermissionReadWriteExecute, map[string]string{
		"key":              wctx.project.Key,
		"permWorkflowName": wctx.workflow.Name,
	})
	assert.NoError(t, err, "should be granted because because is admin")

	// test case: is Maintainer
	consumer.AuthConsumerUser.GroupIDs = nil
	consumer.AuthConsumerUser.AuthentifiedUser.Groups = nil
	consumer.AuthConsumerUser.AuthentifiedUser = maintainer
	ctx = context.WithValue(ctx, contextUserConsumer, consumer)
	err = api.checkWorkflowPermissions(ctx, &responseTracker{}, wctx.workflow.Name, sdk.PermissionRead, map[string]string{
		"key":              wctx.project.Key,
		"permWorkflowName": wctx.workflow.Name,
	})
	assert.NoError(t, err, "should be granted because because is maintainer")

	// test case: forbidden
	consumer.AuthConsumerUser.GroupIDs = nil
	consumer.AuthConsumerUser.AuthentifiedUser.Groups = nil
	consumer.AuthConsumerUser.AuthentifiedUser.Ring = ""
	consumer.AuthConsumerUser.AuthentifiedUser = user
	ctx = context.WithValue(ctx, contextUserConsumer, consumer)
	err = api.checkWorkflowPermissions(ctx, &responseTracker{}, wctx.workflow.Name, sdk.PermissionRead, map[string]string{
		"key":              wctx.project.Key,
		"permWorkflowName": wctx.workflow.Name,
	})
	assert.Error(t, err, "should not be granted")

	// test case: worker for same project
	w2ctx := testRunWorkflowForProject(t, api, api.Router, wctx.project, wctx.userToken)
	consumer.AuthConsumerUser.GroupIDs = nil
	consumer.AuthConsumerUser.AuthentifiedUser = admin
	consumer.AuthConsumerUser.Worker = &sdk.Worker{
		JobRunID: &wctx.job.ID,
	}
	ctx = context.WithValue(ctx, contextUserConsumer, consumer)
	err = api.checkWorkflowPermissions(ctx, &responseTracker{}, w2ctx.workflow.Name, sdk.PermissionRead, map[string]string{
		"key":              w2ctx.project.Key,
		"permWorkflowName": w2ctx.workflow.Name,
	})
	require.Error(t, err, "should not be granted")
	err = api.checkWorkflowAdvancedPermissions(ctx, &responseTracker{}, w2ctx.workflow.Name, sdk.PermissionRead, map[string]string{
		"key":              w2ctx.project.Key,
		"permWorkflowName": w2ctx.workflow.Name,
	})
	require.NoError(t, err, "should be granted")

	// test case: worker for different project
	w3ctx := testRunWorkflow(t, api, api.Router)
	consumer.AuthConsumerUser.GroupIDs = nil
	consumer.AuthConsumerUser.AuthentifiedUser = admin
	consumer.AuthConsumerUser.Worker = &sdk.Worker{
		JobRunID: &wctx.job.ID,
	}
	ctx = context.WithValue(ctx, contextUserConsumer, consumer)
	err = api.checkWorkflowAdvancedPermissions(ctx, &responseTracker{}, w3ctx.workflow.Name, sdk.PermissionRead, map[string]string{
		"key":              w3ctx.project.Key,
		"permWorkflowName": w3ctx.workflow.Name,
	})
	require.Error(t, err, "should not be granted")
}

func Test_checkProjectPermissions(t *testing.T) {
	api, db, _ := newTestAPI(t)

	g := assets.InsertGroup(t, db)
	authUser, _ := assets.InsertLambdaUser(t, db, g)

	p := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))

	require.NoError(t, group.InsertLinkGroupUser(context.TODO(), db, &group.LinkGroupUser{
		GroupID:            p.ProjectGroups[0].Group.ID,
		AuthentifiedUserID: authUser.ID,
		Admin:              false,
	}))

	// Reload the groups for the user
	groups, err := group.LoadAllByUserID(context.TODO(), api.mustDB(), authUser.ID)
	require.NoError(t, err)
	authUser.Groups = groups

	consumer := sdk.AuthUserConsumer{AuthConsumerUser: sdk.AuthUserConsumerData{}}
	consumer.AuthConsumerUser.AuthentifiedUser = authUser
	ctx := context.Background()
	ctx = context.WithValue(ctx, contextDriverManifest, &sdk.AuthDriverManifest{})

	// test case: has enough permission
	ctx = context.WithValue(ctx, contextUserConsumer, &consumer)
	err = api.checkProjectPermissions(ctx, &responseTracker{}, p.Key, sdk.PermissionReadWriteExecute, nil)
	assert.NoError(t, err, "should be granted because has permission (max permission = 7)")

	// test case: is Admin
	consumer.AuthConsumerUser.AuthentifiedUser.Ring = sdk.UserRingAdmin
	consumer.AuthConsumerUser.GroupIDs = nil
	consumer.AuthConsumerUser.AuthentifiedUser.Groups = nil
	ctx = context.WithValue(ctx, contextUserConsumer, &consumer)
	err = api.checkProjectPermissions(ctx, &responseTracker{}, p.Key, sdk.PermissionReadWriteExecute, nil)
	assert.NoError(t, err, "should be granted because because is admin")

	// test case: is Maintainer
	consumer.AuthConsumerUser.GroupIDs = nil
	consumer.AuthConsumerUser.AuthentifiedUser.Groups = nil
	ctx = context.WithValue(ctx, contextUserConsumer, &consumer)
	err = api.checkProjectPermissions(ctx, &responseTracker{}, p.Key, sdk.PermissionRead, nil)
	assert.NoError(t, err, "should be granted because because is maintainer")

	// test case: forbidden
	consumer.AuthConsumerUser.GroupIDs = nil
	consumer.AuthConsumerUser.AuthentifiedUser.Groups = nil
	consumer.AuthConsumerUser.AuthentifiedUser.Ring = ""
	ctx = context.WithValue(ctx, contextUserConsumer, &consumer)
	err = api.checkProjectPermissions(ctx, &responseTracker{}, p.Key, sdk.PermissionRead, nil)
	assert.Error(t, err, "should not be granted")
}

func Test_checkUserPermissions(t *testing.T) {
	api, db, _ := newTestAPI(t)

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
			ctx := context.WithValue(context.TODO(), contextUserConsumer, &sdk.AuthUserConsumer{
				AuthConsumerUser: sdk.AuthUserConsumerData{
					AuthentifiedUserID: c.ConsumerAuthentifiedUser.ID,
					AuthentifiedUser:   c.ConsumerAuthentifiedUser,
				},
				AuthConsumer: sdk.AuthConsumer{
					ValidityPeriods: sdk.NewAuthConsumerValidityPeriod(time.Now(), 0),
				},
			})
			ctx = context.WithValue(ctx, contextDriverManifest, &sdk.AuthDriverManifest{})
			err := api.checkUserPermissions(ctx, &responseTracker{}, c.TargetAuthentifiedUser.Username, c.Permission, nil)
			if c.Granted {
				assert.NoError(t, err, "should be granted")
			} else {
				assert.Error(t, err, "should not be granted")
			}
		})
	}
}

func Test_checkUserPublicPermissions(t *testing.T) {
	api, db, _ := newTestAPI(t)

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
			ctx := context.WithValue(context.TODO(), contextUserConsumer, &sdk.AuthUserConsumer{
				AuthConsumer: sdk.AuthConsumer{
					ValidityPeriods: sdk.NewAuthConsumerValidityPeriod(time.Now(), 0),
				},
				AuthConsumerUser: sdk.AuthUserConsumerData{
					AuthentifiedUserID: c.ConsumerAuthentifiedUser.ID,
					AuthentifiedUser:   c.ConsumerAuthentifiedUser,
				},
			})
			ctx = context.WithValue(ctx, contextDriverManifest, &sdk.AuthDriverManifest{})
			err := api.checkUserPublicPermissions(ctx, &responseTracker{}, c.TargetAuthentifiedUser.Username, c.Permission, nil)
			if c.Granted {
				assert.NoError(t, err, "should be granted")
			} else {
				assert.Error(t, err, "should not be granted")
			}
		})
	}
}

func Test_checkConsumerPermissions(t *testing.T) {
	api, db, _ := newTestAPI(t)

	authUser, _ := assets.InsertLambdaUser(t, db)
	authUserConsumer, err := authentication.LoadUserConsumerByTypeAndUserID(context.TODO(), db, sdk.ConsumerLocal, authUser.ID)
	require.NoError(t, err)
	authUserMaintainer, _ := assets.InsertMaintainerUser(t, db)
	authUserMaintainerConsumer, err := authentication.LoadUserConsumerByTypeAndUserID(context.TODO(), db, sdk.ConsumerLocal, authUserMaintainer.ID)
	require.NoError(t, err)
	authUserAdmin, _ := assets.InsertAdminUser(t, db)

	cases := []struct {
		Name                     string
		ConsumerAuthentifiedUser *sdk.AuthentifiedUser
		TargetConsumer           *sdk.AuthUserConsumer
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
			ctx := context.WithValue(context.TODO(), contextUserConsumer, &sdk.AuthUserConsumer{
				AuthConsumerUser: sdk.AuthUserConsumerData{
					AuthentifiedUserID: c.ConsumerAuthentifiedUser.ID,
					AuthentifiedUser:   c.ConsumerAuthentifiedUser,
				},
				AuthConsumer: sdk.AuthConsumer{
					ValidityPeriods: sdk.NewAuthConsumerValidityPeriod(time.Now(), 0),
				},
			})
			ctx = context.WithValue(ctx, contextDriverManifest, &sdk.AuthDriverManifest{})
			err := api.checkConsumerPermissions(ctx, &responseTracker{}, c.TargetConsumer.ID, c.Permission, nil)
			if c.Granted {
				assert.NoError(t, err, "should be granted")
			} else {
				assert.Error(t, err, "should not be granted")
			}
		})
	}
}

func Test_checkSessionPermissions(t *testing.T) {
	api, db, _ := newTestAPI(t)

	authUser, _ := assets.InsertLambdaUser(t, db)
	authUserConsumer, err := authentication.LoadUserConsumerByTypeAndUserID(context.TODO(), db, sdk.ConsumerLocal, authUser.ID)
	require.NoError(t, err)
	authUserSession, err := authentication.NewSession(context.TODO(), db, &authUserConsumer.AuthConsumer, 10*time.Second)
	require.NoError(t, err)
	authUserMaintainer, _ := assets.InsertMaintainerUser(t, db)
	authUserMaintainerConsumer, err := authentication.LoadUserConsumerByTypeAndUserID(context.TODO(), db, sdk.ConsumerLocal, authUserMaintainer.ID)
	require.NoError(t, err)
	authUserMaintainerSession, err := authentication.NewSession(context.TODO(), db, &authUserMaintainerConsumer.AuthConsumer, 10*time.Second)
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
			ctx := context.WithValue(context.TODO(), contextUserConsumer, &sdk.AuthUserConsumer{
				AuthConsumerUser: sdk.AuthUserConsumerData{
					AuthentifiedUserID: c.ConsumerAuthentifiedUser.ID,
					AuthentifiedUser:   c.ConsumerAuthentifiedUser,
				},
				AuthConsumer: sdk.AuthConsumer{
					ValidityPeriods: sdk.NewAuthConsumerValidityPeriod(time.Now(), 0),
				},
			})
			ctx = context.WithValue(ctx, contextDriverManifest, &sdk.AuthDriverManifest{})
			err := api.checkSessionPermissions(ctx, &responseTracker{}, c.TargetSession.ID, c.Permission, nil)
			if c.Granted {
				assert.NoError(t, err, "should be granted")
			} else {
				assert.Error(t, err, "should not be granted")
			}
		})
	}
}

func Test_checkWorkerModelPermissions(t *testing.T) {
	api, db, _ := newTestAPI(t)

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
	require.NoError(t, workermodel.Insert(context.TODO(), db, &m))
	defer func() {
		require.NoError(t, workermodel.DeleteByID(db, m.ID))
	}()

	assert.Error(t, api.checkWorkerModelPermissions(context.TODO(), &responseTracker{}, m.Name, sdk.PermissionRead, map[string]string{
		"permGroupName": sdk.RandomString(10),
	}), "error should be returned for random worker model name")
	assert.Error(t, api.checkWorkerModelPermissions(context.TODO(), &responseTracker{}, sdk.RandomString(10), sdk.PermissionRead, map[string]string{
		"permGroupName": g.Name,
	}), "error should be returned for random worker model name")
	assert.NoError(t, api.checkWorkerModelPermissions(context.TODO(), &responseTracker{}, m.Name, sdk.PermissionRead, map[string]string{
		"permGroupName": g.Name,
	}), "no error should be returned for the right group an worker model names")
}

func Test_checkWorkflowPermissionsByUser(t *testing.T) {
	api, db, _ := newTestAPI(t)

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
			usr, _ = assets.InsertAdminUser(t, db)
		} else {
			usr, _ = assets.InsertLambdaUser(t, db, groups...)
		}

		proj := assets.InsertTestProject(t, db, api.Cache, tt.args.pKey, tt.args.pKey)
		wrkflw := assets.InsertTestWorkflow(t, db, api.Cache, proj, tt.args.wName)

		for groupName, permLevel := range tt.setup.ProjGroupPermissions {
			g, err := group.LoadByName(context.TODO(), api.mustDB(), groupName+suffix, group.LoadOptions.WithMembers)
			require.NoError(t, err)

			require.NoError(t, group.InsertLinkGroupProject(context.TODO(), db, &group.LinkGroupProject{
				GroupID:   g.ID,
				ProjectID: proj.ID,
				Role:      permLevel,
			}))
		}

		for groupName, permLevel := range tt.setup.WorkflowGroupPermissions {
			g, err := group.LoadByName(context.TODO(), db, groupName+suffix, group.LoadOptions.WithMembers)
			require.NoError(t, err)

			require.NoError(t, group.AddWorkflowGroup(context.TODO(), db, wrkflw, sdk.GroupPermission{
				Group:      *g,
				Permission: permLevel,
			}))
		}

		cons, err := authentication.LoadUserConsumerByTypeAndUserID(context.TODO(), api.mustDB(), sdk.ConsumerLocal, usr.ID, authentication.LoadUserConsumerOptions.WithAuthentifiedUser)
		require.NoError(t, err)
		ctx := context.WithValue(context.TODO(), contextUserConsumer, cons)
		ctx = context.WithValue(ctx, contextDriverManifest, &sdk.AuthDriverManifest{})

		m := map[string]string{}
		m["key"] = tt.args.pKey
		err = api.checkWorkflowPermissions(ctx, &responseTracker{}, tt.args.wName, tt.args.permissionLevel, m)
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
	api, db, _ := newTestAPI(t)

	type setup struct {
		currentUser           string
		currentUserIsAdmin     bool
		currentUserIsMaitainer bool
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
				currentUserIsAdmin: true,
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
				currentUserIsMaitainer: true,
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
				currentUserIsAdmin: true,
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
				currentUserIsMaitainer: true,
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
	organizations, err := organization.LoadOrganizations(context.TODO(), db)
	require.NoError(t, err)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prefix := sdk.RandomString(10) + "."
			currentUser := sdk.AuthentifiedUser{
				Username: prefix + tt.setup.currentUser,
			}
			if tt.setup.currentUserIsAdmin {
				currentUser.Ring = sdk.UserRingAdmin
			} else if tt.setup.currentUserIsMaitainer {
				currentUser.Ring = sdk.UserRingMaintainer
			} else {
				currentUser.Ring = sdk.UserRingUser
			}

			require.NoError(t, user.Insert(context.TODO(), db, &currentUser))
			uo := user.UserOrganization{
				AuthentifiedUserID: currentUser.ID,
				OrganizationID:     organizations[0].ID,
			}
			require.NoError(t, user.InsertUserOrganization(context.TODO(), db, &uo))

			userGrpAdmin := &sdk.AuthentifiedUser{
				Username: prefix + "auto-group-admin",
			}
			require.NoError(t, user.Insert(context.TODO(), db, userGrpAdmin))
			uoAdmin := user.UserOrganization{
				AuthentifiedUserID: userGrpAdmin.ID,
				OrganizationID:     organizations[0].ID,
			}
			require.NoError(t, user.InsertUserOrganization(context.TODO(), db, &uoAdmin))

			var err error
			userGrpAdmin, err = user.LoadByID(context.TODO(), api.mustDB(), userGrpAdmin.ID, user.LoadOptions.WithOrganization)
			require.NoError(t, err)

			tt.args.groupName = prefix + tt.args.groupName

			g := sdk.Group{
				Name: tt.args.groupName,
			}

			require.NoError(t, group.Create(context.TODO(), db, &g, userGrpAdmin))

			for _, adm := range tt.setup.groupAdmins {
				adm = prefix + adm
				uAdm, _ := user.LoadByUsername(context.TODO(), api.mustDB(), adm)
				if uAdm == nil {
					uAdm = &sdk.AuthentifiedUser{
						Username: adm,
						Ring:     sdk.UserRingUser,
					}
					require.NoError(t, user.Insert(context.TODO(), db, uAdm))
					defer assert.NoError(t, user.DeleteByID(api.mustDB(), uAdm.ID))
				}
				uAdm, _ = user.LoadByID(context.TODO(), api.mustDB(), uAdm.ID)

				require.NoError(t, group.InsertLinkGroupUser(context.TODO(), db, &group.LinkGroupUser{
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
					require.NoError(t, user.Insert(context.TODO(), db, uMember))
					defer assert.NoError(t, user.DeleteByID(api.mustDB(), uMember.ID))

				}
				uMember, _ = user.LoadByID(context.TODO(), api.mustDB(), uMember.ID)

				require.NoError(t, group.InsertLinkGroupUser(context.TODO(), db, &group.LinkGroupUser{
					Admin:              false,
					GroupID:            g.ID,
					AuthentifiedUserID: uMember.ID,
				}))
			}

			consumer, err := local.NewConsumer(context.TODO(), db, currentUser.ID)
			require.NoError(t, err)
			consumer, err = authentication.LoadUserConsumerByID(context.TODO(), api.mustDB(), consumer.ID, authentication.LoadUserConsumerOptions.WithAuthentifiedUser)
			require.NoError(t, err)

			ctx := context.TODO()
			ctx = context.WithValue(ctx, contextDriverManifest, &sdk.AuthDriverManifest{})
			ctx = context.WithValue(ctx, contextUserConsumer, consumer)

			err = api.checkGroupPermissions(ctx, &responseTracker{}, tt.args.groupName, tt.args.permissionLevel, nil)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func Test_checkTemplateSlugPermissions(t *testing.T) {
	api, db, _ := newTestAPI(t)

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

	organizations, err := organization.LoadOrganizations(context.TODO(), db)
	require.NoError(t, err)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prefix := sdk.RandomString(10) + "."

			if tt.setup.groupName != "" {
				groupAdmin := &sdk.AuthentifiedUser{
					Username: prefix + "auto-group-admin",
				}
				require.NoError(t, user.Insert(context.TODO(), db, groupAdmin))
				uo := user.UserOrganization{
					AuthentifiedUserID: groupAdmin.ID,
					OrganizationID:     organizations[0].ID,
				}
				require.NoError(t, user.InsertUserOrganization(context.TODO(), db, &uo))

				var err error
				groupAdmin, err = user.LoadByID(context.TODO(), db, groupAdmin.ID, user.LoadOptions.WithOrganization)
				require.NoError(t, err)
				tt.setup.groupName = prefix + tt.setup.groupName
				g := sdk.Group{
					Name: tt.setup.groupName,
				}
				require.NoError(t, group.Create(context.TODO(), db, &g, groupAdmin))
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
			err := api.checkTemplateSlugPermissions(ctx, &responseTracker{}, prefix+tt.args.templateSlug, sdk.PermissionRead, map[string]string{"permGroupName": prefix + tt.args.groupName})
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func Test_checkActionPermissions(t *testing.T) {
	api, db, _ := newTestAPI(t)

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

	assert.Error(t, api.checkActionPermissions(context.TODO(), &responseTracker{}, a.Name, sdk.PermissionRead, map[string]string{
		"permGroupName": sdk.RandomString(10),
	}), "error should be returned for random group name")
	assert.Error(t, api.checkActionPermissions(context.TODO(), &responseTracker{}, sdk.RandomString(10), sdk.PermissionRead, map[string]string{
		"permGroupName": g.Name,
	}), "error should be returned for random action name")
	assert.NoError(t, api.checkActionPermissions(context.TODO(), &responseTracker{}, a.Name, sdk.PermissionRead, map[string]string{
		"permGroupName": g.Name,
	}), "no error should be returned for the right group an action names")
}

func Test_checkActionBuiltinPermissions(t *testing.T) {
	api, db, _ := newTestAPI(t)

	scriptAction := assets.GetBuiltinOrPluginActionByName(t, db, "Script")

	assert.Error(t, api.checkActionBuiltinPermissions(context.TODO(), &responseTracker{}, sdk.RandomString(10), sdk.PermissionRead, nil), "error should be returned for random action name")
	assert.NoError(t, api.checkActionBuiltinPermissions(context.TODO(), &responseTracker{}, scriptAction.Name, sdk.PermissionRead, nil), "no error should be returned for valid action name")
}
