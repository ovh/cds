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
	"github.com/ovh/cds/engine/api/bootstrap"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/sdk"
)

func Test_getWorkflowNodeRunHistoryHandler(t *testing.T) {
	db := test.SetupPG(t, bootstrap.InitiliazeDB)
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
	test.NoError(t, pipeline.InsertPipeline(db, proj, &pip, u))

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
	test.NoError(t, pipeline.InsertPipeline(db, proj, &pip2, u))
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
				sdk.WorkflowNodeTrigger{
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

	wr, errMR := workflow.ManualRun(db, w1, &sdk.WorkflowNodeRunManual{
		User: *u,
	})
	if errMR != nil {
		test.NoError(t, errMR)
	}

	_, errMR2 := workflow.ManualRunFromNode(db, &wr.Workflow, wr.Number, &sdk.WorkflowNodeRunManual{User: *u}, wr.Workflow.RootID)
	if errMR2 != nil {
		test.NoError(t, errMR2)
	}

	// Init router
	router = newRouter(auth.TestLocalAuth(t), mux.NewRouter(), "/Test_getWorkflowNodeRunHistoryHandler")
	router.init()
	//Prepare request
	vars := map[string]string{
		"permProjectKey": proj.Key,
		"workflowName":   w1.Name,
		"number":         fmt.Sprintf("%d", wr.Number),
		"nodeID":         fmt.Sprintf("%d", wr.Workflow.RootID),
	}
	uri := router.getRoute("GET", getWorkflowNodeRunHistoryHandler, vars)
	test.NotEmpty(t, uri)
	req := assets.NewAuthentifiedRequest(t, u, pass, "GET", uri, vars)

	//Do the request
	rec := httptest.NewRecorder()
	router.mux.ServeHTTP(rec, req)
	assert.Equal(t, 200, rec.Code)

	history := []sdk.WorkflowNodeRun{}
	test.NoError(t, json.Unmarshal(rec.Body.Bytes(), &history))
	assert.Equal(t, 2, len(history))
	assert.Equal(t, int64(1), history[0].SubNumber)
	assert.Equal(t, int64(0), history[1].SubNumber)
}
func Test_getWorkflowRunsHandler(t *testing.T) {
	db := test.SetupPG(t, bootstrap.InitiliazeDB)
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
	test.NoError(t, pipeline.InsertPipeline(db, proj, &pip, u))

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
	test.NoError(t, pipeline.InsertPipeline(db, proj, &pip2, u))
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
				sdk.WorkflowNodeTrigger{
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

	for i := 0; i < 10; i++ {
		_, err = workflow.ManualRun(db, w1, &sdk.WorkflowNodeRunManual{
			User: *u,
		})
		test.NoError(t, err)
	}

	// Init router
	router = newRouter(auth.TestLocalAuth(t), mux.NewRouter(), "/Test_getWorkflowRunsHandler")
	router.init()
	//Prepare request
	vars := map[string]string{
		"permProjectKey": proj.Key,
		"workflowName":   w1.Name,
	}
	uri := router.getRoute("GET", getWorkflowRunsHandler, vars)
	test.NotEmpty(t, uri)
	req := assets.NewAuthentifiedRequest(t, u, pass, "GET", uri, vars)

	//Do the request
	rec := httptest.NewRecorder()
	router.mux.ServeHTTP(rec, req)
	assert.Equal(t, 200, rec.Code)
	assert.Equal(t, "0-10/10", rec.Header().Get("Content-Range"))

	uri = router.getRoute("GET", getWorkflowRunsHandler, vars)
	test.NotEmpty(t, uri)
	req = assets.NewAuthentifiedRequest(t, u, pass, "GET", uri, vars)
	q := req.URL.Query()
	q.Set("offset", "5")
	q.Set("limit", "9")
	req.URL.RawQuery = q.Encode()
	//Do the request
	rec = httptest.NewRecorder()
	router.mux.ServeHTTP(rec, req)
	assert.Equal(t, 206, rec.Code)
	assert.Equal(t, "5-9/10", rec.Header().Get("Content-Range"))

	link := rec.Header().Get("Link")
	assert.NotEmpty(t, link)
	t.Log(link)

	test.NotEmpty(t, uri)
	req = assets.NewAuthentifiedRequest(t, u, pass, "GET", uri, vars)
	q = req.URL.Query()
	q.Set("offset", "0")
	q.Set("limit", "100")
	req.URL.RawQuery = q.Encode()
	//Do the request
	rec = httptest.NewRecorder()
	router.mux.ServeHTTP(rec, req)
	assert.Equal(t, 400, rec.Code)
	assert.Equal(t, "", rec.Header().Get("Content-Range"))

}

func Test_getLatestWorkflowRunHandler(t *testing.T) {
	db := test.SetupPG(t, bootstrap.InitiliazeDB)
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
	test.NoError(t, pipeline.InsertPipeline(db, proj, &pip, u))

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
	test.NoError(t, pipeline.InsertPipeline(db, proj, &pip2, u))
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
				sdk.WorkflowNodeTrigger{
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

	for i := 0; i < 10; i++ {
		_, err = workflow.ManualRun(db, w1, &sdk.WorkflowNodeRunManual{
			User: *u,
			Payload: map[string]string{
				"git.branch": "master",
				"git.hash":   fmt.Sprintf("%d", i),
			},
		})
		test.NoError(t, err)
	}

	// Init router
	router = newRouter(auth.TestLocalAuth(t), mux.NewRouter(), "/Test_getLatestWorkflowRunHandler")
	router.init()
	//Prepare request
	vars := map[string]string{
		"permProjectKey": proj.Key,
		"workflowName":   w1.Name,
	}
	uri := router.getRoute("GET", getLatestWorkflowRunHandler, vars)
	test.NotEmpty(t, uri)
	req := assets.NewAuthentifiedRequest(t, u, pass, "GET", uri, vars)

	//Do the request
	rec := httptest.NewRecorder()
	router.mux.ServeHTTP(rec, req)
	assert.Equal(t, 200, rec.Code)

	wr := &sdk.WorkflowRun{}
	test.NoError(t, json.Unmarshal(rec.Body.Bytes(), wr))
	assert.Equal(t, int64(10), wr.Number)

	//Test getWorkflowRunTagsHandler
	uri = router.getRoute("GET", getWorkflowRunTagsHandler, vars)
	test.NotEmpty(t, uri)
	req = assets.NewAuthentifiedRequest(t, u, pass, "GET", uri, vars)
	//Do the request
	rec = httptest.NewRecorder()
	router.mux.ServeHTTP(rec, req)
	assert.Equal(t, 200, rec.Code)

	tags := map[string][]string{}
	test.NoError(t, json.Unmarshal(rec.Body.Bytes(), &tags))
	assert.Len(t, tags, 2)
	assert.Len(t, tags["git.branch"], 1)
	assert.Len(t, tags["git.hash"], 10)

}

func Test_getWorkflowRunHandler(t *testing.T) {
	db := test.SetupPG(t, bootstrap.InitiliazeDB)
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
	test.NoError(t, pipeline.InsertPipeline(db, proj, &pip, u))

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
	test.NoError(t, pipeline.InsertPipeline(db, proj, &pip2, u))
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
				sdk.WorkflowNodeTrigger{
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

	for i := 0; i < 10; i++ {
		_, err = workflow.ManualRun(db, w1, &sdk.WorkflowNodeRunManual{
			User: *u,
		})
		test.NoError(t, err)
	}

	// Init router
	router = newRouter(auth.TestLocalAuth(t), mux.NewRouter(), "/Test_getWorkflowRunHandler")
	router.init()
	//Prepare request
	vars := map[string]string{
		"permProjectKey": proj.Key,
		"workflowName":   w1.Name,
		"number":         "9",
	}
	uri := router.getRoute("GET", getWorkflowRunHandler, vars)
	test.NotEmpty(t, uri)
	req := assets.NewAuthentifiedRequest(t, u, pass, "GET", uri, vars)

	//Do the request
	rec := httptest.NewRecorder()
	router.mux.ServeHTTP(rec, req)
	assert.Equal(t, 200, rec.Code)

	wr := &sdk.WorkflowRun{}
	test.NoError(t, json.Unmarshal(rec.Body.Bytes(), wr))
	assert.Equal(t, int64(9), wr.Number)
}

func Test_getWorkflowNodeRunHandler(t *testing.T) {
	db := test.SetupPG(t, bootstrap.InitiliazeDB)
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
	test.NoError(t, pipeline.InsertPipeline(db, proj, &pip, u))

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
	test.NoError(t, pipeline.InsertPipeline(db, proj, &pip2, u))
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
				sdk.WorkflowNodeTrigger{
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

	// Init router
	router = newRouter(auth.TestLocalAuth(t), mux.NewRouter(), "/Test_getWorkflowNodeRunHandler")
	router.init()
	//Prepare request
	vars := map[string]string{
		"permProjectKey": proj.Key,
		"workflowName":   w1.Name,
		"number":         fmt.Sprintf("%d", lastrun.Number),
		"id":             fmt.Sprintf("%d", lastrun.WorkflowNodeRuns[w1.RootID][0].ID),
	}
	uri := router.getRoute("GET", getWorkflowNodeRunHandler, vars)
	test.NotEmpty(t, uri)
	req := assets.NewAuthentifiedRequest(t, u, pass, "GET", uri, vars)

	//Do the request
	rec := httptest.NewRecorder()
	router.mux.ServeHTTP(rec, req)
	assert.Equal(t, 200, rec.Code)
}

func Test_postWorkflowRunHandler(t *testing.T) {
	db := test.SetupPG(t, bootstrap.InitiliazeDB)
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
	test.NoError(t, pipeline.InsertPipeline(db, proj, &pip, u))

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
	test.NoError(t, pipeline.InsertPipeline(db, proj, &pip2, u))
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
				sdk.WorkflowNodeTrigger{
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

	// Init router
	router = newRouter(auth.TestLocalAuth(t), mux.NewRouter(), "/Test_postWorkflowRunHandler")
	router.init()
	//Prepare request
	vars := map[string]string{
		"permProjectKey": proj.Key,
		"workflowName":   w1.Name,
	}
	uri := router.getRoute("POST", postWorkflowRunHandler, vars)
	test.NotEmpty(t, uri)

	opts := &postWorkflowRunHandlerOption{}
	req := assets.NewAuthentifiedRequest(t, u, pass, "POST", uri, opts)

	//Do the request
	rec := httptest.NewRecorder()
	router.mux.ServeHTTP(rec, req)
	assert.Equal(t, 200, rec.Code)

	wr := &sdk.WorkflowRun{}
	test.NoError(t, json.Unmarshal(rec.Body.Bytes(), wr))
	assert.Equal(t, int64(1), wr.Number)
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
	test.NoError(t, pipeline.InsertPipeline(db, proj, &pip, u))

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
	test.NoError(t, pipeline.InsertPipeline(db, proj, &pip2, u))
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
