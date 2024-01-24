package workflow_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/go-gorp/gorp"
	"github.com/golang/mock/gomock"
	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/repositoriesmanager"
	"github.com/ovh/cds/engine/api/services"
	"github.com/ovh/cds/engine/api/services/mock_services"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/engine/api/vcs"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/sdk"
	"github.com/stretchr/testify/assert"
)

// Test ResyncCommitStatus with a notification where all is disabled.
// Must: no error returned, only list status is called
func TestResyncCommitStatusNotifDisabled(t *testing.T) {
	db, cache := test.SetupPG(t)

	ctx := context.TODO()

	pkey := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, cache, pkey, pkey)
	vcsServer := &sdk.VCSProject{
		ProjectID: proj.ID,
		Name:      "gerrit",
		Type:      sdk.VCSTypeGerrit,
	}
	assert.NoError(t, vcs.Insert(context.TODO(), db, vcsServer))

	// Create Application
	app := sdk.Application{
		Name:               sdk.RandomString(10),
		ProjectID:          proj.ID,
		RepositoryFullname: "foo/myrepo",
		VCSServer:          "gerrit",
		RepositoryStrategy: sdk.RepositoryStrategy{
			ConnectionType: "ssh",
		},
	}
	assert.NoError(t, application.Insert(db, *proj, &app))
	assert.NoError(t, repositoriesmanager.InsertForApplication(db, &app))

	tr := true
	wr := &sdk.WorkflowRun{
		WorkflowNodeRuns: map[int64][]sdk.WorkflowNodeRun{
			1: {
				{
					ID:             1,
					ApplicationID:  app.ID,
					Status:         sdk.StatusSuccess,
					WorkflowNodeID: 1,
					VCSHash:        "6c3efde",
				},
			},
		},
		Workflow: sdk.Workflow{
			WorkflowData: sdk.WorkflowData{
				Node: sdk.Node{
					ID: 1,
					Context: &sdk.NodeContext{
						ApplicationID: app.ID,
					},
				},
			},
			Applications: map[int64]sdk.Application{
				app.ID: app,
			},
			Notifications: []sdk.WorkflowNotification{
				{
					Settings: sdk.UserNotificationSettings{
						Template: &sdk.UserNotificationTemplate{
							DisableComment: &tr,
							DisableStatus:  &tr,
						},
					},
					Type: "vcs",
				},
			},
		},
	}

	allSrv, err := services.LoadAll(context.TODO(), db)
	assert.NoError(t, err)
	for _, s := range allSrv {
		if err := services.Delete(db, &s); err != nil {
			t.Fatalf("unable to delete service: %v", err)
		}
	}
	// Setup a mock for all services called by the API
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	servicesClients := mock_services.NewMockClient(ctrl)
	services.NewClient = func(_ gorp.SqlExecutor, _ []sdk.Service) services.Client {
		return servicesClients
	}
	defer func() {
		services.NewClient = services.NewDefaultClient
	}()

	servicesClients.EXPECT().
		DoJSONRequest(gomock.Any(), "GET", "/vcs/gerrit/repos/foo/myrepo/commits/6c3efde/statuses",
			gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil, 201, nil)

	err = workflow.ResyncCommitStatus(ctx, db.DbMap, cache, *proj, wr)
	assert.NoError(t, err)
}

// Test TestResyncCommitStatusSetStatus with a notification where all is disabled.
// Must: no error returned, setStatus must be called
func TestResyncCommitStatusSetStatus(t *testing.T) {
	db, cache := test.SetupPG(t)

	ctx := context.TODO()

	pkey := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, cache, pkey, pkey)
	vcsServer := &sdk.VCSProject{
		ProjectID: proj.ID,
		Name:      "gerrit",
		Type:      sdk.VCSTypeGerrit,
	}
	assert.NoError(t, vcs.Insert(context.TODO(), db, vcsServer))

	// Create Application
	app := sdk.Application{
		Name:               sdk.RandomString(10),
		ProjectID:          proj.ID,
		RepositoryFullname: "foo/myrepo",
		VCSServer:          "gerrit",
		RepositoryStrategy: sdk.RepositoryStrategy{
			ConnectionType: "ssh",
		},
	}
	assert.NoError(t, application.Insert(db, *proj, &app))
	assert.NoError(t, repositoriesmanager.InsertForApplication(db, &app))

	tr := true
	wr := &sdk.WorkflowRun{
		WorkflowNodeRuns: map[int64][]sdk.WorkflowNodeRun{
			1: {
				{
					ID:             1,
					ApplicationID:  app.ID,
					Status:         sdk.StatusSuccess,
					WorkflowNodeID: 1,
					VCSHash:        "6c3efde",
				},
			},
		},
		Workflow: sdk.Workflow{
			WorkflowData: sdk.WorkflowData{
				Node: sdk.Node{
					ID: 1,
					Context: &sdk.NodeContext{
						ApplicationID: app.ID,
					},
				},
			},
			Applications: map[int64]sdk.Application{
				app.ID: app,
			},
			Notifications: []sdk.WorkflowNotification{
				{
					Settings: sdk.UserNotificationSettings{
						Template: &sdk.UserNotificationTemplate{
							DisableComment: &tr,
						},
					},
					Type: "vcs",
				},
			},
		},
	}

	// Setup a mock for all services called by the API
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	servicesClients := mock_services.NewMockClient(ctrl)
	services.NewClient = func(_ gorp.SqlExecutor, _ []sdk.Service) services.Client {
		return servicesClients
	}
	defer func() {
		services.NewClient = services.NewDefaultClient
	}()

	servicesClients.EXPECT().
		DoJSONRequest(gomock.Any(), "GET", "/vcs/gerrit/repos/foo/myrepo/commits/6c3efde/statuses", gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil, 201, nil).MaxTimes(1)

	servicesClients.EXPECT().
		DoJSONRequest(gomock.Any(), "GET", "/vcs/gerrit", gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, method, path string, in interface{}, out interface{}, mods ...interface{}) (http.Header, int, error) {
			vcs := sdk.VCSConfiguration{Type: "gerrit"}
			*(out.(*sdk.VCSConfiguration)) = vcs
			return nil, 200, nil
		}).MaxTimes(1)

	servicesClients.EXPECT().
		DoJSONRequest(gomock.Any(), "POST", "/vcs/gerrit/status", gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil, 201, nil)

	err := workflow.ResyncCommitStatus(ctx, db.DbMap, cache, *proj, wr)
	assert.NoError(t, err)
}

// Test TestResyncCommitStatusCommentPR with a notification where all is disabled.
// Must: no error returned, postComment must be called
func TestResyncCommitStatusCommentPR(t *testing.T) {
	db, cache := test.SetupPG(t)

	ctx := context.TODO()

	pkey := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, cache, pkey, pkey)
	vcsServer := &sdk.VCSProject{
		ProjectID: proj.ID,
		Name:      "gerrit",
		Type:      sdk.VCSTypeGerrit,
	}
	assert.NoError(t, vcs.Insert(context.TODO(), db, vcsServer))

	// Create Application
	app := sdk.Application{
		Name:               sdk.RandomString(10),
		ProjectID:          proj.ID,
		RepositoryFullname: "foo/myrepo",
		VCSServer:          "gerrit",
		RepositoryStrategy: sdk.RepositoryStrategy{
			ConnectionType: "ssh",
		},
	}
	assert.NoError(t, application.Insert(db, *proj, &app))
	assert.NoError(t, repositoriesmanager.InsertForApplication(db, &app))

	tr := true
	fls := false
	wr := &sdk.WorkflowRun{
		WorkflowNodeRuns: map[int64][]sdk.WorkflowNodeRun{
			1: {
				{
					ID:             1,
					ApplicationID:  app.ID,
					Status:         sdk.StatusFail,
					WorkflowNodeID: 1,
					VCSHash:        "6c3efde",
					BuildParameters: []sdk.Parameter{
						{
							Name:  "gerrit.change.id",
							Type:  "string",
							Value: "MyGerritChangeId",
						},
					},
				},
			},
		},
		Workflow: sdk.Workflow{
			WorkflowData: sdk.WorkflowData{
				Node: sdk.Node{
					ID: 1,
					Context: &sdk.NodeContext{
						ApplicationID: app.ID,
					},
				},
			},
			Applications: map[int64]sdk.Application{
				app.ID: app,
			},
			Notifications: []sdk.WorkflowNotification{
				{
					Settings: sdk.UserNotificationSettings{
						Template: &sdk.UserNotificationTemplate{
							DisableComment: &fls,
							DisableStatus:  &tr,
							Body:           "MyTemplate",
						},
					},
					Type: "vcs",
				},
			},
		},
	}

	// Setup a mock for all services called by the API
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	servicesClients := mock_services.NewMockClient(ctrl)
	services.NewClient = func(_ gorp.SqlExecutor, _ []sdk.Service) services.Client {
		return servicesClients
	}
	defer func() {
		services.NewClient = services.NewDefaultClient
	}()

	servicesClients.EXPECT().
		DoJSONRequest(gomock.Any(), "GET", "/vcs/gerrit/repos/foo/myrepo/commits/6c3efde/statuses", gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil, 201, nil).MaxTimes(1)

	servicesClients.EXPECT().
		DoJSONRequest(gomock.Any(), "GET", "/vcs/gerrit", gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, method, path string, in interface{}, out interface{}, mods ...interface{}) (http.Header, int, error) {
			vcs := sdk.VCSConfiguration{Type: "gerrit"}
			*(out.(*sdk.VCSConfiguration)) = vcs
			return nil, 200, nil
		}).MaxTimes(1)

	servicesClients.EXPECT().
		DoJSONRequest(gomock.Any(), "POST", "/vcs/gerrit/repos/foo/myrepo/pullrequests/comments",
			gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, method, path string, in interface{}, out interface{}, _ interface{}) (http.Header, int, error) {
			assert.Equal(t, in.(sdk.VCSPullRequestCommentRequest).ChangeID, "MyGerritChangeId")
			assert.Equal(t, in.(sdk.VCSPullRequestCommentRequest).Message, "MyTemplate")
			return nil, 200, nil
		}).MaxTimes(1)

	err := workflow.ResyncCommitStatus(ctx, db.DbMap, cache, *proj, wr)
	assert.NoError(t, err)
}

// Test TestResyncCommitStatusCommentPRNotTerminated with a notification where all is disabled.
// Must: no error returned, postComment must be called
func TestResyncCommitStatusCommentPRNotTerminated(t *testing.T) {
	db, cache := test.SetupPG(t)

	ctx := context.TODO()

	pkey := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, cache, pkey, pkey)
	vcsServer := &sdk.VCSProject{
		ProjectID: proj.ID,
		Name:      "gerrit",
		Type:      sdk.VCSTypeGerrit,
	}
	assert.NoError(t, vcs.Insert(context.TODO(), db, vcsServer))

	// Create Application
	app := sdk.Application{
		Name:               sdk.RandomString(10),
		ProjectID:          proj.ID,
		RepositoryFullname: "foo/myrepo",
		VCSServer:          "gerrit",
		RepositoryStrategy: sdk.RepositoryStrategy{
			ConnectionType: "ssh",
		},
	}
	assert.NoError(t, application.Insert(db, *proj, &app))
	assert.NoError(t, repositoriesmanager.InsertForApplication(db, &app))

	tr := true
	fls := false
	wr := &sdk.WorkflowRun{
		WorkflowNodeRuns: map[int64][]sdk.WorkflowNodeRun{
			1: {
				{
					ID:             1,
					ApplicationID:  app.ID,
					Status:         sdk.StatusBuilding,
					WorkflowNodeID: 1,
					VCSHash:        "6c3efde",
					BuildParameters: []sdk.Parameter{
						{
							Name:  "gerrit.change.id",
							Type:  "string",
							Value: "MyGerritChangeId",
						},
					},
				},
			},
		},
		Workflow: sdk.Workflow{
			WorkflowData: sdk.WorkflowData{
				Node: sdk.Node{
					ID: 1,
					Context: &sdk.NodeContext{
						ApplicationID: app.ID,
					},
				},
			},
			Applications: map[int64]sdk.Application{
				app.ID: app,
			},
			Notifications: []sdk.WorkflowNotification{
				{
					Settings: sdk.UserNotificationSettings{
						Template: &sdk.UserNotificationTemplate{
							DisableComment: &fls,
							DisableStatus:  &tr,
							Body:           "MyTemplate",
						},
					},
					Type: "vcs",
				},
			},
		},
	}

	// Setup a mock for all services called by the API
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	servicesClients := mock_services.NewMockClient(ctrl)
	services.NewClient = func(_ gorp.SqlExecutor, _ []sdk.Service) services.Client {
		return servicesClients
	}
	defer func() {
		services.NewClient = services.NewDefaultClient
	}()

	servicesClients.EXPECT().
		DoJSONRequest(gomock.Any(), "GET", "/vcs/gerrit/repos/foo/myrepo/commits/6c3efde/statuses",
			gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil, 201, nil).MaxTimes(1)

	servicesClients.EXPECT().
		DoJSONRequest(gomock.Any(), "GET", "/vcs/gerrit",
			gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, method, path string, in interface{}, out interface{}) (http.Header, int, error) {
			vcs := sdk.VCSConfiguration{Type: "gerrit"}
			*(out.(*sdk.VCSConfiguration)) = vcs
			return nil, 200, nil
		}).MaxTimes(1)

	err := workflow.ResyncCommitStatus(ctx, db.DbMap, cache, *proj, wr)
	assert.NoError(t, err)
}

// Test TestResyncCommitStatus with a notification where all is disabled.
// Must: no error returned, postComment must be called
func TestResyncCommitStatusCommitCache(t *testing.T) {
	db, cache := test.SetupPG(t)

	ctx := context.TODO()

	pkey := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, cache, pkey, pkey)
	vcsServer := &sdk.VCSProject{
		ProjectID: proj.ID,
		Name:      "gerrit",
		Type:      sdk.VCSTypeGerrit,
	}
	assert.NoError(t, vcs.Insert(context.TODO(), db, vcsServer))

	// Create Application
	app := sdk.Application{
		Name:               sdk.RandomString(10),
		ProjectID:          proj.ID,
		RepositoryFullname: "foo/myrepo",
		VCSServer:          "gerrit",
		RepositoryStrategy: sdk.RepositoryStrategy{
			ConnectionType: "ssh",
		},
	}
	assert.NoError(t, application.Insert(db, *proj, &app))
	assert.NoError(t, repositoriesmanager.InsertForApplication(db, &app))

	tr := true
	fls := false
	wr := &sdk.WorkflowRun{
		WorkflowNodeRuns: map[int64][]sdk.WorkflowNodeRun{
			1: {
				{
					ID:             1,
					ApplicationID:  app.ID,
					Status:         sdk.StatusFail,
					WorkflowNodeID: 1,
					VCSHash:        "6c3efde",
					BuildParameters: []sdk.Parameter{
						{
							Name:  "gerrit.change.id",
							Type:  "string",
							Value: "MyGerritChangeId",
						},
					},
				},
			},
		},
		Workflow: sdk.Workflow{
			WorkflowData: sdk.WorkflowData{
				Node: sdk.Node{
					ID: 1,
					Context: &sdk.NodeContext{
						ApplicationID: app.ID,
					},
				},
			},
			Applications: map[int64]sdk.Application{
				app.ID: app,
			},
			Notifications: []sdk.WorkflowNotification{
				{
					Settings: sdk.UserNotificationSettings{
						Template: &sdk.UserNotificationTemplate{
							DisableComment: &fls,
							DisableStatus:  &tr,
							Body:           "MyTemplate",
						},
					},
					Type: "vcs",
				},
			},
		},
	}

	// Setup a mock for all services called by the API
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	servicesClients := mock_services.NewMockClient(ctrl)
	services.NewClient = func(_ gorp.SqlExecutor, _ []sdk.Service) services.Client {
		return servicesClients
	}
	defer func() {
		services.NewClient = services.NewDefaultClient
	}()

	servicesClients.EXPECT().
		DoJSONRequest(gomock.Any(), "GET", "/vcs/gerrit/repos/foo/myrepo/commits/6c3efde/statuses",
			gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, method, path string, in interface{}, out interface{}, args ...interface{}) (http.Header, int, error) {
			ss := []sdk.VCSCommitStatus{
				{
					State: "Success",
					Ref:   "6c3efde",
				},
			}
			*(out.(*[]sdk.VCSCommitStatus)) = ss
			return nil, 200, nil
		}).MaxTimes(1)

	servicesClients.EXPECT().
		DoJSONRequest(gomock.Any(), "GET", "/vcs/gerrit", gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, method, path string, in interface{}, out interface{}, mods ...interface{}) (http.Header, int, error) {
			vcs := sdk.VCSConfiguration{Type: "gerrit"}
			*(out.(*sdk.VCSConfiguration)) = vcs
			return nil, 200, nil
		}).MaxTimes(1)

	servicesClients.EXPECT().
		DoJSONRequest(gomock.Any(), "POST", "/vcs/gerrit/repos/foo/myrepo/pullrequests/comments",
			gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, method, path string, in interface{}, out interface{}, _ interface{}) (http.Header, int, error) {
			assert.Equal(t, in.(sdk.VCSPullRequestCommentRequest).ChangeID, "MyGerritChangeId")
			assert.Equal(t, in.(sdk.VCSPullRequestCommentRequest).Message, "MyTemplate")
			return nil, 200, nil
		}).MaxTimes(1)
	e := workflow.VCSEventMessenger{}
	err := e.SendVCSEvent(ctx, db.DbMap, cache, *proj, *wr, wr.WorkflowNodeRuns[1][0])
	assert.NoError(t, err)
}

func Test_isSameCommit(t *testing.T) {
	type args struct {
		sha1  string
		sha1b string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "same",
			args: args{sha1: "aaaaaa", sha1b: "aaaaaa"},
			want: true,
		},
		{
			name: "same",
			args: args{sha1: "4e269fccb82a", sha1b: "4e269fccb82a1b98a510b172b2c8db8ec9b4abb0"},
			want: true,
		},
		{
			name: "same",
			args: args{sha1: "4e269fccb82a1b98a510b172b2c8db8ec9b4abb0", sha1b: "4e269fccb82a"},
			want: true,
		},
		{
			name: "not same",
			args: args{sha1: "aa4e269fccb82a1b98a510b172b2c8db8ec9b4abb0", sha1b: "aa4e269fccb82a"},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := workflow.IsSameCommit(tt.args.sha1, tt.args.sha1b); got != tt.want {
				t.Errorf("isSameCommit() = %v, want %v", got, tt.want)
			}
		})
	}
}
