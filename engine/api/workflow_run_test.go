package main

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/ovh/cds/engine/api/auth"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/sdk"
	"github.com/stretchr/testify/assert"
)

func Test_getWorkflowRunsHandler(t *testing.T) {
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
	router = &Router{auth.TestLocalAuth(t), mux.NewRouter(), "/Test_getWorkflowRunsHandler"}
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
	q.Set("limit", "10")
	req.URL.RawQuery = q.Encode()
	//Do the request
	rec = httptest.NewRecorder()
	router.mux.ServeHTTP(rec, req)
	assert.Equal(t, 206, rec.Code)
	assert.Equal(t, "5-10/10", rec.Header().Get("Content-Range"))

	uri = router.getRoute("GET", getWorkflowRunsHandler, vars)
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
	router = &Router{auth.TestLocalAuth(t), mux.NewRouter(), "/Test_getLatestWorkflowRunHandler"}
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
}

func Test_getWorkflowRunHandler(t *testing.T) {

}

func Test_getWorkflowNodeRunHandler(t *testing.T) {

}
