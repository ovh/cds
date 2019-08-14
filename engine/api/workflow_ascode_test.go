package api

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/repositoriesmanager"
	"github.com/ovh/cds/engine/api/services"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/sdk"
)

func TestPostWorkflowAsCodeHandler(t *testing.T) {
	api, db, _, end := newTestAPI(t)
	defer end()

	u, pass := assets.InsertAdminUser(db)

	UUID := sdk.UUID()

	mockServiceVCS := &sdk.Service{Name: "Test_postWorkflowAsCodeHandlerVCS", Type: services.TypeVCS}
	_ = services.Delete(db, mockServiceVCS)
	test.NoError(t, services.Insert(db, mockServiceVCS))

	mockServiceRepositories := &sdk.Service{Name: "Test_postWorkflowAsCodeHandlerRepo", Type: services.TypeRepositories}
	_ = services.Delete(db, mockServiceRepositories)
	test.NoError(t, services.Insert(db, mockServiceRepositories))

	mockServiceHook := &sdk.Service{Name: "Test_postWorkflowAsCodeHandlerHook", Type: services.TypeHooks}
	_ = services.Delete(db, mockServiceHook)
	test.NoError(t, services.Insert(db, mockServiceHook))

	//This is a mock for the repositories service
	services.HTTPClient = mock(
		func(r *http.Request) (*http.Response, error) {
			body := new(bytes.Buffer)
			w := new(http.Response)
			enc := json.NewEncoder(body)
			w.Body = ioutil.NopCloser(body)
			w.StatusCode = http.StatusOK
			switch r.URL.String() {
			case "/operations":
				ope := new(sdk.Operation)
				ope.UUID = UUID
				ope.Status = sdk.OperationStatusDone
				if err := enc.Encode(ope); err != nil {
					return writeError(w, err)
				}
			case "/operations/" + UUID:
				ope := new(sdk.Operation)
				ope.UUID = UUID
				ope.Status = sdk.OperationStatusDone
				if err := enc.Encode(ope); err != nil {
					return writeError(w, err)
				}
			case "/vcs/github/repos/foo/myrepo":
				vcsRepo := sdk.VCSRepo{
					Name:         "foo/myrepo",
					SSHCloneURL:  "git:foo",
					HTTPCloneURL: "https:foo",
				}
				if err := enc.Encode(vcsRepo); err != nil {
					return writeError(w, err)
				}
			case "/vcs/github/webhooks":
				hookInfo := repositoriesmanager.WebhooksInfos{
					WebhooksSupported: true,
					WebhooksDisabled:  false,
				}
				if err := enc.Encode(hookInfo); err != nil {
					return writeError(w, err)
				}
			case "/vcs/github/repos/foo/myrepo/hooks":
				hook := sdk.VCSHook{
					ID: "myod",
				}
				if err := enc.Encode(hook); err != nil {
					return writeError(w, err)
				}
			case "/vcs/github/repos/foo/myrepo/pullrequests":
				vcsPR := sdk.VCSPullRequest{
					URL: "myURL",
				}
				if err := enc.Encode(vcsPR); err != nil {
					return writeError(w, err)
				}
			case "/task/bulk":
				var hooks map[string]sdk.NodeHook
				bts, err := ioutil.ReadAll(r.Body)
				if err != nil {
					return writeError(w, err)
				}
				if err := json.Unmarshal(bts, &hooks); err != nil {
					return writeError(w, err)
				}
				if err := enc.Encode(hooks); err != nil {
					return writeError(w, err)
				}
			default:
				w.StatusCode = http.StatusNotFound
			}

			return w, nil
		},
	)

	// Create Project
	pkey := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, api.Cache, pkey, pkey, u)
	assert.NoError(t, repositoriesmanager.InsertForProject(db, proj, &sdk.ProjectVCSServer{
		Name: "github",
		Data: map[string]string{
			"token":  "foo",
			"secret": "bar",
		},
	}))

	// Create Pipeline
	pip := sdk.Pipeline{
		Name:      sdk.RandomString(10),
		ProjectID: proj.ID,
	}
	assert.NoError(t, pipeline.InsertPipeline(db, api.Cache, proj, &pip, u))

	// Create Application
	app := sdk.Application{
		Name:               sdk.RandomString(10),
		ProjectID:          proj.ID,
		RepositoryFullname: "foo/myrepo",
		VCSServer:          "github",
	}
	assert.NoError(t, application.Insert(db, api.Cache, proj, &app, u))
	assert.NoError(t, repositoriesmanager.InsertForApplication(db, &app, proj.Key))

	repoModel, err := workflow.LoadHookModelByName(db, sdk.RepositoryWebHookModelName)
	assert.NoError(t, err)

	// Create Workflow
	w := sdk.Workflow{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       sdk.RandomString(10),
		WorkflowData: &sdk.WorkflowData{
			Node: sdk.Node{
				Name: "root",
				Type: sdk.NodeTypePipeline,
				Context: &sdk.NodeContext{
					PipelineID:    pip.ID,
					ApplicationID: app.ID,
				},
				Hooks: []sdk.NodeHook{
					{
						HookModelName: sdk.RepositoryWebHookModelName,
						UUID:          sdk.RandomString(10),
						Config:        sdk.RepositoryWebHookModel.DefaultConfig.Clone(),
						HookModelID:   repoModel.ID,
					},
				},
			},
		},
	}
	assert.NoError(t, workflow.RenameNode(db, &w))

	var errP error
	proj, errP = project.Load(api.mustDB(), api.Cache, proj.Key, u, project.LoadOptions.WithApplicationWithDeploymentStrategies, project.LoadOptions.WithPipelines, project.LoadOptions.WithEnvironments, project.LoadOptions.WithIntegrations)
	assert.NoError(t, errP)
	if !assert.NoError(t, workflow.Insert(db, api.Cache, &w, proj, u)) {
		return
	}

	t.Logf("%+v", w)

	uri := api.Router.GetRoute("POST", api.postWorkflowAsCodeHandler, map[string]string{
		"key":              proj.Key,
		"permWorkflowName": w.Name,
	})

	req, err := http.NewRequest("POST", uri, nil)
	test.NoError(t, err)
	assets.AuthentifyRequest(t, req, u, pass)

	// Do the request
	wr := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(wr, req)
	assert.Equal(t, 200, wr.Code)
	myOpe := new(sdk.Operation)
	test.NoError(t, json.Unmarshal(wr.Body.Bytes(), myOpe))
	assert.NotEmpty(t, myOpe.UUID)

	time.Sleep(2 * time.Second)

	// Get operation
	uriGET := api.Router.GetRoute("GET", api.getWorkflowAsCodeHandler, map[string]string{
		"key":              proj.Key,
		"permWorkflowName": w.Name,
		"uuid":             myOpe.UUID,
	})
	reqGET, err := http.NewRequest("GET", uriGET, nil)
	test.NoError(t, err)
	assets.AuthentifyRequest(t, reqGET, u, pass)
	wrGet := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(wrGet, reqGET)
	assert.Equal(t, 200, wrGet.Code)
	myOpeGet := new(sdk.Operation)
	test.NoError(t, json.Unmarshal(wrGet.Body.Bytes(), myOpeGet))
	assert.Equal(t, "myURL", myOpeGet.Setup.Push.PRLink)
}
