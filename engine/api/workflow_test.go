package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v2"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/group"
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
	// Init database
	api, db, router, end := newTestAPI(t)
	defer end()

	// Init user
	u, pass := assets.InsertAdminUser(api.mustDB())
	// Init project
	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, api.Cache, key, key, u)
	//Prepare request
	vars := map[string]string{
		"permProjectKey": proj.Key,
	}
	uri := router.GetRoute("GET", api.getWorkflowsHandler, vars)
	test.NotEmpty(t, uri)
	req := assets.NewAuthentifiedRequest(t, u, pass, "GET", uri, vars)

	//Do the request
	w := httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)
}

func Test_getWorkflowHandler(t *testing.T) {
	// Init database
	api, db, router, end := newTestAPI(t)
	defer end()

	// Init user
	u, pass := assets.InsertAdminUser(api.mustDB())
	// Init project
	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, api.Cache, key, key, u)
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

func Test_getWorkflowHandler_AsProvider(t *testing.T) {
	api, tsURL, tsClose := newTestServer(t)
	defer tsClose()

	api.Config.Providers = append(api.Config.Providers, ProviderConfiguration{
		Name:  "test-provider",
		Token: "my-token",
	})

	u, _ := assets.InsertLambdaUser(api.mustDB())

	pkey := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, api.mustDB(), api.Cache, pkey, pkey, u)
	test.NoError(t, group.InsertUserInGroup(api.mustDB(), proj.ProjectGroups[0].Group.ID, u.ID, true))

	pip := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip1",
	}

	test.NoError(t, pipeline.InsertPipeline(api.mustDB(), api.Cache, proj, &pip, u))

	proj, _ = project.LoadByID(api.mustDB(), api.Cache, proj.ID, u, project.LoadOptions.WithApplications, project.LoadOptions.WithPipelines, project.LoadOptions.WithEnvironments, project.LoadOptions.WithGroups)

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

	test.NoError(t, workflow.Insert(api.mustDB(), api.Cache, &wf, proj, u))

	sdkclient := cdsclient.NewProviderClient(cdsclient.ProviderConfig{
		Host:  tsURL,
		Name:  "test-provider",
		Token: "my-token",
	})

	w, err := sdkclient.WorkflowLoad(pkey, wf.Name)
	test.NoError(t, err)
	t.Logf("%+v", w)

	///
}

func Test_getWorkflowHandler_withUsage(t *testing.T) {
	// Init database
	api, db, router, end := newTestAPI(t)
	defer end()

	// Init user
	u, pass := assets.InsertAdminUser(api.mustDB())
	// Init project
	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, api.Cache, key, key, u)
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

	test.NoError(t, pipeline.InsertPipeline(db, api.Cache, proj, &pip, u))

	proj, _ = project.LoadByID(db, api.Cache, proj.ID, u, project.LoadOptions.WithApplications, project.LoadOptions.WithPipelines, project.LoadOptions.WithEnvironments, project.LoadOptions.WithGroups)

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

	test.NoError(t, workflow.Insert(db, api.Cache, &wf, proj, u))

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
	// Init database
	api, db, router, end := newTestAPI(t)
	defer end()

	// Init user
	u, pass := assets.InsertAdminUser(api.mustDB())
	// Init project
	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, api.Cache, key, key, u)
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
	// Init database
	api, db, router, end := newTestAPI(t)
	defer end()

	// Init user
	u, pass := assets.InsertAdminUser(api.mustDB())
	// Init project
	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, api.Cache, key, key, u)
	// Init pipeline
	pip := sdk.Pipeline{
		Name:      "pipeline1",
		ProjectID: proj.ID,
	}
	test.NoError(t, pipeline.InsertPipeline(api.mustDB(), api.Cache, proj, &pip, nil))

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
	test.NoError(t, application.Insert(api.mustDB(), api.Cache, proj, &app, u))

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
	// Init database
	api, db, router, end := newTestAPI(t)
	defer end()

	// Init user
	u, pass := assets.InsertAdminUser(api.mustDB())
	// Init project
	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, api.Cache, key, key, u)
	// Init pipeline
	pip := sdk.Pipeline{
		Name:      "pipeline1",
		ProjectID: proj.ID,
	}
	test.NoError(t, pipeline.InsertPipeline(api.mustDB(), api.Cache, proj, &pip, nil))

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
	test.NoError(t, application.Insert(api.mustDB(), api.Cache, proj, &app, u))

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
	// Init database
	api, db, router, end := newTestAPI(t)
	defer end()

	// Init user
	u, pass := assets.InsertAdminUser(api.mustDB())
	// Init project
	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, api.Cache, key, key, u)

	// Init pipeline
	pip := sdk.Pipeline{
		Name:      "pipeline1",
		ProjectID: proj.ID,
	}
	test.NoError(t, pipeline.InsertPipeline(api.mustDB(), api.Cache, proj, &pip, nil))

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
			},
		},
	}

	req := assets.NewAuthentifiedRequest(t, u, pass, "POST", uri, &workflow)

	//Do the request
	w := httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)
	assert.Equal(t, 201, w.Code)
	test.NoError(t, json.Unmarshal(w.Body.Bytes(), &workflow))

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
	test.NoError(t, application.Insert(api.mustDB(), api.Cache, proj, &app, u))

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
}

// TODO: to uncomment
// func Test_postWorkflowHandlerWithError(t *testing.T) {
// 	// Init database
// 	api, db, router, end := newTestAPI(t)
// 	defer end()

// 	// Init user
// 	u, pass := assets.InsertAdminUser(api.mustDB())
// 	// Init project
// 	key := sdk.RandomString(10)
// 	proj := assets.InsertTestProject(t, db, api.Cache, key, key, u)

// 	// Init pipeline
// 	pip := sdk.Pipeline{
// 		Name:      "pipeline1",
// 		ProjectID: proj.ID,
// 	}
// 	test.NoError(t, pipeline.InsertPipeline(api.mustDB(), api.Cache, proj, &pip, nil))

// 	//Prepare request
// 	vars := map[string]string{
// 		"permProjectKey": proj.Key,
// 	}
// 	uri := router.GetRoute("POST", api.postWorkflowHandler, vars)
// 	test.NotEmpty(t, uri)

// 	var workflow = &sdk.Workflow{
// 		Name:        "Name",
// 		Description: "Description",
// 		WorkflowData: &sdk.WorkflowData{
// 			Node: sdk.Node{
// 				Type: sdk.NodeTypePipeline,
// 				Context: &sdk.NodeContext{
// 					PipelineID: pip.ID,
// 				},
// 				Triggers: []sdk.NodeTrigger{{
// 					ChildNode: sdk.Node{
// 						Type: sdk.NodeTypePipeline,
// 						Context: &sdk.NodeContext{
// 							PipelineID: pip.ID,
// 							DefaultPayload: map[string]interface{}{
// 								"test": "content",
// 							},
// 						},
// 					},
// 				}},
// 			},
// 		},
// 	}

// 	req := assets.NewAuthentifiedRequest(t, u, pass, "POST", uri, &workflow)

// 	//Do the request
// 	w := httptest.NewRecorder()
// 	router.Mux.ServeHTTP(w, req)
// 	assert.Equal(t, 400, w.Code)
// }

func Test_postWorkflowRollbackHandler(t *testing.T) {
	// Init database
	api, db, router, end := newTestAPI(t)
	defer end()

	// Init user
	u, pass := assets.InsertAdminUser(api.mustDB())
	// Init project
	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, api.Cache, key, key, u)

	// Init pipeline
	pip := sdk.Pipeline{
		Name:      "pipeline1",
		ProjectID: proj.ID,
	}
	test.NoError(t, pipeline.InsertPipeline(api.mustDB(), api.Cache, proj, &pip, nil))

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
	test.NoError(t, application.Insert(api.mustDB(), api.Cache, proj, &app, u))

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

	test.NoError(t, workflow.IsValid(context.Background(), api.Cache, db, wf, proj, u, workflow.LoadOptions{}))
	eWf, err := exportentities.NewWorkflow(*wf)
	test.NoError(t, err)
	wfBts, err := yaml.Marshal(eWf)
	test.NoError(t, err)
	eWfUpdate, err := exportentities.NewWorkflow(*workflow1)
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
}

func Test_postAndDeleteWorkflowLabelHandler(t *testing.T) {
	// Init database
	api, db, router, end := newTestAPI(t)
	defer end()

	// Init user
	u, pass := assets.InsertAdminUser(api.mustDB())
	// Init project
	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, api.Cache, key, key, u)

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
	test.NoError(t, pipeline.InsertPipeline(api.mustDB(), api.Cache, proj, &pip, nil))

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

	wfUpdated, errW := workflow.Load(context.TODO(), db, api.Cache, proj, wf.Name, u, workflow.LoadOptions{WithLabels: true})
	test.NoError(t, errW)

	assert.NotNil(t, wfUpdated.Labels)
	assert.Equal(t, 1, len(wfUpdated.Labels))
	assert.Equal(t, lbl1.Name, wfUpdated.Labels[0].Name)

	//Unlink label
	vars = map[string]string{
		"key":              proj.Key,
		"permWorkflowName": "Name",
		"labelID":          fmt.Sprintf("%d", lbl1.ID),
	}
	uri = router.GetRoute("DELETE", api.deleteWorkflowLabelHandler, vars)
	test.NotEmpty(t, uri)

	req = assets.NewAuthentifiedRequest(t, u, pass, "DELETE", uri, nil)
	//Do the request
	w = httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)

	wfUpdated, errW = workflow.Load(context.TODO(), db, api.Cache, proj, wf.Name, u, workflow.LoadOptions{WithLabels: true})
	test.NoError(t, errW)
	assert.NotNil(t, wfUpdated.Labels)
	assert.Equal(t, 0, len(wfUpdated.Labels))
}

func Test_deleteWorkflowHandler(t *testing.T) {
	// Init database
	api, db, router, end := newTestAPI(t)
	defer end()
	test.NoError(t, workflow.CreateBuiltinWorkflowHookModels(db))

	// Init user
	u, pass := assets.InsertAdminUser(api.mustDB())
	// Init project
	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, api.Cache, key, key, u)
	// Init pipeline
	pip := sdk.Pipeline{
		Name:      "pipeline1",
		ProjectID: proj.ID,
	}
	test.NoError(t, pipeline.InsertPipeline(api.mustDB(), api.Cache, proj, &pip, nil))

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
			},
		},
	}

	req := assets.NewAuthentifiedRequest(t, u, pass, "POST", uri, &workflow)

	//Do the request
	w := httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)
	assert.Equal(t, 201, w.Code)
	test.NoError(t, json.Unmarshal(w.Body.Bytes(), &workflow))

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
}

func TestBenchmarkGetWorkflowsWithoutAPIAsAdmin(t *testing.T) {
	t.SkipNow()
	// Init database
	db, cache, end := test.SetupPG(t)
	defer end()

	// Init user
	u, _ := assets.InsertAdminUser(db)
	// Init project
	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, cache, key, key, u)
	// Init pipeline
	pip := sdk.Pipeline{
		Name:      "pipeline1",
		ProjectID: proj.ID,
	}

	assert.NoError(t, pipeline.InsertPipeline(db, cache, proj, &pip, nil))

	app := sdk.Application{
		Name: sdk.RandomString(10),
	}

	assert.NoError(t, application.Insert(db, cache, proj, &app, u))

	prj, err := project.Load(db, cache, proj.Key, u, project.LoadOptions.WithPipelines, project.LoadOptions.WithApplications, project.LoadOptions.WithWorkflows)
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

		assert.NoError(t, workflow.Insert(db, cache, &wf, prj, u))
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
	proj := assets.InsertTestProject(t, db, api.Cache, key, key, nil)

	// Init user
	u, pass := assets.InsertLambdaUser(db, &proj.ProjectGroups[0].Group)

	// Init pipeline
	pip := sdk.Pipeline{
		Name:      "pipeline1",
		ProjectID: proj.ID,
	}
	assert.NoError(t, pipeline.InsertPipeline(db, api.Cache, proj, &pip, u))

	app := sdk.Application{
		Name: sdk.RandomString(10),
	}

	assert.NoError(t, application.Insert(db, api.Cache, proj, &app, u))

	prj, err := project.Load(db, api.Cache, proj.Key, u, project.LoadOptions.WithPipelines, project.LoadOptions.WithApplications, project.LoadOptions.WithWorkflows)
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

		assert.NoError(t, workflow.Insert(db, api.Cache, &wf, prj, u))
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
