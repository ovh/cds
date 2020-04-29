package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-gorp/gorp"
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

func TestPostUpdateWorkflowAsCodeHandler(t *testing.T) {
	api, db, _, end := newTestAPI(t)
	defer end()

	_, _ = assets.InsertService(t, db, t.Name()+"_HOOKS", services.TypeHooks)
	_, _ = assets.InsertService(t, db, t.Name()+"_VCS", services.TypeVCS)
	_, _ = assets.InsertService(t, db, t.Name()+"_REPO", services.TypeRepositories)

	u, pass := assets.InsertAdminUser(t, db)

	UUID := sdk.UUID()

	svcs, errS := services.LoadAll(context.TODO(), db)
	assert.NoError(t, errS)
	for _, s := range svcs {
		_ = services.Delete(db, &s) // nolint
	}

	a, _ := assets.InsertService(t, db, "Test_postWorkflowAsCodeHandlerVCS", services.TypeVCS)
	b, _ := assets.InsertService(t, db, "Test_postWorkflowAsCodeHandlerRepo", services.TypeRepositories)
	c, _ := assets.InsertService(t, db, "Test_postWorkflowAsCodeHandlerHook", services.TypeHooks)

	defer func() {
		services.Delete(db, a)
		services.Delete(db, b)
		services.Delete(db, c)
	}()

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
				ope.Status = sdk.OperationStatusProcessing
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
			case "/vcs/github/repos/foo/myrepo/branches":
				bs := make([]sdk.VCSBranch, 1)
				bs[0] = sdk.VCSBranch{
					DisplayID: "master",
					Default:   true,
				}
				if err := enc.Encode(bs); err != nil {
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
				if r.Method == http.MethodGet {
					vcsPRs := []sdk.VCSPullRequest{}
					if err := enc.Encode(vcsPRs); err != nil {
						return writeError(w, err)
					}
				} else {
					vcsPR := sdk.VCSPullRequest{
						URL: "myURL",
					}
					if err := enc.Encode(vcsPR); err != nil {
						return writeError(w, err)
					}
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
				t.Logf("[WRONG ROUTE] %s", r.URL.String())
				w.StatusCode = http.StatusNotFound
			}

			return w, nil
		},
	)

	assert.NoError(t, workflow.CreateBuiltinWorkflowHookModels(db))

	proj := createProject(t, db, api)
	pip := createPipeline(t, db, api, proj)
	app := createApplication(t, db, api, proj)

	repoModel, err := workflow.LoadHookModelByName(db, sdk.RepositoryWebHookModelName)
	assert.NoError(t, err)

	w := initWorkflow(t, db, proj, app, pip, repoModel)
	w.FromRepository = "myfromrepositoryurl"

	var errP error
	proj, errP = project.Load(api.mustDB(), proj.Key,
		project.LoadOptions.WithApplicationWithDeploymentStrategies,
		project.LoadOptions.WithPipelines,
		project.LoadOptions.WithEnvironments,
		project.LoadOptions.WithIntegrations,
	)
	assert.NoError(t, errP)
	if !assert.NoError(t, workflow.Insert(context.Background(), db, api.Cache, *proj, w)) {
		return
	}

	// Updating workflow
	w.WorkflowData.Node.Triggers = []sdk.NodeTrigger{
		{
			ChildNode: sdk.Node{
				Type: "fork",
				Name: "secondnode",
			},
		},
	}

	uri := api.Router.GetRoute("POST", api.postWorkflowAsCodeHandler, map[string]string{
		"key":              proj.Key,
		"permWorkflowName": w.Name,
	})

	req := assets.NewJWTAuthentifiedRequest(t, pass, "POST", uri, w)

	// Do the request
	wr := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(wr, req)
	assert.Equal(t, 200, wr.Code)
	myOpe := new(sdk.Operation)
	test.NoError(t, json.Unmarshal(wr.Body.Bytes(), myOpe))
	assert.NotEmpty(t, myOpe.UUID)

	retry := 0
	for {
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
		assert.NoError(t, json.Unmarshal(wrGet.Body.Bytes(), myOpeGet))
		if myOpeGet.Status < sdk.OperationStatusDone {
			time.Sleep(1 * time.Second)
			retry++

			if retry > 10 {
				t.Fail()
				break
			}
			continue
		}
		test.NoError(t, json.Unmarshal(wrGet.Body.Bytes(), myOpeGet))
		assert.Equal(t, "myURL", myOpeGet.Setup.Push.PRLink)
		break
	}

}

func TestPostMigrateWorkflowAsCodeHandler(t *testing.T) {
	api, db, _, end := newTestAPI(t)
	defer end()

	u, pass := assets.InsertAdminUser(t, db)

	UUID := sdk.UUID()

	a, _ := assets.InsertService(t, db, "Test_postWorkflowAsCodeHandlerVCS", services.TypeVCS)
	b, _ := assets.InsertService(t, db, "Test_postWorkflowAsCodeHandlerRepo", services.TypeRepositories)
	c, _ := assets.InsertService(t, db, "Test_postWorkflowAsCodeHandlerHook", services.TypeHooks)

	defer func() {
		_ = services.Delete(db, a)
		_ = services.Delete(db, b)
		_ = services.Delete(db, c)
	}()

	assert.NoError(t, workflow.CreateBuiltinWorkflowHookModels(db))

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
				ope.Status = sdk.OperationStatusProcessing
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
			case "/vcs/github/repos/foo/myrepo/branches":
				bs := make([]sdk.VCSBranch, 1)
				bs[0] = sdk.VCSBranch{
					DisplayID: "master",
					Default:   true,
				}
				if err := enc.Encode(bs); err != nil {
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
				if r.Method == http.MethodGet {
					vcsPRs := []sdk.VCSPullRequest{}
					if err := enc.Encode(vcsPRs); err != nil {
						return writeError(w, err)
					}
				} else {
					vcsPR := sdk.VCSPullRequest{
						URL: "myURL",
					}
					if err := enc.Encode(vcsPR); err != nil {
						return writeError(w, err)
					}
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
				for k, h := range hooks {
					if h.HookModelName == sdk.RepositoryWebHookModelName {
						cfg := hooks[k].Config
						cfg["webHookURL"] = sdk.WorkflowNodeHookConfigValue{
							Value:        "http://lolcat.host",
							Configurable: false,
						}
					}
				}
				if err := enc.Encode(hooks); err != nil {
					return writeError(w, err)
				}
			default:
				t.Logf("[WRONG ROUTE] %s", r.URL.String())
				w.StatusCode = http.StatusNotFound
			}

			return w, nil
		},
	)

	// Create Project
	proj := createProject(t, db, api)

	// Create Pipeline
	pip := createPipeline(t, db, api, proj)

	// Create Application
	app := createApplication(t, db, api, proj)

	repoModel, err := workflow.LoadHookModelByName(db, sdk.RepositoryWebHookModelName)
	assert.NoError(t, err)

	// Create Workflow
	w := initWorkflow(t, db, proj, app, pip, repoModel)

	var errP error
	proj, errP = project.Load(api.mustDB(), proj.Key,
		project.LoadOptions.WithApplicationWithDeploymentStrategies,
		project.LoadOptions.WithPipelines,
		project.LoadOptions.WithEnvironments,
		project.LoadOptions.WithIntegrations,
	)
	assert.NoError(t, errP)
	if !assert.NoError(t, workflow.Insert(context.Background(), db, api.Cache, *proj, w)) {
		return
	}

	t.Logf("%+v", w)

	uri := api.Router.GetRoute("POST", api.postWorkflowAsCodeHandler, map[string]string{
		"key":              proj.Key,
		"permWorkflowName": w.Name,
	})

	req := assets.NewJWTAuthentifiedRequest(t, pass, "POST", fmt.Sprintf("%s?migrate=true", uri), nil)

	// Do the request
	wr := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(wr, req)
	assert.Equal(t, 200, wr.Code)
	myOpe := new(sdk.Operation)
	test.NoError(t, json.Unmarshal(wr.Body.Bytes(), myOpe))
	assert.NotEmpty(t, myOpe.UUID)

	cpt := 0
	for {
		if cpt > 10 {
			t.Fail()
			break
		}
		cpt++
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

		if myOpeGet.Status < sdk.OperationStatusDone {
			continue
		}
		assert.Equal(t, "myURL", myOpeGet.Setup.Push.PRLink)
		break
	}
}

func createProject(t *testing.T, db *gorp.DbMap, api *API) *sdk.Project {
	// Create Project
	pkey := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, api.Cache, pkey, pkey)
	assert.NoError(t, repositoriesmanager.InsertForProject(db, proj, &sdk.ProjectVCSServer{
		Name: "github",
		Data: map[string]string{
			"token":  "foo",
			"secret": "bar",
		},
	}))
	return proj
}

func createPipeline(t *testing.T, db gorp.SqlExecutor, api *API, proj *sdk.Project) *sdk.Pipeline {
	pip := sdk.Pipeline{
		Name:      sdk.RandomString(10),
		ProjectID: proj.ID,
	}
	assert.NoError(t, pipeline.InsertPipeline(db, &pip))
	return &pip
}

func createApplication(t *testing.T, db gorp.SqlExecutor, api *API, proj *sdk.Project) *sdk.Application {
	app := sdk.Application{
		Name:               sdk.RandomString(10),
		ProjectID:          proj.ID,
		RepositoryFullname: "foo/myrepo",
		VCSServer:          "github",
	}
	assert.NoError(t, application.Insert(db, *proj, &app))
	assert.NoError(t, repositoriesmanager.InsertForApplication(db, &app, proj.Key))
	return &app
}

func initWorkflow(t *testing.T, db gorp.SqlExecutor, proj *sdk.Project, app *sdk.Application, pip *sdk.Pipeline, repoModel *sdk.WorkflowHookModel) *sdk.Workflow {
	w := sdk.Workflow{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       sdk.RandomString(10),
		WorkflowData: sdk.WorkflowData{
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
	w.WorkflowData.Node.Hooks[0].Config["webHookURL"] = sdk.WorkflowNodeHookConfigValue{Type: "string", Value: "http://ovh/com"}
	assert.NoError(t, workflow.RenameNode(context.Background(), db, &w))
	return &w
}
