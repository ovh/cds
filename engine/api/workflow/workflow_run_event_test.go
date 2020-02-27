package workflow_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/repositoriesmanager"
	"github.com/ovh/cds/engine/api/services"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/sdk"
)

// Test ResyncCommitStatus with a notification where all is disabled.
// Must: no error returned, only list status is called
func TestResyncCommitStatusNotifDisabled(t *testing.T) {
	db, cache, end := test.SetupPG(t)
	defer end()

	ctx := context.TODO()

	pkey := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, cache, pkey, pkey)
	assert.NoError(t, repositoriesmanager.InsertForProject(db, proj, &sdk.ProjectVCSServer{
		Name: "gerrit",
		Data: map[string]string{
			"token":  "foo",
			"secret": "bar",
		},
	}))

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
	assert.NoError(t, application.Insert(db, cache, proj, &app))
	assert.NoError(t, repositoriesmanager.InsertForApplication(db, &app, proj.Key))

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
			WorkflowData: &sdk.WorkflowData{
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
	for _, s := range allSrv {
		if err := services.Delete(db, &s); err != nil {
			t.Fatalf("unable to delete service: %v", err)
		}
	}
	// Prepare VCS Mock
	mockVCSSservice, _ := assets.InsertService(t, db, "TestResyncCommitStatusNotifDisabled", services.TypeVCS)
	defer func() {
		_ = services.Delete(db, mockVCSSservice) // nolint
	}()

	statusCall := false
	//This is a mock for the repositories service
	services.HTTPClient = mock(
		func(r *http.Request) (*http.Response, error) {
			t.Logf("[MOCK] %s %v", r.Method, r.URL)
			body := new(bytes.Buffer)
			w := new(http.Response)
			enc := json.NewEncoder(body)
			w.Body = ioutil.NopCloser(body)

			switch r.URL.String() {
			case "/vcs/gerrit/repos/foo/myrepo/commits/6c3efde/statuses":
				if err := enc.Encode(nil); err != nil {
					return writeError(w, err)
				}
				statusCall = true
			default:
				t.Fail()
				return writeError(w, fmt.Errorf("route %s must not be called", r.URL.String()))
			}
			return w, nil
		},
	)

	err = workflow.ResyncCommitStatus(ctx, db, cache, proj, wr)
	assert.NoError(t, err)
	assert.True(t, statusCall)

}

// Test TestResyncCommitStatusSetStatus with a notification where all is disabled.
// Must: no error returned, setStatus must be called
func TestResyncCommitStatusSetStatus(t *testing.T) {
	db, cache, end := test.SetupPG(t)
	defer end()

	ctx := context.TODO()

	pkey := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, cache, pkey, pkey)
	assert.NoError(t, repositoriesmanager.InsertForProject(db, proj, &sdk.ProjectVCSServer{
		Name: "gerrit",
		Data: map[string]string{
			"token":  "foo",
			"secret": "bar",
		},
	}))

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
	assert.NoError(t, application.Insert(db, cache, proj, &app))
	assert.NoError(t, repositoriesmanager.InsertForApplication(db, &app, proj.Key))

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
			WorkflowData: &sdk.WorkflowData{
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

	allSrv, err := services.LoadAll(context.TODO(), db)
	for _, s := range allSrv {
		if err := services.Delete(db, &s); err != nil {
			t.Fatalf("unable to delete service: %v", err)
		}
	}
	// Prepare VCS Mock
	mockVCSSservice, _ := assets.InsertService(t, db, "TestResyncCommitStatusSetStatus", services.TypeVCS)
	defer func() {
		_ = services.Delete(db, mockVCSSservice) // nolint
	}()

	postStatusCall := false
	//This is a mock for the repositories service
	services.HTTPClient = mock(
		func(r *http.Request) (*http.Response, error) {
			t.Logf("[MOCK] %s %v", r.Method, r.URL)
			body := new(bytes.Buffer)
			w := new(http.Response)
			enc := json.NewEncoder(body)
			w.Body = ioutil.NopCloser(body)

			switch r.URL.String() {
			case "/vcs/gerrit/repos/foo/myrepo/commits/6c3efde/statuses":
				if err := enc.Encode(nil); err != nil {
					return writeError(w, err)
				}
			case "/vcs/gerrit":
				conf := sdk.VCSConfiguration{
					Type: "gerrit",
				}
				if err := enc.Encode(conf); err != nil {
					return writeError(w, err)
				}
			case "/vcs/gerrit/status":
				if err := enc.Encode(nil); err != nil {
					return writeError(w, err)
				}
				postStatusCall = true
			default:
				t.Logf("THIS MUS NOT BE CALLED %s", r.URL.String())
				t.Fail()
				return writeError(w, fmt.Errorf("route %s must not be called", r.URL.String()))
			}
			return w, nil
		},
	)

	err = workflow.ResyncCommitStatus(ctx, db, cache, proj, wr)
	assert.NoError(t, err)
	assert.True(t, postStatusCall)
}

// Test TestResyncCommitStatusCommentPR with a notification where all is disabled.
// Must: no error returned, postComment must be called
func TestResyncCommitStatusCommentPR(t *testing.T) {
	db, cache, end := test.SetupPG(t)
	defer end()

	ctx := context.TODO()

	pkey := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, cache, pkey, pkey)
	assert.NoError(t, repositoriesmanager.InsertForProject(db, proj, &sdk.ProjectVCSServer{
		Name: "gerrit",
		Data: map[string]string{
			"token":  "foo",
			"secret": "bar",
		},
	}))

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
	assert.NoError(t, application.Insert(db, cache, proj, &app))
	assert.NoError(t, repositoriesmanager.InsertForApplication(db, &app, proj.Key))

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
			WorkflowData: &sdk.WorkflowData{
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

	allSrv, err := services.LoadAll(context.TODO(), db)
	for _, s := range allSrv {
		if err := services.Delete(db, &s); err != nil {
			t.Fatalf("unable to delete service: %v", err)
		}
	}
	// Prepare VCS Mock
	mockVCSSservice, _ := assets.InsertService(t, db, "TestResyncCommitStatusCommentPR", services.TypeVCS)
	defer func() {
		_ = services.Delete(db, mockVCSSservice) // nolint
	}()

	commentCall := false
	//This is a mock for the repositories service
	services.HTTPClient = mock(
		func(r *http.Request) (*http.Response, error) {
			t.Logf("[MOCK] %s %v", r.Method, r.URL)
			body := new(bytes.Buffer)
			w := new(http.Response)
			enc := json.NewEncoder(body)
			w.Body = ioutil.NopCloser(body)

			switch r.URL.String() {
			case "/vcs/gerrit/repos/foo/myrepo/commits/6c3efde/statuses":
				if err := enc.Encode(nil); err != nil {
					return writeError(w, err)
				}
			case "/vcs/gerrit":
				conf := sdk.VCSConfiguration{
					Type: "gerrit",
				}
				if err := enc.Encode(conf); err != nil {
					return writeError(w, err)
				}
			case "/vcs/gerrit/repos/foo/myrepo/pullrequests/comments":
				commentCall = true
				dec := json.NewDecoder(r.Body)
				var request sdk.VCSPullRequestCommentRequest
				assert.NoError(t, dec.Decode(&request))
				assert.Equal(t, request.ChangeID, "MyGerritChangeId")
				assert.Equal(t, request.Message, "MyTemplate")
				if err := enc.Encode(nil); err != nil {
					return writeError(w, err)
				}
			default:
				t.Logf("THIS MUS NOT BE CALLED %s", r.URL.String())
				t.Fail()
				return writeError(w, fmt.Errorf("route %s must not be called", r.URL.String()))
			}
			return w, nil
		},
	)

	err = workflow.ResyncCommitStatus(ctx, db, cache, proj, wr)
	assert.NoError(t, err)
	assert.True(t, commentCall)
}
