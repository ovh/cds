package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-gorp/gorp"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"

	"github.com/ovh/cds/engine/api/auth"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/sdk"
)

func Test_getWorkflowsHandler(t *testing.T) {
	// Init database
	db := test.SetupPG(t)

	// Init router
	router = newRouter(auth.TestLocalAuth(t), mux.NewRouter(), "/Test_getWorkflowsHandler")
	router.init()
	// Init user
	u, pass := assets.InsertAdminUser(db)
	// Init project
	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, key, key, u)
	//Prepare request
	vars := map[string]string{
		"permProjectKey": proj.Key,
	}
	uri := router.getRoute("GET", getWorkflowsHandler, vars)
	test.NotEmpty(t, uri)
	req := assets.NewAuthentifiedRequest(t, u, pass, "GET", uri, vars)

	//Do the request
	w := httptest.NewRecorder()
	router.mux.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)
}

func Test_getWorkflowHandler(t *testing.T) {
	// Init database
	db := test.SetupPG(t)

	// Init router
	router = newRouter(auth.TestLocalAuth(t), mux.NewRouter(), "/Test_getWorkflowHandler")
	router.init()
	// Init user
	u, pass := assets.InsertAdminUser(db)
	// Init project
	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, key, key, u)
	//Prepare request
	vars := map[string]string{
		"permProjectKey": proj.Key,
		"workflowName":   "workflow1",
	}
	uri := router.getRoute("GET", getWorkflowHandler, vars)
	test.NotEmpty(t, uri)

	req := assets.NewAuthentifiedRequest(t, u, pass, "GET", uri, nil)
	//Do the request
	w := httptest.NewRecorder()
	router.mux.ServeHTTP(w, req)
	assert.Equal(t, 404, w.Code)
}

func Test_postWorkflowHandlerWithoutRootShouldFail(t *testing.T) {
	// Init database
	db := test.SetupPG(t)
	// Init router
	router = newRouter(auth.TestLocalAuth(t), mux.NewRouter(), "/Test_postWorkflowHandler")
	router.init()
	// Init user
	u, pass := assets.InsertAdminUser(db)
	// Init project
	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, key, key, u)
	//Prepare request
	vars := map[string]string{
		"permProjectKey": proj.Key,
	}
	uri := router.getRoute("POST", postWorkflowHandler, vars)
	test.NotEmpty(t, uri)

	var workflowResponse sdk.Workflow
	req := assets.NewAuthentifiedRequest(t, u, pass, "POST", uri, &workflowResponse)
	//Do the request
	w := httptest.NewRecorder()
	router.mux.ServeHTTP(w, req)
	assert.Equal(t, 400, w.Code)
}

func Test_postWorkflowHandlerWithRootShouldSuccess(t *testing.T) {
	// Init database
	db := test.SetupPG(t)
	// Init router
	router = newRouter(auth.TestLocalAuth(t), mux.NewRouter(), "/Test_postWorkflowHandler")
	router.init()
	// Init user
	u, pass := assets.InsertAdminUser(db)
	// Init project
	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, key, key, u)
	// Init pipeline
	pip := sdk.Pipeline{
		Name:      "pipeline1",
		ProjectID: proj.ID,
		Type:      sdk.BuildPipeline,
	}
	test.NoError(t, pipeline.InsertPipeline(db, &pip, nil))

	//Prepare request
	vars := map[string]string{
		"permProjectKey": proj.Key,
	}
	uri := router.getRoute("POST", postWorkflowHandler, vars)
	test.NotEmpty(t, uri)

	var workflow = &sdk.Workflow{
		Name:        "Name",
		Description: "Description",
		Root: &sdk.WorkflowNode{
			PipelineID: pip.ID,
		},
	}

	req := assets.NewAuthentifiedRequest(t, u, pass, "POST", uri, &workflow)
	//Do the request
	w := httptest.NewRecorder()
	router.mux.ServeHTTP(w, req)
	assert.Equal(t, 201, w.Code)

	test.NoError(t, json.Unmarshal(w.Body.Bytes(), &workflow))
	assert.NotEqual(t, 0, workflow.ID)
}

func Test_putWorkflowHandler(t *testing.T) {
	// Init database
	db := test.SetupPG(t)
	// Init router
	router = newRouter(auth.TestLocalAuth(t), mux.NewRouter(), "/Test_postWorkflowHandler")
	router.init()
	// Init user
	u, pass := assets.InsertAdminUser(db)
	// Init project
	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, key, key, u)
	// Init pipeline
	pip := sdk.Pipeline{
		Name:      "pipeline1",
		ProjectID: proj.ID,
		Type:      sdk.BuildPipeline,
	}
	test.NoError(t, pipeline.InsertPipeline(db, &pip, nil))

	//Prepare request
	vars := map[string]string{
		"permProjectKey": proj.Key,
	}
	uri := router.getRoute("POST", postWorkflowHandler, vars)
	test.NotEmpty(t, uri)

	var workflow = &sdk.Workflow{
		Name:        "Name",
		Description: "Description",
		Root: &sdk.WorkflowNode{
			PipelineID: pip.ID,
		},
	}

	req := assets.NewAuthentifiedRequest(t, u, pass, "POST", uri, &workflow)

	//Do the request
	w := httptest.NewRecorder()
	router.mux.ServeHTTP(w, req)
	assert.Equal(t, 201, w.Code)
	test.NoError(t, json.Unmarshal(w.Body.Bytes(), &workflow))

	//Prepare request
	vars = map[string]string{
		"permProjectKey": proj.Key,
		"workflowName":   "Name",
	}
	uri = router.getRoute("PUT", putWorkflowHandler, vars)
	test.NotEmpty(t, uri)

	var workflow1 = &sdk.Workflow{
		Name:        "Name 2",
		Description: "Description 2",
		Root: &sdk.WorkflowNode{
			PipelineID: pip.ID,
		},
	}

	req = assets.NewAuthentifiedRequest(t, u, pass, "PUT", uri, &workflow1)

	//Do the request
	w = httptest.NewRecorder()
	router.mux.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)

	test.NoError(t, json.Unmarshal(w.Body.Bytes(), &workflow1))

	assert.NotEqual(t, 0, workflow1.ID)
	assert.Equal(t, "Name 2", workflow1.Name)
}

func Test_deleteWorkflowHandler(t *testing.T) {
	// Init database
	db := test.SetupPG(t)
	// Init router
	router = newRouter(auth.TestLocalAuth(t), mux.NewRouter(), "/Test_deleteWorkflowHandler")
	router.init()
	// Init user
	u, pass := assets.InsertAdminUser(db)
	// Init project
	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, key, key, u)
	// Init pipeline
	pip := sdk.Pipeline{
		Name:      "pipeline1",
		ProjectID: proj.ID,
		Type:      sdk.BuildPipeline,
	}
	test.NoError(t, pipeline.InsertPipeline(db, &pip, nil))

	//Prepare request
	vars := map[string]string{
		"permProjectKey": proj.Key,
	}
	uri := router.getRoute("POST", postWorkflowHandler, vars)
	test.NotEmpty(t, uri)

	var workflow = &sdk.Workflow{
		Name:        "Name",
		Description: "Description",
		Root: &sdk.WorkflowNode{
			PipelineID: pip.ID,
		},
	}

	req := assets.NewAuthentifiedRequest(t, u, pass, "POST", uri, &workflow)

	//Do the request
	w := httptest.NewRecorder()
	router.mux.ServeHTTP(w, req)
	assert.Equal(t, 201, w.Code)
	test.NoError(t, json.Unmarshal(w.Body.Bytes(), &workflow))

	vars = map[string]string{
		"permProjectKey": proj.Key,
		"workflowName":   "Name",
	}
	uri = router.getRoute("DELETE", deleteWorkflowHandler, vars)
	test.NotEmpty(t, uri)

	req = assets.NewAuthentifiedRequest(t, u, pass, "DELETE", uri, nil)

	//Do the request
	w = httptest.NewRecorder()
	router.mux.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)
}

func Test_getWorkflowNodeRunJobStepHandler(t *testing.T) {
	db := test.SetupPG(t)
	u, pass := assets.InsertAdminUser(db)
	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, key, key, u)

	//First pipeline
	pip := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip1",
		Type:       sdk.BuildPipeline,
	}
	test.NoError(t, pipeline.InsertPipeline(db, &pip, u))

	s := sdk.NewStage("stage 1")
	s.Enabled = true
	s.PipelineID = pip.ID
	pipeline.InsertStage(db, s)
	j := &sdk.Job{
		Enabled: true,
		Action: sdk.Action{
			Enabled: true,
		},
	}
	pipeline.InsertJob(db, j, s.ID, &pip)
	s.Jobs = append(s.Jobs, *j)

	pip.Stages = append(pip.Stages, *s)

	//Second pipeline
	pip2 := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip2",
		Type:       sdk.BuildPipeline,
	}
	test.NoError(t, pipeline.InsertPipeline(db, &pip2, u))
	s = sdk.NewStage("stage 1")
	s.Enabled = true
	s.PipelineID = pip2.ID
	pipeline.InsertStage(db, s)
	j = &sdk.Job{
		Enabled: true,
		Action: sdk.Action{
			Enabled: true,
		},
	}
	pipeline.InsertJob(db, j, s.ID, &pip2)
	s.Jobs = append(s.Jobs, *j)

	w := sdk.Workflow{
		Name:       "test_1",
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Root: &sdk.WorkflowNode{
			Pipeline: pip,
			Triggers: []sdk.WorkflowNodeTrigger{
				{
					WorkflowDestNode: sdk.WorkflowNode{
						Pipeline: pip,
					},
				},
			},
		},
	}

	test.NoError(t, workflow.Insert(db, &w, u))
	w1, err := workflow.Load(db, key, "test_1", u)
	test.NoError(t, err)

	_, err = workflow.ManualRun(db, w1, &sdk.WorkflowNodeRunManual{
		User: *u,
	})
	test.NoError(t, err)

	c, cancel := context.WithTimeout(context.Background(), time.Second*2)
	defer cancel()
	workflow.Scheduler(c, func() *gorp.DbMap { return db })
	time.Sleep(2 * time.Second)

	lastrun, err := workflow.LoadLastRun(db, proj.Key, w1.Name)

	// Update step status
	jobRun := &lastrun.WorkflowNodeRuns[w1.RootID][0].Stages[0].RunJobs[0]
	log := &sdk.Log{
		StepOrder: 1,
		Val:       "My Log",
	}
	jobRun.Job.StepStatus = []sdk.StepStatus{
		{
			StepOrder: 1,
			Status:    sdk.StatusBuilding.String(),
		},
	}

	// Update node job run
	errUJ := workflow.UpdateNodeRun(db, &lastrun.WorkflowNodeRuns[w1.RootID][0])
	test.NoError(t, errUJ)

	// Add log
	errAL := workflow.AddLog(db, jobRun, log)
	test.NoError(t, errAL)

	// Init router
	router = newRouter(auth.TestLocalAuth(t), mux.NewRouter(), "/Test_getWorkflowNodeRunJobStepHandler")
	router.init()
	//Prepare request
	vars := map[string]string{
		"permProjectKey": proj.Key,
		"workflowName":   w1.Name,
		"number":         fmt.Sprintf("%d", lastrun.Number),
		"id":             fmt.Sprintf("%d", lastrun.WorkflowNodeRuns[w1.RootID][0].ID),
		"runJobId":       fmt.Sprintf("%d", jobRun.ID),
		"stepOrder":      "1",
	}
	uri := router.getRoute("GET", getWorkflowNodeRunJobStepHandler, vars)
	test.NotEmpty(t, uri)
	req := assets.NewAuthentifiedRequest(t, u, pass, "GET", uri, vars)

	//Do the request
	rec := httptest.NewRecorder()
	router.mux.ServeHTTP(rec, req)

	stepState := &sdk.BuildState{}
	json.Unmarshal(rec.Body.Bytes(), stepState)
	assert.Equal(t, 200, rec.Code)
	assert.Equal(t, "My Log", stepState.StepLogs.Val)
	assert.Equal(t, sdk.StatusBuilding, stepState.Status)
}
