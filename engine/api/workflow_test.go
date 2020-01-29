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

	"github.com/ovh/cds/engine/api/repositoriesmanager"
	"github.com/ovh/cds/engine/api/services"
	"github.com/ovh/cds/engine/service"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/authentication"
	"github.com/ovh/cds/engine/api/authentication/builtin"
	"github.com/ovh/cds/engine/api/bootstrap"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/integration"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient"
	"github.com/ovh/cds/sdk/exportentities"
)

func Test_getWorkflowsHandler(t *testing.T) {
	api, db, _, end := newTestAPI(t, bootstrap.InitiliazeDB)
	defer end()
	u, pass := assets.InsertLambdaUser(t, api.mustDB())
	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))
	require.NoError(t, group.InsertLinkGroupUser(context.TODO(), api.mustDB(), &group.LinkGroupUser{
		GroupID:            proj.ProjectGroups[0].Group.ID,
		AuthentifiedUserID: u.ID,
		Admin:              true,
	}))

	pip := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip1",
	}

	test.NoError(t, pipeline.InsertPipeline(api.mustDB(), api.Cache, proj, &pip))

	wf := sdk.Workflow{
		Name:       "workflow1",
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		WorkflowData: &sdk.WorkflowData{
			Node: sdk.Node{
				Name: "root",
				Context: &sdk.NodeContext{
					PipelineID: pip.ID,
				},
			},
		},
	}

	test.NoError(t, workflow.Insert(context.TODO(), api.mustDB(), api.Cache, &wf, proj))

	vars := map[string]string{
		"permProjectKey": proj.Key,
	}
	uri := api.Router.GetRoute("GET", api.getWorkflowsHandler, vars)
	test.NotEmpty(t, uri)
	req := assets.NewAuthentifiedRequest(t, u, pass, "GET", uri, nil)

	//Do the request
	w := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)

	wfList := []sdk.Workflow{}
	test.NoError(t, json.Unmarshal(w.Body.Bytes(), &wfList))
	for _, w := range wfList {
		assert.Equal(t, true, w.Permissions.Readable, "readable should be true")
		assert.Equal(t, true, w.Permissions.Writable, "writable should be true")
		assert.Equal(t, true, w.Permissions.Executable, "writable should be true")
	}

	var err error

	userAdmin, passAdmin := assets.InsertAdminUser(t, db)
	uri = api.Router.GetRoute("GET", api.getWorkflowsHandler, vars)
	req, err = http.NewRequest("GET", uri, nil)
	test.NoError(t, err)
	assets.AuthentifyRequest(t, req, userAdmin, passAdmin)

	// Do the request
	w = httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)

	wfList = []sdk.Workflow{}
	test.NoError(t, json.Unmarshal(w.Body.Bytes(), &wfList))
	for _, w := range wfList {
		assert.Equal(t, true, w.Permissions.Readable, "readable should be true")
		assert.Equal(t, true, w.Permissions.Writable, "writable should be true")
		assert.Equal(t, true, w.Permissions.Executable, "executable should be true")
	}

	userMaintainer, passMaintainer := assets.InsertMaintainerUser(t, db)
	uri = api.Router.GetRoute("GET", api.getWorkflowsHandler, vars)
	req, err = http.NewRequest("GET", uri, nil)
	test.NoError(t, err)
	assets.AuthentifyRequest(t, req, userMaintainer, passMaintainer)

	// Do the request
	w = httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)

	wfList = []sdk.Workflow{}
	test.NoError(t, json.Unmarshal(w.Body.Bytes(), &wfList))
	for _, w := range wfList {
		assert.Equal(t, true, w.Permissions.Readable, "readable should be true")
		assert.Equal(t, false, w.Permissions.Writable, "writable should be false")
		assert.Equal(t, false, w.Permissions.Executable, "executable should be false")
	}
}

func Test_getWorkflowNotificationsConditionsHandler(t *testing.T) {
	api, db, router, end := newTestAPI(t, bootstrap.InitiliazeDB)
	defer end()
	u, pass := assets.InsertAdminUser(t, db)
	consumer, _ := authentication.LoadConsumerByTypeAndUserID(context.TODO(), db, sdk.ConsumerLocal, u.ID, authentication.LoadConsumerOptions.WithAuthentifiedUser)
	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, api.Cache, key, key)

	//First pipeline
	pip := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip1",
	}
	test.NoError(t, pipeline.InsertPipeline(db, api.Cache, proj, &pip))

	s := sdk.NewStage("stage 1")
	s.Enabled = true
	s.PipelineID = pip.ID
	test.NoError(t, pipeline.InsertStage(db, s))
	j := &sdk.Job{
		Enabled: true,
		Action: sdk.Action{
			Enabled: true,
		},
	}
	test.NoError(t, pipeline.InsertJob(db, j, s.ID, &pip))
	s.Jobs = append(s.Jobs, *j)

	pip.Stages = append(pip.Stages, *s)

	//Second pipeline
	pip2 := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip2",
	}
	test.NoError(t, pipeline.InsertPipeline(db, api.Cache, proj, &pip2))
	s = sdk.NewStage("stage 1")
	s.Enabled = true
	s.PipelineID = pip2.ID
	test.NoError(t, pipeline.InsertStage(db, s))
	j = &sdk.Job{
		Enabled: true,
		Action: sdk.Action{
			Enabled: true,
		},
	}
	test.NoError(t, pipeline.InsertJob(db, j, s.ID, &pip2))
	s.Jobs = append(s.Jobs, *j)

	w := sdk.Workflow{
		Name:       "test_1",
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		WorkflowData: &sdk.WorkflowData{
			Node: sdk.Node{
				Name: "root",
				Type: sdk.NodeTypePipeline,
				Context: &sdk.NodeContext{
					PipelineID: pip.ID,
				},
				Triggers: []sdk.NodeTrigger{
					{
						ChildNode: sdk.Node{
							Name: "child",
							Type: sdk.NodeTypePipeline,
							Context: &sdk.NodeContext{
								PipelineID: pip.ID,
							},
						},
					},
				},
			},
		},
	}

	proj2, errP := project.Load(api.mustDB(), api.Cache, proj.Key, project.LoadOptions.WithPipelines, project.LoadOptions.WithGroups, project.LoadOptions.WithIntegrations)
	test.NoError(t, errP)

	test.NoError(t, workflow.Insert(context.TODO(), db, api.Cache, &w, proj2))
	w1, err := workflow.Load(context.TODO(), db, api.Cache, proj, "test_1", workflow.LoadOptions{})
	test.NoError(t, err)

	wrCreate, err := workflow.CreateRun(db, w1, nil, u)
	assert.NoError(t, err)
	wrCreate.Workflow = *w1
	_, errMR := workflow.StartWorkflowRun(context.TODO(), db, api.Cache, proj, wrCreate, &sdk.WorkflowRunPostHandlerOption{
		Manual: &sdk.WorkflowNodeRunManual{Username: u.GetUsername()},
	}, consumer, nil)
	if errMR != nil {
		test.NoError(t, errMR)
	}
	//Prepare request
	vars := map[string]string{
		"key":              proj.Key,
		"permWorkflowName": w1.Name,
	}
	uri := router.GetRoute("GET", api.getWorkflowNotificationsConditionsHandler, vars)
	test.NotEmpty(t, uri)
	req := assets.NewAuthentifiedRequest(t, u, pass, "GET", uri, vars)

	//Do the request
	writer := httptest.NewRecorder()
	router.Mux.ServeHTTP(writer, req)
	assert.Equal(t, 200, writer.Code)

	data := struct {
		Operators      map[string]string `json:"operators"`
		ConditionNames []string          `json:"names"`
	}{}
	test.NoError(t, json.Unmarshal(writer.Body.Bytes(), &data))

	found := false
	for _, conditionName := range data.ConditionNames {
		if conditionName == "cds.ui.pipeline.run" {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("cannot find cds.ui.pipeline.run variable in response : %+v", data)
	}
}

func Test_getWorkflowHandler(t *testing.T) {
	api, db, router, end := newTestAPI(t)
	defer end()

	// Init user
	u, pass := assets.InsertAdminUser(t, api.mustDB())
	// Init project
	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, api.Cache, key, key)
	//Prepare request
	vars := map[string]string{
		"key":              proj.Key,
		"permWorkflowName": "workflow1",
	}
	uri := router.GetRoute("GET", api.getWorkflowHandler, vars)
	test.NotEmpty(t, uri)

	req := assets.NewAuthentifiedRequest(t, u, pass, "GET", uri, nil)
	//Do the request
	w := httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)
	assert.Equal(t, 404, w.Code)
}

func Test_getWorkflowHandler_CheckPermission(t *testing.T) {
	api, db, _, end := newTestAPI(t, bootstrap.InitiliazeDB)
	defer end()
	u, pass := assets.InsertLambdaUser(t, api.mustDB())
	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))
	require.NoError(t, group.InsertLinkGroupUser(context.TODO(), api.mustDB(), &group.LinkGroupUser{
		GroupID:            proj.ProjectGroups[0].Group.ID,
		AuthentifiedUserID: u.ID,
		Admin:              true,
	}))

	pip := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip1",
	}

	test.NoError(t, pipeline.InsertPipeline(api.mustDB(), api.Cache, proj, &pip))

	wf := sdk.Workflow{
		Name:       "workflow1",
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		WorkflowData: &sdk.WorkflowData{
			Node: sdk.Node{
				Name: "root",
				Context: &sdk.NodeContext{
					PipelineID: pip.ID,
				},
			},
		},
	}

	test.NoError(t, workflow.Insert(context.TODO(), api.mustDB(), api.Cache, &wf, proj))

	vars := map[string]string{
		"key":              proj.Key,
		"permWorkflowName": "workflow1",
	}
	uri := api.Router.GetRoute("GET", api.getWorkflowHandler, vars)
	test.NotEmpty(t, uri)
	req := assets.NewAuthentifiedRequest(t, u, pass, "GET", uri, nil)

	//Do the request
	w := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)

	wfGet := sdk.Workflow{}
	test.NoError(t, json.Unmarshal(w.Body.Bytes(), &wfGet))
	assert.Equal(t, true, wfGet.Permissions.Readable, "readable should be true")
	assert.Equal(t, true, wfGet.Permissions.Writable, "writable should be true")
	assert.Equal(t, true, wfGet.Permissions.Executable, "writable should be true")

	var err error

	userAdmin, passAdmin := assets.InsertAdminUser(t, db)
	uri = api.Router.GetRoute("GET", api.getWorkflowHandler, vars)
	req, err = http.NewRequest("GET", uri, nil)
	test.NoError(t, err)
	assets.AuthentifyRequest(t, req, userAdmin, passAdmin)

	// Do the request
	w = httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)

	wfGet = sdk.Workflow{}
	test.NoError(t, json.Unmarshal(w.Body.Bytes(), &wfGet))
	assert.Equal(t, true, wfGet.Permissions.Readable, "readable should be true")
	assert.Equal(t, true, wfGet.Permissions.Writable, "writable should be true")
	assert.Equal(t, true, wfGet.Permissions.Executable, "executable should be true")

	userMaintainer, passMaintainer := assets.InsertMaintainerUser(t, db)
	uri = api.Router.GetRoute("GET", api.getWorkflowHandler, vars)
	req, err = http.NewRequest("GET", uri, nil)
	test.NoError(t, err)
	assets.AuthentifyRequest(t, req, userMaintainer, passMaintainer)

	// Do the request
	w = httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)

	wfGet = sdk.Workflow{}
	test.NoError(t, json.Unmarshal(w.Body.Bytes(), &wfGet))
	assert.Equal(t, true, wfGet.Permissions.Readable, "readable should be true")
	assert.Equal(t, false, wfGet.Permissions.Writable, "writable should be false")
	assert.Equal(t, false, wfGet.Permissions.Executable, "executable should be false")

}

func Test_getWorkflowHandler_AsProvider(t *testing.T) {
	api, tsURL, tsClose := newTestServer(t)
	defer tsClose()

	admin, _ := assets.InsertAdminUser(t, api.mustDB())
	localConsumer, err := authentication.LoadConsumerByTypeAndUserID(context.TODO(), api.mustDB(), sdk.ConsumerLocal, admin.ID, authentication.LoadConsumerOptions.WithAuthentifiedUser)
	require.NoError(t, err)

	_, jws, err := builtin.NewConsumer(context.TODO(), api.mustDB(), sdk.RandomString(10), sdk.RandomString(10), localConsumer, admin.GetGroupIDs(), Scope(sdk.AuthConsumerScopeProject))

	u, _ := assets.InsertLambdaUser(t, api.mustDB())

	pkey := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, api.mustDB(), api.Cache, pkey, pkey)
	require.NoError(t, group.InsertLinkGroupUser(context.TODO(), api.mustDB(), &group.LinkGroupUser{
		GroupID:            proj.ProjectGroups[0].Group.ID,
		AuthentifiedUserID: u.ID,
		Admin:              true,
	}))

	pip := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip1",
	}

	test.NoError(t, pipeline.InsertPipeline(api.mustDB(), api.Cache, proj, &pip))

	proj, _ = project.LoadByID(api.mustDB(), api.Cache, proj.ID,
		project.LoadOptions.WithApplications,
		project.LoadOptions.WithPipelines,
		project.LoadOptions.WithEnvironments,
		project.LoadOptions.WithGroups,
	)

	wf := sdk.Workflow{
		Name:       "workflow1",
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		WorkflowData: &sdk.WorkflowData{
			Node: sdk.Node{
				Name: "root",
				Context: &sdk.NodeContext{
					PipelineID: pip.ID,
				},
			},
		},
	}

	test.NoError(t, workflow.Insert(context.TODO(), api.mustDB(), api.Cache, &wf, proj))

	sdkclient := cdsclient.NewProviderClient(cdsclient.ProviderConfig{
		Host:  tsURL,
		Token: jws,
	})

	w, err := sdkclient.WorkflowLoad(pkey, wf.Name)
	test.NoError(t, err)
	t.Logf("%+v", w)

	///
}

func Test_getWorkflowHandler_withUsage(t *testing.T) {
	api, db, router, end := newTestAPI(t)
	defer end()

	// Init user
	u, pass := assets.InsertAdminUser(t, api.mustDB())
	// Init project
	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, api.Cache, key, key)
	//Prepare request
	vars := map[string]string{
		"key":              proj.Key,
		"permWorkflowName": "workflow1",
	}
	uri := router.GetRoute("GET", api.getWorkflowHandler, vars)
	test.NotEmpty(t, uri)

	pip := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip1",
	}

	test.NoError(t, pipeline.InsertPipeline(db, api.Cache, proj, &pip))

	proj, _ = project.LoadByID(db, api.Cache, proj.ID,
		project.LoadOptions.WithApplications,
		project.LoadOptions.WithPipelines,
		project.LoadOptions.WithEnvironments,
		project.LoadOptions.WithGroups,
	)

	wf := sdk.Workflow{
		Name:       "workflow1",
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		WorkflowData: &sdk.WorkflowData{
			Node: sdk.Node{
				Name: "root",
				Context: &sdk.NodeContext{
					PipelineID: pip.ID,
				},
			},
		},
	}

	test.NoError(t, workflow.Insert(context.TODO(), db, api.Cache, &wf, proj))

	req := assets.NewAuthentifiedRequest(t, u, pass, "GET", uri+"?withUsage=true", nil)
	//Do the request
	w := httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)

	workflowResp := &sdk.Workflow{}
	test.NoError(t, json.Unmarshal(w.Body.Bytes(), workflowResp))

	assert.NotNil(t, workflowResp.Usage)
	assert.NotNil(t, workflowResp.Usage.Pipelines)
	assert.Equal(t, 1, len(workflowResp.Usage.Pipelines))
	assert.Equal(t, "pip1", workflowResp.Usage.Pipelines[0].Name)
}

func Test_postWorkflowHandlerWithoutRootShouldFail(t *testing.T) {
	api, db, router, end := newTestAPI(t)
	defer end()

	// Init user
	u, pass := assets.InsertAdminUser(t, api.mustDB())
	// Init project
	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, api.Cache, key, key)
	//Prepare request
	vars := map[string]string{
		"permProjectKey": proj.Key,
	}
	uri := router.GetRoute("POST", api.postWorkflowHandler, vars)
	test.NotEmpty(t, uri)

	var workflowResponse sdk.Workflow
	req := assets.NewAuthentifiedRequest(t, u, pass, "POST", uri, &workflowResponse)
	//Do the request
	w := httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)
	assert.Equal(t, 400, w.Code)
}

func Test_postWorkflowHandlerWithRootShouldSuccess(t *testing.T) {

	api, db, router, end := newTestAPI(t)
	defer end()

	// Init user
	u, pass := assets.InsertAdminUser(t, api.mustDB())
	// Init project
	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, api.Cache, key, key)
	// Init pipeline
	pip := sdk.Pipeline{
		Name:      "pipeline1",
		ProjectID: proj.ID,
	}
	test.NoError(t, pipeline.InsertPipeline(api.mustDB(), api.Cache, proj, &pip))

	//Prepare request
	vars := map[string]string{
		"permProjectKey": proj.Key,
	}
	uri := router.GetRoute("POST", api.postWorkflowHandler, vars)
	test.NotEmpty(t, uri)

	// Insert application
	app := sdk.Application{
		Name:               "app1",
		RepositoryFullname: "test/app1",
		VCSServer:          "github",
	}
	test.NoError(t, application.Insert(api.mustDB(), api.Cache, proj, &app))

	var workflow = &sdk.Workflow{
		Name:        "Name",
		Description: "Description",
		WorkflowData: &sdk.WorkflowData{
			Node: sdk.Node{
				Type: sdk.NodeTypePipeline,
				Context: &sdk.NodeContext{
					ApplicationID: app.ID,
					PipelineID:    pip.ID,
				},
			},
		},
	}

	req := assets.NewAuthentifiedRequest(t, u, pass, "POST", uri, &workflow)
	//Do the request
	w := httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)
	assert.Equal(t, 201, w.Code)

	test.NoError(t, json.Unmarshal(w.Body.Bytes(), &workflow))
	assert.NotEqual(t, 0, workflow.ID)

	assert.NotEqual(t, 0, workflow.WorkflowData.Node.Context.ApplicationID)
	assert.NotNil(t, workflow.WorkflowData.Node.Context.DefaultPayload)

	payload, err := workflow.WorkflowData.Node.Context.DefaultPayloadToMap()
	test.NoError(t, err)

	assert.NotEmpty(t, payload["git.branch"], "git.branch should not be empty")
}
func Test_postWorkflowHandlerWithBadPayloadShouldFail(t *testing.T) {

	api, db, router, end := newTestAPI(t)
	defer end()

	// Init user
	u, pass := assets.InsertAdminUser(t, api.mustDB())
	// Init project
	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, api.Cache, key, key)
	// Init pipeline
	pip := sdk.Pipeline{
		Name:      "pipeline1",
		ProjectID: proj.ID,
	}
	test.NoError(t, pipeline.InsertPipeline(api.mustDB(), api.Cache, proj, &pip))

	//Prepare request
	vars := map[string]string{
		"permProjectKey": proj.Key,
	}
	uri := router.GetRoute("POST", api.postWorkflowHandler, vars)
	test.NotEmpty(t, uri)

	// Insert application
	app := sdk.Application{
		Name:               "app1",
		RepositoryFullname: "test/app1",
		VCSServer:          "github",
	}
	test.NoError(t, application.Insert(api.mustDB(), api.Cache, proj, &app))

	var workflow = &sdk.Workflow{
		Name:        "Name",
		Description: "Description",
		WorkflowData: &sdk.WorkflowData{
			Node: sdk.Node{
				Type: sdk.NodeTypePipeline,
				Context: &sdk.NodeContext{
					ApplicationID:  app.ID,
					PipelineID:     pip.ID,
					DefaultPayload: map[string]string{"cds.test": "test"},
				},
			},
		},
	}

	req := assets.NewAuthentifiedRequest(t, u, pass, "POST", uri, &workflow)
	//Do the request
	w := httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)
	assert.Equal(t, 400, w.Code)
}

func Test_putWorkflowHandler(t *testing.T) {

	api, db, router, end := newTestAPI(t)
	defer end()

	// Init user
	u, pass := assets.InsertAdminUser(t, api.mustDB())

	assert.NoError(t, workflow.CreateBuiltinWorkflowHookModels(db))

	repoHookModel, err := workflow.LoadHookModelByName(db, sdk.RepositoryWebHookModel.Name)
	assert.NoError(t, err)

	mockVCSservice, _ := assets.InsertService(t, db, "Test_putWorkflowHandler_TypeVCS", services.TypeVCS)
	defer func() {
		_ = services.Delete(db, mockVCSservice)
	}()
	mockHookservice, _ := assets.InsertService(t, db, "Test_putWorkflowHandler_TypeHooks", services.TypeHooks)
	defer func() {
		_ = services.Delete(db, mockHookservice)
	}()

	updatehookCalled := false

	//This is a mock for the repositories service
	services.HTTPClient = mock(
		func(r *http.Request) (*http.Response, error) {
			body := new(bytes.Buffer)
			w := new(http.Response)
			enc := json.NewEncoder(body)
			w.Body = ioutil.NopCloser(body)

			switch r.URL.String() {
			case "/vcs/github/repos/foo/bar/branches":
				bs := []sdk.VCSBranch{}
				b := sdk.VCSBranch{
					DisplayID: "master",
					Default:   true,
				}
				bs = append(bs, b)
				if err := enc.Encode(bs); err != nil {
					return writeError(w, err)
				}
			case "/task/bulk":
				hooks := map[string]sdk.NodeHook{}
				if err := service.UnmarshalBody(r, &hooks); err != nil {
					return nil, sdk.WithStack(err)
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
			case "/vcs/github/webhooks":
				hookInfo := repositoriesmanager.WebhooksInfos{
					WebhooksSupported: true,
					WebhooksDisabled:  false,
				}
				if err := enc.Encode(hookInfo); err != nil {
					return writeError(w, err)
				}
			case "/vcs/github/repos/foo/bar/hooks":
				hook := sdk.VCSHook{}
				if err := service.UnmarshalBody(r, &hook); err != nil {
					return nil, sdk.WithStack(err)
				}
				hook.ID = "666"
				hook.Events = []string{"push"}
				if err := enc.Encode(hook); err != nil {
					return writeError(w, err)
				}
			case "/vcs/github/repos/foo/bar/hooks?url=http%3A%2F%2Flolcat.host&id=666":
				updatehookCalled = true
				hook := sdk.VCSHook{}
				if err := service.UnmarshalBody(r, &hook); err != nil {
					return nil, sdk.WithStack(err)
				}
				assert.Equal(t, len(hook.Events), 2)
				if err := enc.Encode(hook); err != nil {
					return writeError(w, err)
				}
			default:
				t.Fatalf("unknown route %s", r.URL.String())
			}

			return w, nil
		},
	)

	// Init project
	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, api.Cache, key, key)
	assert.NoError(t, repositoriesmanager.InsertForProject(db, proj, &sdk.ProjectVCSServer{
		Name: "github",
		Data: map[string]string{
			"token":  "foo",
			"secret": "bar",
		},
	}))

	// Init pipeline
	pip := sdk.Pipeline{
		Name:      "pipeline1",
		ProjectID: proj.ID,
	}
	test.NoError(t, pipeline.InsertPipeline(api.mustDB(), api.Cache, proj, &pip))

	// Create application
	app := sdk.Application{
		ProjectID:          proj.ID,
		Name:               sdk.RandomString(10),
		RepositoryFullname: "foo/bar",
		VCSServer:          "github",
	}
	assert.NoError(t, application.Insert(db, api.Cache, proj, &app))
	assert.NoError(t, repositoriesmanager.InsertForApplication(db, &app, proj.Key))

	//Prepare request
	vars := map[string]string{
		"permProjectKey": proj.Key,
	}
	uri := router.GetRoute("POST", api.postWorkflowHandler, vars)
	test.NotEmpty(t, uri)

	var workflow = &sdk.Workflow{
		Name:        "Name",
		Description: "Description",
		WorkflowData: &sdk.WorkflowData{
			Node: sdk.Node{
				Type: sdk.NodeTypePipeline,
				Context: &sdk.NodeContext{
					PipelineID:    pip.ID,
					ApplicationID: app.ID,
				},
				Hooks: []sdk.NodeHook{
					{
						Config:        repoHookModel.DefaultConfig,
						HookModelName: repoHookModel.Name,
						HookModelID:   repoHookModel.ID,
					},
				},
			},
		},
	}

	req := assets.NewAuthentifiedRequest(t, u, pass, "POST", uri, &workflow)

	//Do the request
	w := httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)
	assert.Equal(t, 201, w.Code)
	test.NoError(t, json.Unmarshal(w.Body.Bytes(), &workflow))
	assert.False(t, updatehookCalled)

	//Prepare request
	vars = map[string]string{
		"key":              proj.Key,
		"permWorkflowName": "Name",
	}
	uri = router.GetRoute("PUT", api.putWorkflowHandler, vars)
	test.NotEmpty(t, uri)

	model := sdk.IntegrationModel{
		Name:  sdk.RandomString(10),
		Event: true,
	}
	test.NoError(t, integration.InsertModel(api.mustDB(), &model))

	projInt := sdk.ProjectIntegration{
		Config: sdk.IntegrationConfig{
			"test": sdk.IntegrationConfigValue{
				Description: "here is a test",
				Type:        sdk.IntegrationConfigTypeString,
				Value:       "test",
			},
		},
		Name:               sdk.RandomString(10),
		ProjectID:          proj.ID,
		Model:              model,
		IntegrationModelID: model.ID,
	}
	test.NoError(t, integration.InsertIntegration(db, &projInt))

	var workflow1 = &sdk.Workflow{
		Name:        "Name",
		Description: "Description 2",
		WorkflowData: &sdk.WorkflowData{
			Node: sdk.Node{
				Type: sdk.NodeTypePipeline,
				Context: &sdk.NodeContext{
					PipelineID:    pip.ID,
					ApplicationID: app.ID,
				},
				Hooks: []sdk.NodeHook{
					{
						Config:        repoHookModel.DefaultConfig,
						HookModelName: repoHookModel.Name,
						HookModelID:   repoHookModel.ID,
					},
				},
			},
		},
		EventIntegrations: []sdk.ProjectIntegration{projInt},
	}
	workflow1.WorkflowData.Node.Hooks[0].Config[sdk.HookConfigEventFilter] = sdk.WorkflowNodeHookConfigValue{
		Value:        "push;create",
		Configurable: true,
		Type:         sdk.HookConfigTypeMultiChoice,
	}

	req = assets.NewAuthentifiedRequest(t, u, pass, "PUT", uri, &workflow1)

	//Do the request
	w = httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)

	test.NoError(t, json.Unmarshal(w.Body.Bytes(), &workflow1))

	assert.NotEqual(t, 0, workflow1.ID)
	assert.Equal(t, "Description 2", workflow1.Description)

	assert.NotEqual(t, 0, workflow1.WorkflowData.Node.Context.ApplicationID)
	assert.NotNil(t, workflow1.WorkflowData.Node.Context.DefaultPayload)
	assert.NotNil(t, workflow1.EventIntegrations)

	payload, err := workflow1.WorkflowData.Node.Context.DefaultPayloadToMap()
	test.NoError(t, err)
	assert.True(t, updatehookCalled)

	assert.NotEmpty(t, payload["git.branch"], "git.branch should not be empty")
}

func Test_deleteWorkflowEventIntegrationHandler(t *testing.T) {
	api, db, router, end := newTestAPI(t)
	defer end()

	// Init user
	u, pass := assets.InsertAdminUser(t, api.mustDB())
	// Init project
	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, api.Cache, key, key)
	// Init pipeline
	pip := sdk.Pipeline{
		Name:      "pipeline1",
		ProjectID: proj.ID,
	}
	test.NoError(t, pipeline.InsertPipeline(api.mustDB(), api.Cache, proj, &pip))
	//Prepare request
	vars := map[string]string{
		"permProjectKey": proj.Key,
	}
	uri := router.GetRoute("POST", api.postWorkflowHandler, vars)
	test.NotEmpty(t, uri)

	var wf = &sdk.Workflow{
		Name:        "Name",
		Description: "Description",
		WorkflowData: &sdk.WorkflowData{
			Node: sdk.Node{
				Type: sdk.NodeTypePipeline,
				Context: &sdk.NodeContext{
					PipelineID: pip.ID,
				},
			},
		},
	}

	req := assets.NewAuthentifiedRequest(t, u, pass, "POST", uri, &wf)

	//Do the request
	w := httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)
	assert.Equal(t, 201, w.Code)
	test.NoError(t, json.Unmarshal(w.Body.Bytes(), &wf))

	//Prepare request
	vars = map[string]string{
		"key":              proj.Key,
		"permWorkflowName": "Name",
	}
	uri = router.GetRoute("PUT", api.putWorkflowHandler, vars)
	test.NotEmpty(t, uri)

	// Insert application
	app := sdk.Application{
		Name:               "app1",
		RepositoryFullname: "test/app1",
		VCSServer:          "github",
	}
	test.NoError(t, application.Insert(api.mustDB(), api.Cache, proj, &app))

	model := sdk.IntegrationModel{
		Name:  sdk.RandomString(10),
		Event: true,
	}
	test.NoError(t, integration.InsertModel(api.mustDB(), &model))

	projInt := sdk.ProjectIntegration{
		Config: sdk.IntegrationConfig{
			"test": sdk.IntegrationConfigValue{
				Description: "here is a test",
				Type:        sdk.IntegrationConfigTypeString,
				Value:       "test",
			},
		},
		Name:               sdk.RandomString(10),
		ProjectID:          proj.ID,
		Model:              model,
		IntegrationModelID: model.ID,
	}
	test.NoError(t, integration.InsertIntegration(db, &projInt))

	var workflow1 = &sdk.Workflow{
		Name:        "Name",
		Description: "Description 2",
		WorkflowData: &sdk.WorkflowData{
			Node: sdk.Node{
				Type: sdk.NodeTypePipeline,
				Context: &sdk.NodeContext{
					PipelineID:    pip.ID,
					ApplicationID: app.ID,
				},
			},
		},
		EventIntegrations: []sdk.ProjectIntegration{projInt},
	}

	req = assets.NewAuthentifiedRequest(t, u, pass, "PUT", uri, &workflow1)

	//Do the request
	w = httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)

	test.NoError(t, json.Unmarshal(w.Body.Bytes(), &workflow1))

	assert.NotEqual(t, 0, workflow1.ID)
	assert.Equal(t, "Description 2", workflow1.Description)

	assert.NotEqual(t, 0, workflow1.WorkflowData.Node.Context.ApplicationID)
	assert.NotNil(t, workflow1.WorkflowData.Node.Context.DefaultPayload)
	assert.NotNil(t, workflow1.EventIntegrations)
	assert.Equal(t, len(workflow1.EventIntegrations), 1)

	vars["integrationID"] = fmt.Sprintf("%d", projInt.ID)
	uri = router.GetRoute("DELETE", api.deleteWorkflowEventsIntegrationHandler, vars)
	req = assets.NewAuthentifiedRequest(t, u, pass, "DELETE", uri, nil)

	//Do the request
	w = httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)

	wfUpdated, err := workflow.Load(context.TODO(), api.mustDB(), api.Cache, proj, wf.Name, workflow.LoadOptions{WithIntegrations: true})
	test.NoError(t, err, "cannot load workflow")

	test.Equal(t, 0, len(wfUpdated.EventIntegrations))
}

func Test_postWorkflowHandlerWithError(t *testing.T) {
	t.SkipNow()

	// This call on postWorkflowHandler should raise an error
	// because default payload on non-root node should be illegal
	// issue #4593

	api, db, router, end := newTestAPI(t)
	defer end()

	// Init user
	u, pass := assets.InsertAdminUser(t, api.mustDB())
	// Init project
	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, api.Cache, key, key)
	// Init pipeline
	pip := sdk.Pipeline{
		Name:      "pipeline1",
		ProjectID: proj.ID,
	}

	require.NoError(t, pipeline.InsertPipeline(api.mustDB(), api.Cache, proj, &pip))

	//Prepare request
	vars := map[string]string{
		"permProjectKey": proj.Key,
	}
	uri := router.GetRoute("POST", api.postWorkflowHandler, vars)
	test.NotEmpty(t, uri)

	var workflow = &sdk.Workflow{
		Name:        "Name",
		Description: "Description",
		WorkflowData: &sdk.WorkflowData{
			Node: sdk.Node{
				Type: sdk.NodeTypePipeline,
				Context: &sdk.NodeContext{
					PipelineID: pip.ID,
				},
				Triggers: []sdk.NodeTrigger{{
					ChildNode: sdk.Node{
						Type: sdk.NodeTypePipeline,
						Context: &sdk.NodeContext{
							PipelineID: pip.ID,
							DefaultPayload: map[string]interface{}{
								"test": "content",
							},
						},
					},
				}},
			},
		},
	}

	req := assets.NewAuthentifiedRequest(t, u, pass, "POST", uri, &workflow)

	//Do the request
	w := httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)
	assert.Equal(t, 400, w.Code)
}

func Test_postWorkflowRollbackHandler(t *testing.T) {

	api, db, router, end := newTestAPI(t)
	defer end()

	// Init user
	u, pass := assets.InsertAdminUser(t, api.mustDB())
	// Init project
	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, api.Cache, key, key)

	// Init pipeline
	pip := sdk.Pipeline{
		Name:      "pipeline1",
		ProjectID: proj.ID,
	}
	test.NoError(t, pipeline.InsertPipeline(api.mustDB(), api.Cache, proj, &pip))

	// Create WORKFLOW NAME

	//Prepare request
	vars := map[string]string{
		"permProjectKey": proj.Key,
	}
	uri := router.GetRoute("POST", api.postWorkflowHandler, vars)
	test.NotEmpty(t, uri)

	var wf = &sdk.Workflow{
		Name:        "Name",
		Description: "Description",
		WorkflowData: &sdk.WorkflowData{
			Node: sdk.Node{
				Type: sdk.NodeTypePipeline,
				Context: &sdk.NodeContext{
					PipelineID: pip.ID,
				},
			},
		},
	}

	req := assets.NewAuthentifiedRequest(t, u, pass, "POST", uri, &wf)

	//Do the request
	w := httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)
	assert.Equal(t, 201, w.Code)
	test.NoError(t, json.Unmarshal(w.Body.Bytes(), &wf))

	// UPDATE WORKFLOW : add APPLICATION ON ROOT NODE

	//Prepare request
	vars = map[string]string{
		"key":              proj.Key,
		"permWorkflowName": "Name",
	}
	uri = router.GetRoute("PUT", api.putWorkflowHandler, vars)
	test.NotEmpty(t, uri)

	// Insert application
	app := sdk.Application{
		Name:               "app1",
		RepositoryFullname: "test/app1",
		VCSServer:          "github",
	}
	test.NoError(t, application.Insert(api.mustDB(), api.Cache, proj, &app))

	var workflow1 = &sdk.Workflow{
		ID:          wf.ID,
		Name:        "Name",
		Description: "Description 2",
		WorkflowData: &sdk.WorkflowData{
			Node: sdk.Node{
				Type: sdk.NodeTypePipeline,
				Context: &sdk.NodeContext{
					ApplicationID: app.ID,
					PipelineID:    pip.ID,
				},
			},
		},
	}

	req = assets.NewAuthentifiedRequest(t, u, pass, "PUT", uri, &workflow1)
	//Do the request
	w = httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)

	test.NoError(t, json.Unmarshal(w.Body.Bytes(), &workflow1))

	assert.NotEqual(t, 0, workflow1.ID)
	assert.Equal(t, "Description 2", workflow1.Description)

	assert.NotEqual(t, 0, workflow1.WorkflowData.Node.Context.ApplicationID)
	assert.NotNil(t, workflow1.WorkflowData.Node.Context.DefaultPayload)

	payload, err := workflow1.WorkflowData.Node.Context.DefaultPayloadToMap()
	test.NoError(t, err)

	assert.NotEmpty(t, payload["git.branch"], "git.branch should not be empty")

	test.NoError(t, workflow.IsValid(context.Background(), api.Cache, db, wf, proj, workflow.LoadOptions{}))
	eWf, err := exportentities.NewWorkflow(context.TODO(), *wf)
	test.NoError(t, err)
	wfBts, err := yaml.Marshal(eWf)
	test.NoError(t, err)
	eWfUpdate, err := exportentities.NewWorkflow(context.TODO(), *workflow1)
	test.NoError(t, err)
	wfUpdatedBts, err := yaml.Marshal(eWfUpdate)
	test.NoError(t, err)

	// INSERT AUDIT

	wfAudit := sdk.AuditWorkflow{
		AuditCommon: sdk.AuditCommon{
			Created:     time.Now(),
			EventType:   "WorkflowUpdate",
			TriggeredBy: u.Username,
		},
		ProjectKey: proj.Key,
		WorkflowID: wf.ID,
		DataType:   "yaml",
		DataBefore: string(wfBts),
		DataAfter:  string(wfUpdatedBts),
	}
	test.NoError(t, workflow.InsertAudit(api.mustDB(), &wfAudit))

	// ROLLBACK TO PREVIOUS WORKFLOW

	//Prepare request
	vars = map[string]string{
		"key":              proj.Key,
		"permWorkflowName": "Name",
		"auditID":          fmt.Sprintf("%d", wfAudit.ID),
	}
	uri = router.GetRoute("POST", api.postWorkflowRollbackHandler, vars)
	test.NotEmpty(t, uri)
	req = assets.NewAuthentifiedRequest(t, u, pass, "POST", uri, nil)

	//Do the request
	w = httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)

	var wfRollback sdk.Workflow
	test.NoError(t, json.Unmarshal(w.Body.Bytes(), &wfRollback))

	if wfRollback.WorkflowData == nil {
		t.Fatal(fmt.Errorf("workflow not found"))
	}

	test.Equal(t, int64(0), wfRollback.WorkflowData.Node.Context.ApplicationID)

	assert.Equal(t, true, wfRollback.Permissions.Readable)
	assert.Equal(t, true, wfRollback.Permissions.Executable)
	assert.Equal(t, true, wfRollback.Permissions.Writable)
}

func Test_postAndDeleteWorkflowLabelHandler(t *testing.T) {

	api, db, router, end := newTestAPI(t)
	defer end()

	// Init user
	u, pass := assets.InsertAdminUser(t, api.mustDB())
	// Init project
	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, api.Cache, key, key)

	lbl1 := sdk.Label{
		Name:      sdk.RandomString(5),
		ProjectID: proj.ID,
	}
	test.NoError(t, project.InsertLabel(db, &lbl1))

	// Init pipeline
	pip := sdk.Pipeline{
		Name:      "pipeline1",
		ProjectID: proj.ID,
	}
	test.NoError(t, pipeline.InsertPipeline(api.mustDB(), api.Cache, proj, &pip))

	integrationModel, err := integration.LoadModelByName(db, sdk.KafkaIntegration.Name, false)
	if err != nil {
		assert.NoError(t, integration.CreateBuiltinModels(db))
		models, _ := integration.LoadModels(db)
		assert.True(t, len(models) > 0)
	}

	integrationModel, err = integration.LoadModelByName(db, sdk.KafkaIntegration.Name, false)
	test.NoError(t, err)

	pname := sdk.RandomString(10)
	pp := sdk.ProjectIntegration{
		Name:               pname,
		Config:             sdk.KafkaIntegration.DefaultConfig.Clone(),
		IntegrationModelID: integrationModel.ID,
	}

	// ADD integration
	vars := map[string]string{}
	vars[permProjectKey] = proj.Key
	uri := router.GetRoute("POST", api.postProjectIntegrationHandler, vars)
	req := assets.NewAuthentifiedRequest(t, u, pass, "POST", uri, pp)
	w := httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)

	pi := sdk.ProjectIntegration{}
	test.NoError(t, json.Unmarshal(w.Body.Bytes(), &pi))
	assert.Equal(t, pname, pi.Name)

	proj, err = project.Load(api.mustDB(), api.Cache, proj.Key,
		project.LoadOptions.WithApplicationWithDeploymentStrategies,
		project.LoadOptions.WithPipelines,
		project.LoadOptions.WithEnvironments,
		project.LoadOptions.WithGroups,
		project.LoadOptions.WithIntegrations,
	)

	test.NoError(t, err)

	vars = map[string]string{
		"permProjectKey": proj.Key,
	}
	uri = router.GetRoute("POST", api.postWorkflowHandler, vars)
	test.NotEmpty(t, uri)

	name := sdk.RandomString(10)
	var wf = &sdk.Workflow{
		Name:        name,
		Description: "Description",
		WorkflowData: &sdk.WorkflowData{
			Node: sdk.Node{
				Type: sdk.NodeTypePipeline,
				Context: &sdk.NodeContext{
					PipelineID:           pip.ID,
					ProjectIntegrationID: pi.ID,
				},
			},
		},
	}

	req = assets.NewAuthentifiedRequest(t, u, pass, "POST", uri, &wf)

	//Do the request
	w = httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)
	assert.Equal(t, 201, w.Code)
	test.NoError(t, json.Unmarshal(w.Body.Bytes(), &wf))

	//Prepare request
	vars = map[string]string{
		"key":              proj.Key,
		"permWorkflowName": name,
	}
	uri = router.GetRoute("POST", api.postWorkflowLabelHandler, vars)
	test.NotEmpty(t, uri)

	req = assets.NewAuthentifiedRequest(t, u, pass, "POST", uri, &lbl1)
	//Do the request
	w = httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)

	test.NoError(t, json.Unmarshal(w.Body.Bytes(), &lbl1))

	assert.NotEqual(t, 0, lbl1.ID)
	assert.Equal(t, proj.ID, lbl1.ProjectID)
	assert.Equal(t, wf.ID, lbl1.WorkflowID)

	wfUpdated, errW := workflow.Load(context.TODO(), db, api.Cache, proj, wf.Name, workflow.LoadOptions{WithLabels: true})
	test.NoError(t, errW)

	assert.NotNil(t, wfUpdated.Labels)
	assert.Equal(t, 1, len(wfUpdated.Labels))
	assert.Equal(t, lbl1.Name, wfUpdated.Labels[0].Name)

	//Unlink label
	vars = map[string]string{
		"key":              proj.Key,
		"permWorkflowName": name,
		"labelID":          fmt.Sprintf("%d", lbl1.ID),
	}
	uri = router.GetRoute("DELETE", api.deleteWorkflowLabelHandler, vars)
	test.NotEmpty(t, uri)

	req = assets.NewAuthentifiedRequest(t, u, pass, "DELETE", uri, nil)
	//Do the request
	w = httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)

	wfUpdated, errW = workflow.Load(context.TODO(), db, api.Cache, proj, wf.Name, workflow.LoadOptions{WithLabels: true})
	test.NoError(t, errW)
	assert.NotNil(t, wfUpdated.Labels)
	assert.Equal(t, 0, len(wfUpdated.Labels))
}

func Test_deleteWorkflowHandler(t *testing.T) {

	api, db, router, end := newTestAPI(t)
	defer end()
	test.NoError(t, workflow.CreateBuiltinWorkflowHookModels(db))

	// Init user
	u, pass := assets.InsertAdminUser(t, api.mustDB())
	// Init project
	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, api.Cache, key, key)
	// Init pipeline
	pip := sdk.Pipeline{
		Name:      "pipeline1",
		ProjectID: proj.ID,
	}
	test.NoError(t, pipeline.InsertPipeline(api.mustDB(), api.Cache, proj, &pip))

	//Prepare request
	vars := map[string]string{
		"permProjectKey": proj.Key,
	}
	uri := router.GetRoute("POST", api.postWorkflowHandler, vars)
	test.NotEmpty(t, uri)

	var wkf = &sdk.Workflow{
		Name:        "Name",
		Description: "Description",
		WorkflowData: &sdk.WorkflowData{
			Node: sdk.Node{
				Type: sdk.NodeTypePipeline,
				Context: &sdk.NodeContext{
					PipelineID: pip.ID,
				},
			},
		},
	}

	req := assets.NewAuthentifiedRequest(t, u, pass, "POST", uri, &wkf)

	//Do the request
	w := httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)
	assert.Equal(t, 201, w.Code)
	test.NoError(t, json.Unmarshal(w.Body.Bytes(), &wkf))

	vars = map[string]string{
		"key":              proj.Key,
		"permWorkflowName": "Name",
	}
	uri = router.GetRoute("DELETE", api.deleteWorkflowHandler, vars)
	test.NotEmpty(t, uri)

	req = assets.NewAuthentifiedRequest(t, u, pass, "DELETE", uri, nil)

	//Do the request
	w = httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)

	// Waiting until the deletion is over
	ctx, cancel := context.WithTimeout(context.TODO(), 1*time.Minute)
	defer cancel()

	tickCheck := time.NewTicker(1 * time.Second)
	defer tickCheck.Stop()

loop:
	for {
		select {
		case <-ctx.Done():
			t.Fatal(ctx.Err())
		case <-tickCheck.C:
			wk, _ := workflow.Load(ctx, db, api.Cache, proj, wkf.Name, workflow.LoadOptions{Minimal: true})
			if wk == nil {
				break loop
			}
		}
	}

}

func TestBenchmarkGetWorkflowsWithoutAPIAsAdmin(t *testing.T) {
	t.SkipNow()

	db, cache, end := test.SetupPG(t)
	defer end()

	// Init project
	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, cache, key, key)
	// Init pipeline
	pip := sdk.Pipeline{
		Name:      "pipeline1",
		ProjectID: proj.ID,
	}

	assert.NoError(t, pipeline.InsertPipeline(db, cache, proj, &pip))

	app := sdk.Application{
		Name: sdk.RandomString(10),
	}

	assert.NoError(t, application.Insert(db, cache, proj, &app))

	prj, err := project.Load(db, cache, proj.Key,
		project.LoadOptions.WithPipelines,
		project.LoadOptions.WithApplications,
		project.LoadOptions.WithWorkflows,
	)
	assert.NoError(t, err)

	for i := 0; i < 300; i++ {
		wf := sdk.Workflow{
			ProjectID:  proj.ID,
			ProjectKey: proj.Key,
			Name:       sdk.RandomString(10),
			WorkflowData: &sdk.WorkflowData{
				Node: sdk.Node{
					Name: "root",
					Context: &sdk.NodeContext{
						PipelineID:    pip.ID,
						ApplicationID: app.ID,
					},
				},
			},
		}

		assert.NoError(t, workflow.Insert(context.TODO(), db, cache, &wf, prj))
	}

	res := testing.Benchmark(func(b *testing.B) {
		b.ResetTimer()
		for n := 0; n < b.N; n++ {
			if _, err := workflow.LoadAll(db, prj.Key); err != nil {
				b.Logf("Cannot load workflows : %v", err)
				b.Fail()
				return
			}
		}
		b.StopTimer()
	})

	t.Logf("N : %d", res.N)
	t.Logf("ns/op : %d", res.NsPerOp())
	assert.False(t, res.NsPerOp() >= 500000000, "Workflows load is too long: GOT %d and EXPECTED lower than 500000000 (500ms)", res.NsPerOp())
}

func TestBenchmarkGetWorkflowsWithAPI(t *testing.T) {
	t.SkipNow()
	api, db, router, end := newTestAPI(t)
	defer end()

	// Init project
	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, api.Cache, key, key)

	// Init user
	u, pass := assets.InsertLambdaUser(t, db, &proj.ProjectGroups[0].Group)

	// Init pipeline
	pip := sdk.Pipeline{
		Name:      "pipeline1",
		ProjectID: proj.ID,
	}
	assert.NoError(t, pipeline.InsertPipeline(db, api.Cache, proj, &pip))

	app := sdk.Application{
		Name: sdk.RandomString(10),
	}

	assert.NoError(t, application.Insert(db, api.Cache, proj, &app))

	prj, err := project.Load(db, api.Cache, proj.Key,
		project.LoadOptions.WithPipelines,
		project.LoadOptions.WithApplications,
		project.LoadOptions.WithWorkflows,
	)
	assert.NoError(t, err)

	for i := 0; i < 300; i++ {
		wf := sdk.Workflow{
			ProjectID:  proj.ID,
			ProjectKey: proj.Key,
			Name:       sdk.RandomString(10),
			Groups:     proj.ProjectGroups,
			WorkflowData: &sdk.WorkflowData{
				Node: sdk.Node{
					Name: "root",
					Context: &sdk.NodeContext{
						PipelineID:    pip.ID,
						ApplicationID: app.ID,
					},
				},
			},
		}

		assert.NoError(t, workflow.Insert(context.TODO(), db, api.Cache, &wf, prj))
	}

	//Prepare request
	vars := map[string]string{
		"permProjectKey": proj.Key,
	}
	uri := router.GetRoute("GET", api.getWorkflowsHandler, vars)
	test.NotEmpty(t, uri)

	res := testing.Benchmark(func(b *testing.B) {
		b.ResetTimer()

		for n := 0; n < b.N; n++ {
			b.StopTimer()
			req := assets.NewAuthentifiedRequest(t, u, pass, "GET", uri, vars)
			b.StartTimer()
			//Do the request
			w := httptest.NewRecorder()
			router.Mux.ServeHTTP(w, req)
			assert.Equal(t, 200, w.Code)
			b.StopTimer()
			workflows := []sdk.Workflow{}
			json.Unmarshal(w.Body.Bytes(), &workflows)
			test.Equal(t, 300, len(workflows))
		}
		b.StopTimer()
	})

	t.Logf("N : %d", res.N)
	t.Logf("ns/op : %d", res.NsPerOp())
	assert.False(t, res.NsPerOp() >= 500000000, "Workflows load is too long: GOT %d and EXPECTED lower than 500000000 (500ms)", res.NsPerOp())
}
