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
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/authentication"
	"github.com/ovh/cds/engine/api/authentication/builtin"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/integration"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/repositoriesmanager"
	"github.com/ovh/cds/engine/api/services"
	"github.com/ovh/cds/engine/api/services/mock_services"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient"
	"github.com/ovh/cds/sdk/exportentities"
)

func Test_getWorkflowsHandler(t *testing.T) {
	api, db, _ := newTestAPI(t)

	u, pass := assets.InsertLambdaUser(t, db)
	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))
	require.NoError(t, group.InsertLinkGroupUser(context.TODO(), db, &group.LinkGroupUser{
		GroupID:            proj.ProjectGroups[0].Group.ID,
		AuthentifiedUserID: u.ID,
		Admin:              true,
	}))

	pip := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip1",
	}

	require.NoError(t, pipeline.InsertPipeline(api.mustDB(), &pip))

	wf := sdk.Workflow{
		Name:       "workflow1",
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		WorkflowData: sdk.WorkflowData{
			Node: sdk.Node{
				Name: "root",
				Context: &sdk.NodeContext{
					PipelineID: pip.ID,
				},
			},
		},
	}

	require.NoError(t, workflow.Insert(context.TODO(), db, api.Cache, *proj, &wf))

	vars := map[string]string{
		"permProjectKey": proj.Key,
	}
	uri := api.Router.GetRoute("GET", api.getWorkflowsHandler, vars)
	require.NotEmpty(t, uri)
	req := assets.NewAuthentifiedRequest(t, u, pass, "GET", uri, nil)

	//Do the request
	w := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)

	wfList := []sdk.Workflow{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &wfList))
	require.Len(t, wfList, 1)
	for _, w := range wfList {
		assert.Equal(t, true, w.Permissions.Readable, "readable should be true")
		assert.Equal(t, true, w.Permissions.Writable, "writable should be true")
		assert.Equal(t, true, w.Permissions.Executable, "writable should be true")
	}

	var err error

	userAdmin, passAdmin := assets.InsertAdminUser(t, db)
	uri = api.Router.GetRoute("GET", api.getWorkflowsHandler, vars)
	req, err = http.NewRequest("GET", uri, nil)
	require.NoError(t, err)
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
	api, db, router := newTestAPI(t)

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
	test.NoError(t, pipeline.InsertPipeline(db, &pip))

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
	test.NoError(t, pipeline.InsertPipeline(db, &pip2))
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
		WorkflowData: sdk.WorkflowData{
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

	proj2, errP := project.Load(context.TODO(), api.mustDB(), proj.Key, project.LoadOptions.WithPipelines, project.LoadOptions.WithGroups, project.LoadOptions.WithIntegrations)
	test.NoError(t, errP)

	test.NoError(t, workflow.Insert(context.TODO(), db, api.Cache, *proj2, &w))
	w1, err := workflow.Load(context.TODO(), db, api.Cache, *proj, "test_1", workflow.LoadOptions{})
	test.NoError(t, err)

	wrCreate, err := workflow.CreateRun(api.mustDB(), w1, sdk.WorkflowRunPostHandlerOption{AuthConsumerID: consumer.ID})
	assert.NoError(t, err)
	wrCreate.Workflow = *w1
	_, errMR := workflow.StartWorkflowRun(context.TODO(), db, api.Cache, *proj, wrCreate, &sdk.WorkflowRunPostHandlerOption{
		Manual: &sdk.WorkflowNodeRunManual{Username: u.GetUsername()},
	}, *consumer, nil)
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
	api, db, router := newTestAPI(t)

	// Init user
	u, pass := assets.InsertAdminUser(t, db)
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
	api, db, _ := newTestAPI(t)

	u, pass := assets.InsertLambdaUser(t, db)
	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))
	require.NoError(t, group.InsertLinkGroupUser(context.TODO(), db, &group.LinkGroupUser{
		GroupID:            proj.ProjectGroups[0].Group.ID,
		AuthentifiedUserID: u.ID,
		Admin:              true,
	}))

	pip := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip1",
	}

	test.NoError(t, pipeline.InsertPipeline(api.mustDB(), &pip))

	wf := sdk.Workflow{
		Name:       "workflow1",
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		WorkflowData: sdk.WorkflowData{
			Node: sdk.Node{
				Name: "root",
				Context: &sdk.NodeContext{
					PipelineID: pip.ID,
				},
			},
		},
	}

	require.NoError(t, workflow.Insert(context.TODO(), db, api.Cache, *proj, &wf))

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
	uri = api.Router.GetRoute(http.MethodGet, api.getWorkflowHandler, vars)
	req, err = http.NewRequest(http.MethodGet, uri, nil)
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
	api, db, tsURL := newTestServer(t)

	admin, _ := assets.InsertAdminUser(t, db)
	localConsumer, err := authentication.LoadConsumerByTypeAndUserID(context.TODO(), api.mustDB(), sdk.ConsumerLocal, admin.ID, authentication.LoadConsumerOptions.WithAuthentifiedUser)
	require.NoError(t, err)

	_, jws, err := builtin.NewConsumer(context.TODO(), db, sdk.RandomString(10), sdk.RandomString(10), localConsumer, admin.GetGroupIDs(),
		sdk.NewAuthConsumerScopeDetails(sdk.AuthConsumerScopeProject))

	u, _ := assets.InsertLambdaUser(t, db)

	pkey := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, api.Cache, pkey, pkey)
	require.NoError(t, group.InsertLinkGroupUser(context.TODO(), db, &group.LinkGroupUser{
		GroupID:            proj.ProjectGroups[0].Group.ID,
		AuthentifiedUserID: u.ID,
		Admin:              true,
	}))

	pip := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip1",
	}

	test.NoError(t, pipeline.InsertPipeline(api.mustDB(), &pip))

	proj, _ = project.LoadByID(api.mustDB(), proj.ID,
		project.LoadOptions.WithApplications,
		project.LoadOptions.WithPipelines,
		project.LoadOptions.WithEnvironments,
		project.LoadOptions.WithGroups,
	)

	wf := sdk.Workflow{
		Name:       "workflow1",
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		WorkflowData: sdk.WorkflowData{
			Node: sdk.Node{
				Name: "root",
				Context: &sdk.NodeContext{
					PipelineID: pip.ID,
				},
			},
		},
	}

	require.NoError(t, workflow.Insert(context.TODO(), db, api.Cache, *proj, &wf))

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
	api, db, router := newTestAPI(t)

	// Init user
	u, pass := assets.InsertAdminUser(t, db)
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

	test.NoError(t, pipeline.InsertPipeline(db, &pip))

	proj, _ = project.LoadByID(db, proj.ID,
		project.LoadOptions.WithApplications,
		project.LoadOptions.WithPipelines,
		project.LoadOptions.WithEnvironments,
		project.LoadOptions.WithGroups,
	)

	wf := sdk.Workflow{
		Name:       "workflow1",
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		WorkflowData: sdk.WorkflowData{
			Node: sdk.Node{
				Name: "root",
				Context: &sdk.NodeContext{
					PipelineID: pip.ID,
				},
			},
		},
	}

	test.NoError(t, workflow.Insert(context.TODO(), db, api.Cache, *proj, &wf))

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
	api, db, router := newTestAPI(t)

	// Init user
	u, pass := assets.InsertAdminUser(t, db)
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

	api, db, router := newTestAPI(t)

	// Init user
	u, pass := assets.InsertAdminUser(t, db)
	// Init project
	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, api.Cache, key, key)
	// Init pipeline
	pip := sdk.Pipeline{
		Name:      "pipeline1",
		ProjectID: proj.ID,
	}
	test.NoError(t, pipeline.InsertPipeline(api.mustDB(), &pip))

	//Prepare request
	vars := map[string]string{
		"permProjectKey": proj.Key,
	}
	uri := router.GetRoute("POST", api.postWorkflowHandler, vars)
	require.NotEmpty(t, uri)

	// Insert application
	app := sdk.Application{
		Name:               "app1",
		RepositoryFullname: "test/app1",
		VCSServer:          "github",
	}
	require.NoError(t, application.Insert(db, proj.ID, &app))

	var workflow = &sdk.Workflow{
		Name:        "Name",
		Description: "Description",
		WorkflowData: sdk.WorkflowData{
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

	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &workflow))
	assert.NotEqual(t, 0, workflow.ID)

	assert.NotEqual(t, 0, workflow.WorkflowData.Node.Context.ApplicationID)
	assert.NotNil(t, workflow.WorkflowData.Node.Context.DefaultPayload)

	payload, err := workflow.WorkflowData.Node.Context.DefaultPayloadToMap()
	require.NoError(t, err)

	assert.NotEmpty(t, payload["git.branch"], "git.branch should not be empty")
}
func Test_postWorkflowHandlerWithBadPayloadShouldFail(t *testing.T) {
	api, db, router := newTestAPI(t)

	// Init user
	u, pass := assets.InsertAdminUser(t, db)
	// Init project
	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, api.Cache, key, key)
	// Init pipeline
	pip := sdk.Pipeline{
		Name:      "pipeline1",
		ProjectID: proj.ID,
	}
	test.NoError(t, pipeline.InsertPipeline(api.mustDB(), &pip))

	//Prepare request
	vars := map[string]string{
		"permProjectKey": proj.Key,
	}
	uri := router.GetRoute("POST", api.postWorkflowHandler, vars)
	require.NotEmpty(t, uri)

	// Insert application
	app := sdk.Application{
		Name:               "app1",
		RepositoryFullname: "test/app1",
		VCSServer:          "github",
	}
	require.NoError(t, application.Insert(db, proj.ID, &app))

	var workflow = &sdk.Workflow{
		Name:        "Name",
		Description: "Description",
		WorkflowData: sdk.WorkflowData{
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
	api, db, router := newTestAPI(t)

	// Init user
	u, pass := assets.InsertAdminUser(t, db)

	require.NoError(t, workflow.CreateBuiltinWorkflowHookModels(api.mustDB()))

	repoHookModel, err := workflow.LoadHookModelByName(db, sdk.RepositoryWebHookModel.Name)
	assert.NoError(t, err)

	mockVCSservice, _ := assets.InsertService(t, db, "Test_putWorkflowHandler_TypeVCS", sdk.TypeVCS)
	defer func() {
		_ = services.Delete(db, mockVCSservice)
	}()
	mockHookservice, _ := assets.InsertService(t, db, "Test_putWorkflowHandler_TypeHooks", sdk.TypeHooks)
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
	vcsServer := sdk.ProjectVCSServerLink{
		ProjectID: proj.ID,
		Name:      "github",
	}
	vcsServer.Set("token", "foo")
	vcsServer.Set("secret", "bar")
	assert.NoError(t, repositoriesmanager.InsertProjectVCSServerLink(context.TODO(), db, &vcsServer))

	// Init pipeline
	pip := sdk.Pipeline{
		Name:      "pipeline1",
		ProjectID: proj.ID,
	}
	test.NoError(t, pipeline.InsertPipeline(api.mustDB(), &pip))

	// Create application
	app := sdk.Application{
		ProjectID:          proj.ID,
		Name:               sdk.RandomString(10),
		RepositoryFullname: "foo/bar",
		VCSServer:          "github",
	}
	require.NoError(t, application.Insert(db, proj.ID, &app))
	require.NoError(t, repositoriesmanager.InsertForApplication(db, &app))

	//Prepare request
	vars := map[string]string{
		"permProjectKey": proj.Key,
	}
	uri := router.GetRoute("POST", api.postWorkflowHandler, vars)
	test.NotEmpty(t, uri)

	var workflow = &sdk.Workflow{
		Name:        "Name",
		Description: "Description",
		WorkflowData: sdk.WorkflowData{
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
	require.NoError(t, integration.InsertModel(db, &model))

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
		WorkflowData: sdk.WorkflowData{
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
	api, db, router := newTestAPI(t)

	// Init user
	u, pass := assets.InsertAdminUser(t, db)
	// Init project
	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, api.Cache, key, key)
	// Init pipeline
	pip := sdk.Pipeline{
		Name:      "pipeline1",
		ProjectID: proj.ID,
	}
	test.NoError(t, pipeline.InsertPipeline(api.mustDB(), &pip))
	//Prepare request
	vars := map[string]string{
		"permProjectKey": proj.Key,
	}
	uri := router.GetRoute("POST", api.postWorkflowHandler, vars)
	test.NotEmpty(t, uri)

	var wf = &sdk.Workflow{
		Name:        "Name",
		Description: "Description",
		WorkflowData: sdk.WorkflowData{
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
	require.NotEmpty(t, uri)

	// Insert application
	app := sdk.Application{
		Name:               "app1",
		RepositoryFullname: "test/app1",
		VCSServer:          "github",
	}
	require.NoError(t, application.Insert(db, proj.ID, &app))

	model := sdk.IntegrationModel{
		Name:  sdk.RandomString(10),
		Event: true,
	}
	require.NoError(t, integration.InsertModel(db, &model))

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
		WorkflowData: sdk.WorkflowData{
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

	wfUpdated, err := workflow.Load(context.TODO(), api.mustDB(), api.Cache, *proj, wf.Name, workflow.LoadOptions{WithIntegrations: true})
	test.NoError(t, err, "cannot load workflow")

	test.Equal(t, 0, len(wfUpdated.EventIntegrations))
}

func Test_postWorkflowHandlerWithError(t *testing.T) {
	t.SkipNow()

	// This call on postWorkflowHandler should raise an error
	// because default payload on non-root node should be illegal
	// issue #4593

	api, db, router := newTestAPI(t)

	// Init user
	u, pass := assets.InsertAdminUser(t, db)
	// Init project
	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, api.Cache, key, key)
	// Init pipeline
	pip := sdk.Pipeline{
		Name:      "pipeline1",
		ProjectID: proj.ID,
	}

	require.NoError(t, pipeline.InsertPipeline(api.mustDB(), &pip))

	//Prepare request
	vars := map[string]string{
		"permProjectKey": proj.Key,
	}
	uri := router.GetRoute("POST", api.postWorkflowHandler, vars)
	test.NotEmpty(t, uri)

	var workflow = &sdk.Workflow{
		Name:        "Name",
		Description: "Description",
		WorkflowData: sdk.WorkflowData{
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

	api, db, router := newTestAPI(t)

	// Init user
	u, pass := assets.InsertAdminUser(t, db)
	// Init project
	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, api.Cache, key, key)

	// Init pipeline
	pip := sdk.Pipeline{
		Name:      "pipeline1",
		ProjectID: proj.ID,
	}
	test.NoError(t, pipeline.InsertPipeline(api.mustDB(), &pip))

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
		WorkflowData: sdk.WorkflowData{
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
	require.NoError(t, application.Insert(db, proj.ID, &app))

	var workflow1 = &sdk.Workflow{
		ID:          wf.ID,
		Name:        "Name",
		Description: "Description 2",
		WorkflowData: sdk.WorkflowData{
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

	test.NoError(t, workflow.CompleteWorkflow(context.Background(), db, wf, *proj, workflow.LoadOptions{}))
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

	test.Equal(t, int64(0), wfRollback.WorkflowData.Node.Context.ApplicationID)

	assert.Equal(t, true, wfRollback.Permissions.Readable)
	assert.Equal(t, true, wfRollback.Permissions.Executable)
	assert.Equal(t, true, wfRollback.Permissions.Writable)
}

func Test_postAndDeleteWorkflowLabelHandler(t *testing.T) {
	api, db, router := newTestAPI(t)

	u, jwt := assets.InsertAdminUser(t, db)

	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, api.Cache, key, key)

	pip := sdk.Pipeline{
		Name:      "pipeline1",
		ProjectID: proj.ID,
	}
	require.NoError(t, pipeline.InsertPipeline(api.mustDB(), &pip))

	name := sdk.RandomString(10)
	var wf = &sdk.Workflow{
		Name:        name,
		Description: "Description",
		WorkflowData: sdk.WorkflowData{
			Node: sdk.Node{
				Type: sdk.NodeTypePipeline,
				Context: &sdk.NodeContext{
					PipelineID: pip.ID,
				},
			},
		},
	}
	uri := router.GetRoute("POST", api.postWorkflowHandler, map[string]string{
		"permProjectKey": proj.Key,
	})
	req := assets.NewAuthentifiedRequest(t, u, jwt, "POST", uri, &wf)
	w := httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)
	require.Equal(t, 201, w.Code)
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &wf))

	lbl1 := sdk.Label{
		Name:      sdk.RandomString(5),
		ProjectID: proj.ID,
	}
	uri = router.GetRoute("POST", api.postWorkflowLabelHandler, map[string]string{
		"key":              proj.Key,
		"permWorkflowName": name,
	})
	req = assets.NewAuthentifiedRequest(t, u, jwt, "POST", uri, &lbl1)
	w = httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)
	require.Equal(t, 200, w.Code)
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &lbl1))

	require.NotEqual(t, 0, lbl1.ID)
	require.Equal(t, proj.ID, lbl1.ProjectID)
	require.Equal(t, wf.ID, lbl1.WorkflowID)

	wfUpdated, err := workflow.Load(context.TODO(), db, api.Cache, *proj, wf.Name, workflow.LoadOptions{WithLabels: true})
	require.NoError(t, err)
	require.NotNil(t, wfUpdated.Labels)
	require.Equal(t, 1, len(wfUpdated.Labels))
	require.Equal(t, lbl1.Name, wfUpdated.Labels[0].Name)

	// Unlink label
	uri = router.GetRoute("DELETE", api.deleteWorkflowLabelHandler, map[string]string{
		"key":              proj.Key,
		"permWorkflowName": name,
		"labelID":          fmt.Sprintf("%d", lbl1.ID),
	})
	req = assets.NewAuthentifiedRequest(t, u, jwt, "DELETE", uri, nil)
	w = httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)
	require.Equal(t, 200, w.Code)

	wfUpdated, err = workflow.Load(context.TODO(), db, api.Cache, *proj, wf.Name, workflow.LoadOptions{WithLabels: true})
	require.NoError(t, err)
	require.Equal(t, 0, len(wfUpdated.Labels))
}

func Test_deleteWorkflowHandler(t *testing.T) {
	api, db, router := newTestAPI(t)

	require.NoError(t, workflow.CreateBuiltinWorkflowHookModels(api.mustDB()))

	// Init user
	u, pass := assets.InsertAdminUser(t, db)
	// Init project
	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, api.Cache, key, key)
	// Init pipeline
	pip := sdk.Pipeline{
		Name:      "pipeline1",
		ProjectID: proj.ID,
	}
	test.NoError(t, pipeline.InsertPipeline(api.mustDB(), &pip))

	//Prepare request
	vars := map[string]string{
		"permProjectKey": proj.Key,
	}
	uri := router.GetRoute("POST", api.postWorkflowHandler, vars)
	test.NotEmpty(t, uri)

	var wkf = &sdk.Workflow{
		Name:        "Name",
		Description: "Description",
		WorkflowData: sdk.WorkflowData{
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
			wk, _ := workflow.Load(ctx, db, api.Cache, *proj, wkf.Name, workflow.LoadOptions{Minimal: true})
			if wk == nil {
				break loop
			}
		}
	}

}

func TestBenchmarkGetWorkflowsWithoutAPIAsAdmin(t *testing.T) {
	t.SkipNow()

	db, cache := test.SetupPG(t)

	// Init project
	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, cache, key, key)
	// Init pipeline
	pip := sdk.Pipeline{
		Name:      "pipeline1",
		ProjectID: proj.ID,
	}
	assert.NoError(t, pipeline.InsertPipeline(db, &pip))

	app := sdk.Application{
		Name: sdk.RandomString(10),
	}
	require.NoError(t, application.Insert(db, proj.ID, &app))

	prj, err := project.Load(context.TODO(), db, proj.Key,
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
			WorkflowData: sdk.WorkflowData{
				Node: sdk.Node{
					Name: "root",
					Context: &sdk.NodeContext{
						PipelineID:    pip.ID,
						ApplicationID: app.ID,
					},
				},
			},
		}

		assert.NoError(t, workflow.Insert(context.TODO(), db, cache, *prj, &wf))
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
	api, db, router := newTestAPI(t)

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
	assert.NoError(t, pipeline.InsertPipeline(db, &pip))

	app := sdk.Application{
		Name: sdk.RandomString(10),
	}
	require.NoError(t, application.Insert(db, proj.ID, &app))

	prj, err := project.Load(context.TODO(), db, proj.Key,
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
			WorkflowData: sdk.WorkflowData{
				Node: sdk.Node{
					Name: "root",
					Context: &sdk.NodeContext{
						PipelineID:    pip.ID,
						ApplicationID: app.ID,
					},
				},
			},
		}

		assert.NoError(t, workflow.Insert(context.TODO(), db, api.Cache, *prj, &wf))
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

func Test_putWorkflowShouldNotCallHOOKSIfHookDoesNotChange(t *testing.T) {
	api, db, router := newTestAPI(t)

	_, _ = assets.InsertService(t, db, t.Name()+"_HOOKS", sdk.TypeHooks)

	u, pass := assets.InsertAdminUser(t, db)
	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, api.Cache, key, key)
	pip := sdk.Pipeline{
		Name:      "pipeline1",
		ProjectID: proj.ID,
	}
	assert.NoError(t, pipeline.InsertPipeline(db, &pip))

	wf := sdk.Workflow{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       sdk.RandomString(10),
		Groups:     proj.ProjectGroups,
		WorkflowData: sdk.WorkflowData{
			Node: sdk.Node{
				Name: "root",
				Context: &sdk.NodeContext{
					PipelineID: pip.ID,
				},
				Hooks: []sdk.NodeHook{
					{
						HookModelID: sdk.WebHookModel.ID,
						Config: sdk.WorkflowNodeHookConfig{
							"method": sdk.WorkflowNodeHookConfigValue{
								Value: "POST",
							},
						},
					},
				},
			},
		},
	}

	// Setup a mock for all services called by the API
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	// The mock has been geenrated by mockgen: go get github.com/golang/mock/mockgen
	// If you have to regenerate thi mock you just have to run, from directory $GOPATH/src/github.com/ovh/cds/engine/api/services:
	// mockgen -source=http.go -destination=mock_services/services_mock.go Client
	servicesClients := mock_services.NewMockClient(ctrl)
	services.NewClient = func(_ gorp.SqlExecutor, _ []sdk.Service) services.Client {
		return servicesClients
	}
	defer func() {
		services.NewClient = services.NewDefaultClient
	}()

	// Mock the Hooks service
	servicesClients.EXPECT().
		DoJSONRequest(gomock.Any(), "POST", "/task/bulk", gomock.Any(), gomock.Any()).
		DoAndReturn(
			func(ctx context.Context, method, path string, in interface{}, out interface{}) (http.Header, int, error) {
				actualHooks, ok := in.(map[string]sdk.NodeHook)
				require.True(t, ok)
				require.Len(t, actualHooks, 1)
				for k, h := range actualHooks {
					h.Config["method"] = sdk.WorkflowNodeHookConfigValue{
						Value:        "POST",
						Configurable: true,
					}
					actualHooks[k] = h
				}
				out = actualHooks
				return nil, 200, nil
			},
		)

	// Insert the workflow
	vars := map[string]string{
		"permProjectKey": proj.Key,
	}
	uri := router.GetRoute("POST", api.postWorkflowHandler, vars)
	test.NotEmpty(t, uri)
	req := assets.NewAuthentifiedRequest(t, u, pass, "POST", uri, &wf)
	w := httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)
	assert.Equal(t, 201, w.Code)

	// Load the workflow
	vars = map[string]string{
		"key":              proj.Key,
		"permWorkflowName": wf.Name,
	}
	uri = router.GetRoute("GET", api.getWorkflowHandler, vars)
	test.NotEmpty(t, uri)
	req = assets.NewAuthentifiedRequest(t, u, pass, "GET", uri, nil)
	w = httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)

	// Unmarshal the workflow
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &wf))

	// Then call the PUT handler, it should not trigger /task/bulk on hooks service
	// Update the workflow
	uri = router.GetRoute("PUT", api.putWorkflowHandler, vars)
	test.NotEmpty(t, uri)
	req = assets.NewAuthentifiedRequest(t, u, pass, "PUT", uri, &wf)
	w = httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)

}

func Test_putWorkflowWithDuplicateHooksShouldRaiseAnError(t *testing.T) {
	api, db, router := newTestAPI(t)

	_, _ = assets.InsertService(t, db, t.Name()+"_HOOKS", sdk.TypeHooks)

	u, pass := assets.InsertAdminUser(t, db)
	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, api.Cache, key, key)
	pip := sdk.Pipeline{
		Name:      "pipeline1",
		ProjectID: proj.ID,
	}
	assert.NoError(t, pipeline.InsertPipeline(db, &pip))

	wf := sdk.Workflow{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       sdk.RandomString(10),
		Groups:     proj.ProjectGroups,
		WorkflowData: sdk.WorkflowData{
			Node: sdk.Node{
				Name: "root",
				Context: &sdk.NodeContext{
					PipelineID: pip.ID,
				},
				Hooks: []sdk.NodeHook{
					{
						HookModelID: sdk.WebHookModel.ID,
						Config: sdk.WorkflowNodeHookConfig{
							"method": sdk.WorkflowNodeHookConfigValue{
								Value: "POST",
							},
						},
					},
				},
			},
		},
	}

	// Setup a mock for all services called by the API
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	// The mock has been geenrated by mockgen: go get github.com/golang/mock/mockgen
	// If you have to regenerate thi mock you just have to run, from directory $GOPATH/src/github.com/ovh/cds/engine/api/services:
	// mockgen -source=http.go -destination=mock_services/services_mock.go Client
	servicesClients := mock_services.NewMockClient(ctrl)
	services.NewClient = func(_ gorp.SqlExecutor, _ []sdk.Service) services.Client {
		return servicesClients
	}
	defer func() {
		services.NewClient = services.NewDefaultClient
	}()

	// Mock the Hooks service
	servicesClients.EXPECT().
		DoJSONRequest(gomock.Any(), "POST", "/task/bulk", gomock.Any(), gomock.Any()).
		DoAndReturn(
			func(ctx context.Context, method, path string, in interface{}, out interface{}) (http.Header, int, error) {
				actualHooks, ok := in.(map[string]sdk.NodeHook)
				require.True(t, ok)
				require.Len(t, actualHooks, 1)
				for k, h := range actualHooks {
					h.Config["method"] = sdk.WorkflowNodeHookConfigValue{
						Value:        "POST",
						Configurable: true,
					}
					actualHooks[k] = h
				}
				out = actualHooks
				return nil, 200, nil
			},
		)

	// Insert the workflow
	vars := map[string]string{
		"permProjectKey": proj.Key,
	}
	uri := router.GetRoute("POST", api.postWorkflowHandler, vars)
	test.NotEmpty(t, uri)
	req := assets.NewAuthentifiedRequest(t, u, pass, "POST", uri, &wf)
	w := httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)
	assert.Equal(t, 201, w.Code)

	// Load the workflow
	vars = map[string]string{
		"key":              proj.Key,
		"permWorkflowName": wf.Name,
	}
	uri = router.GetRoute("GET", api.getWorkflowHandler, vars)
	test.NotEmpty(t, uri)
	req = assets.NewAuthentifiedRequest(t, u, pass, "GET", uri, nil)
	w = httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)

	// Unmarshal the workflow
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &wf))

	// Then add another hooks with similar properties. It should raise a 400 HTTP Error

	wf.WorkflowData.Node.Hooks = append(wf.WorkflowData.Node.Hooks,
		sdk.NodeHook{
			HookModelID: sdk.WebHookModel.ID,
			Config: sdk.WorkflowNodeHookConfig{
				"method": sdk.WorkflowNodeHookConfigValue{
					Value: "POST",
				},
			},
		},
	)

	// Update the workflow
	uri = router.GetRoute("PUT", api.putWorkflowHandler, vars)
	test.NotEmpty(t, uri)
	req = assets.NewAuthentifiedRequest(t, u, pass, "PUT", uri, &wf)
	w = httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)
	assert.Equal(t, 400, w.Code)

}

func Test_getWorkflowsHandler_FilterByRepo(t *testing.T) {
	api, db, tsURL := newTestServer(t)

	admin, _ := assets.InsertAdminUser(t, db)
	localConsumer, err := authentication.LoadConsumerByTypeAndUserID(context.TODO(), api.mustDB(), sdk.ConsumerLocal, admin.ID, authentication.LoadConsumerOptions.WithAuthentifiedUser)
	require.NoError(t, err)

	_, jws, err := builtin.NewConsumer(context.TODO(), db, sdk.RandomString(10), sdk.RandomString(10), localConsumer, admin.GetGroupIDs(),
		sdk.NewAuthConsumerScopeDetails(sdk.AuthConsumerScopeProject))

	u, _ := assets.InsertLambdaUser(t, db)

	pkey := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, api.Cache, pkey, pkey)
	require.NoError(t, group.InsertLinkGroupUser(context.TODO(), db, &group.LinkGroupUser{
		GroupID:            proj.ProjectGroups[0].Group.ID,
		AuthentifiedUserID: u.ID,
		Admin:              true,
	}))

	repofullName := sdk.RandomString(10)

	app := &sdk.Application{
		Name:               sdk.RandomString(10),
		RepositoryFullname: "ovh/" + repofullName,
	}
	require.NoError(t, application.Insert(db, proj.ID, app))

	pip := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip1",
	}
	test.NoError(t, pipeline.InsertPipeline(api.mustDB(), &pip))

	wf := sdk.Workflow{
		Name:       "workflow1",
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		WorkflowData: sdk.WorkflowData{
			Node: sdk.Node{
				Name: "root",
				Context: &sdk.NodeContext{
					PipelineID:    pip.ID,
					ApplicationID: app.ID,
				},
			},
		},
	}
	require.NoError(t, workflow.Insert(context.TODO(), db, api.Cache, *proj, &wf))

	wf2 := sdk.Workflow{
		Name:       "workflow2",
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		WorkflowData: sdk.WorkflowData{
			Node: sdk.Node{
				Name: "root",
				Context: &sdk.NodeContext{
					PipelineID: pip.ID,
				},
			},
		},
	}
	require.NoError(t, workflow.Insert(context.TODO(), db, api.Cache, *proj, &wf2))

	// Call with an admin
	sdkclientAdmin := cdsclient.New(cdsclient.Config{
		Host:                              tsURL,
		BuitinConsumerAuthenticationToken: jws,
	})

	wfs, err := sdkclientAdmin.WorkflowList(proj.Key, cdsclient.WithQueryParameter("repo", "ovh/"+repofullName))
	require.NoError(t, err)
	require.Len(t, wfs, 1)
	require.Equal(t, wf.Name, wfs[0].Name)
	require.Equal(t, app.ID, wfs[0].WorkflowData.Node.Context.ApplicationID)
	require.Equal(t, pip.ID, wfs[0].WorkflowData.Node.Context.PipelineID)
}

func Test_getSearchWorkflowHandler(t *testing.T) {
	api, db, tsURL := newTestServer(t)

	admin, _ := assets.InsertAdminUser(t, db)
	localConsumer, err := authentication.LoadConsumerByTypeAndUserID(context.TODO(), api.mustDB(), sdk.ConsumerLocal, admin.ID, authentication.LoadConsumerOptions.WithAuthentifiedUser)
	require.NoError(t, err)

	_, jws, err := builtin.NewConsumer(context.TODO(), db, sdk.RandomString(10), sdk.RandomString(10), localConsumer, admin.GetGroupIDs(),
		sdk.NewAuthConsumerScopeDetails(sdk.AuthConsumerScopeProject))

	u, _ := assets.InsertLambdaUser(t, db)

	pkey := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, api.Cache, pkey, pkey)
	require.NoError(t, group.InsertLinkGroupUser(context.TODO(), db, &group.LinkGroupUser{
		GroupID:            proj.ProjectGroups[0].Group.ID,
		AuthentifiedUserID: u.ID,
		Admin:              true,
	}))

	repofullName := sdk.RandomString(20)

	app := &sdk.Application{
		Name:               sdk.RandomString(10),
		RepositoryFullname: "ovh/" + repofullName,
	}
	require.NoError(t, application.Insert(db, proj.ID, app))

	pip := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip1",
	}
	test.NoError(t, pipeline.InsertPipeline(api.mustDB(), &pip))

	wf := sdk.Workflow{
		Name:       "workflow1",
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		WorkflowData: sdk.WorkflowData{
			Node: sdk.Node{
				Name: "root",
				Context: &sdk.NodeContext{
					PipelineID:    pip.ID,
					ApplicationID: app.ID,
				},
			},
		},
	}
	require.NoError(t, workflow.Insert(context.TODO(), db, api.Cache, *proj, &wf))

	wf2 := sdk.Workflow{
		Name:       "workflow2",
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		WorkflowData: sdk.WorkflowData{
			Node: sdk.Node{
				Name: "root",
				Context: &sdk.NodeContext{
					PipelineID: pip.ID,
				},
			},
		},
	}
	require.NoError(t, workflow.Insert(context.TODO(), db, api.Cache, *proj, &wf2))

	// Run the workflow
	consumer, _ := authentication.LoadConsumerByTypeAndUserID(context.TODO(), api.mustDB(), sdk.ConsumerLocal, u.ID, authentication.LoadConsumerOptions.WithAuthentifiedUser)
	consumerAdmin, _ := authentication.LoadConsumerByTypeAndUserID(context.TODO(), api.mustDB(), sdk.ConsumerLocal, admin.ID, authentication.LoadConsumerOptions.WithAuthentifiedUser)

	wr, err := workflow.CreateRun(api.mustDB(), &wf, sdk.WorkflowRunPostHandlerOption{AuthConsumerID: consumerAdmin.ID})
	assert.NoError(t, err)
	wr.Workflow = wf
	wr.Tag("git.branch", "master")
	_, err = workflow.StartWorkflowRun(context.TODO(), db, api.Cache, *proj, wr, &sdk.WorkflowRunPostHandlerOption{
		Manual: &sdk.WorkflowNodeRunManual{
			Username: u.GetUsername(),
			Payload:  `{"git.branch": "master"}`,
		},
	}, *consumer, nil)
	require.NoError(t, err)

	// Call with an admin
	sdkclientAdmin := cdsclient.New(cdsclient.Config{
		Host:                              tsURL,
		BuitinConsumerAuthenticationToken: jws,
	})

	wfs, err := sdkclientAdmin.WorkflowSearch(
		cdsclient.WithQueryParameter("repository", "ovh/"+repofullName),
		cdsclient.WithQueryParameter("runs", "10"),
	)
	require.NoError(t, err)
	require.Len(t, wfs, 1)
	require.Equal(t, wf.Name, wfs[0].Name)
	require.NotEmpty(t, wfs[0].URLs.APIURL)
	require.NotEmpty(t, wfs[0].URLs.UIURL)
	require.Equal(t, app.ID, wfs[0].WorkflowData.Node.Context.ApplicationID)
	require.Equal(t, pip.ID, wfs[0].WorkflowData.Node.Context.PipelineID)
	require.NotEmpty(t, wfs[0].Runs)
	require.NotEmpty(t, wfs[0].Runs[0].URLs.APIURL)
	require.NotEmpty(t, wfs[0].Runs[0].URLs.UIURL)

	t.Logf("%+v", wfs[0].Runs[0].URLs)

}
