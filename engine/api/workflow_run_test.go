package api

import (
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"testing"

	izanami "github.com/ovhlabs/izanami-go-client"
	"github.com/stretchr/testify/assert"

	"github.com/ovh/cds/engine/api/bootstrap"
	"github.com/ovh/cds/engine/api/environment"
	"github.com/ovh/cds/engine/api/feature"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/permission"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/engine/api/user"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/sdk"
)

func Test_getWorkflowNodeRunHistoryHandler(t *testing.T) {
	api, db, router := newTestAPI(t, bootstrap.InitiliazeDB)
	u, pass := assets.InsertAdminUser(db)
	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, api.Cache, key, key, u)

	//First pipeline
	pip := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip1",
		Type:       sdk.BuildPipeline,
	}
	test.NoError(t, pipeline.InsertPipeline(db, api.Cache, proj, &pip, u))

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
	test.NoError(t, pipeline.InsertPipeline(db, api.Cache, proj, &pip2, u))
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

	proj2, errP := project.Load(api.mustDB(), api.Cache, proj.Key, u, project.LoadOptions.WithPipelines, project.LoadOptions.WithGroups, project.LoadOptions.WithPlatforms)
	test.NoError(t, errP)

	test.NoError(t, workflow.Insert(db, api.Cache, &w, proj2, u))
	w1, err := workflow.Load(db, api.Cache, proj, "test_1", u, workflow.LoadOptions{})
	test.NoError(t, err)

	wr, _, errMR := workflow.ManualRun(nil, db, db, api.Cache, proj, w1, &sdk.WorkflowNodeRunManual{
		User: *u,
	}, nil)
	if errMR != nil {
		test.NoError(t, errMR)
	}

	_, _, errMR2 := workflow.ManualRunFromNode(nil, db, db, api.Cache, proj, &wr.Workflow, wr.Number, &sdk.WorkflowNodeRunManual{User: *u}, wr.Workflow.RootID)
	if errMR2 != nil {
		test.NoError(t, errMR2)
	}

	//Prepare request
	vars := map[string]string{
		"key":              proj.Key,
		"permWorkflowName": w1.Name,
		"number":           fmt.Sprintf("%d", wr.Number),
		"nodeID":           fmt.Sprintf("%d", wr.Workflow.RootID),
	}
	uri := router.GetRoute("GET", api.getWorkflowNodeRunHistoryHandler, vars)
	test.NotEmpty(t, uri)
	req := assets.NewAuthentifiedRequest(t, u, pass, "GET", uri, vars)

	//Do the request
	rec := httptest.NewRecorder()
	router.Mux.ServeHTTP(rec, req)
	assert.Equal(t, 200, rec.Code)

	history := []sdk.WorkflowNodeRun{}
	test.NoError(t, json.Unmarshal(rec.Body.Bytes(), &history))
	assert.Equal(t, 2, len(history))
	assert.Equal(t, int64(1), history[0].SubNumber)
	assert.Equal(t, int64(0), history[1].SubNumber)
}

func Test_getWorkflowRunsHandler(t *testing.T) {
	api, db, router := newTestAPI(t, bootstrap.InitiliazeDB)
	u, pass := assets.InsertAdminUser(api.mustDB())
	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, api.Cache, key, key, u)

	//First pipeline
	pip := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip1",
		Type:       sdk.BuildPipeline,
	}
	test.NoError(t, pipeline.InsertPipeline(api.mustDB(), api.Cache, proj, &pip, u))

	s := sdk.NewStage("stage 1")
	s.Enabled = true
	s.PipelineID = pip.ID
	pipeline.InsertStage(api.mustDB(), s)
	j := &sdk.Job{
		Enabled: true,
		Action: sdk.Action{
			Enabled: true,
		},
	}
	pipeline.InsertJob(api.mustDB(), j, s.ID, &pip)
	s.Jobs = append(s.Jobs, *j)

	pip.Stages = append(pip.Stages, *s)

	//Second pipeline
	pip2 := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip2",
		Type:       sdk.BuildPipeline,
	}
	test.NoError(t, pipeline.InsertPipeline(api.mustDB(), api.Cache, proj, &pip2, u))
	s = sdk.NewStage("stage 1")
	s.Enabled = true
	s.PipelineID = pip2.ID
	pipeline.InsertStage(api.mustDB(), s)
	j = &sdk.Job{
		Enabled: true,
		Action: sdk.Action{
			Enabled: true,
		},
	}
	pipeline.InsertJob(api.mustDB(), j, s.ID, &pip2)
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

	proj2, errP := project.Load(api.mustDB(), api.Cache, proj.Key, u, project.LoadOptions.WithPipelines, project.LoadOptions.WithGroups, project.LoadOptions.WithPlatforms)
	test.NoError(t, errP)

	test.NoError(t, workflow.Insert(api.mustDB(), api.Cache, &w, proj2, u))
	w1, err := workflow.Load(api.mustDB(), api.Cache, proj, "test_1", u, workflow.LoadOptions{})
	test.NoError(t, err)

	for i := 0; i < 10; i++ {
		_, _, err = workflow.ManualRun(nil, api.mustDB(), api.mustDB(), api.Cache, proj, w1, &sdk.WorkflowNodeRunManual{
			User: *u,
		}, nil)
		test.NoError(t, err)
	}

	//Prepare request
	vars := map[string]string{
		"key":              proj.Key,
		"permWorkflowName": w1.Name,
	}
	uri := router.GetRoute("GET", api.getWorkflowRunsHandler, vars)
	test.NotEmpty(t, uri)
	req := assets.NewAuthentifiedRequest(t, u, pass, "GET", uri, vars)

	//Do the request
	rec := httptest.NewRecorder()
	router.Mux.ServeHTTP(rec, req)
	assert.Equal(t, 200, rec.Code)
	assert.Equal(t, "0-10/10", rec.Header().Get("Content-Range"))

	//Prepare request
	vars = map[string]string{
		"permProjectKey": proj.Key,
	}
	uri = router.GetRoute("GET", api.getWorkflowAllRunsHandler, vars)
	test.NotEmpty(t, uri)
	req = assets.NewAuthentifiedRequest(t, u, pass, "GET", uri, vars)

	//Do the request
	rec = httptest.NewRecorder()
	router.Mux.ServeHTTP(rec, req)
	assert.Equal(t, 200, rec.Code)
	assert.Equal(t, "0-10/10", rec.Header().Get("Content-Range"))

	//Prepare request
	vars = map[string]string{
		"key":              proj.Key,
		"permWorkflowName": w1.Name,
	}
	uri = router.GetRoute("GET", api.getWorkflowRunsHandler, vars)
	test.NotEmpty(t, uri)
	req = assets.NewAuthentifiedRequest(t, u, pass, "GET", uri, vars)
	q := req.URL.Query()
	q.Set("offset", "5")
	q.Set("limit", "9")
	req.URL.RawQuery = q.Encode()
	//Do the request
	rec = httptest.NewRecorder()
	router.Mux.ServeHTTP(rec, req)
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
	router.Mux.ServeHTTP(rec, req)
	assert.Equal(t, 400, rec.Code)
	assert.Equal(t, "", rec.Header().Get("Content-Range"))

}

func Test_getWorkflowRunsHandlerWithFilter(t *testing.T) {
	api, db, router := newTestAPI(t, bootstrap.InitiliazeDB)
	u, pass := assets.InsertAdminUser(api.mustDB())
	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, api.Cache, key, key, u)

	//First pipeline
	pip := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip1",
		Type:       sdk.BuildPipeline,
	}
	test.NoError(t, pipeline.InsertPipeline(api.mustDB(), api.Cache, proj, &pip, u))

	s := sdk.NewStage("stage 1")
	s.Enabled = true
	s.PipelineID = pip.ID
	pipeline.InsertStage(api.mustDB(), s)
	j := &sdk.Job{
		Enabled: true,
		Action: sdk.Action{
			Enabled: true,
		},
	}
	pipeline.InsertJob(api.mustDB(), j, s.ID, &pip)
	s.Jobs = append(s.Jobs, *j)

	pip.Stages = append(pip.Stages, *s)

	//Second pipeline
	pip2 := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip2",
		Type:       sdk.BuildPipeline,
	}
	test.NoError(t, pipeline.InsertPipeline(api.mustDB(), api.Cache, proj, &pip2, u))
	s = sdk.NewStage("stage 1")
	s.Enabled = true
	s.PipelineID = pip2.ID
	pipeline.InsertStage(api.mustDB(), s)
	j = &sdk.Job{
		Enabled: true,
		Action: sdk.Action{
			Enabled: true,
		},
	}
	pipeline.InsertJob(api.mustDB(), j, s.ID, &pip2)
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

	proj2, errP := project.Load(api.mustDB(), api.Cache, proj.Key, u, project.LoadOptions.WithPipelines, project.LoadOptions.WithGroups, project.LoadOptions.WithPlatforms)
	test.NoError(t, errP)

	test.NoError(t, workflow.Insert(api.mustDB(), api.Cache, &w, proj2, u))
	w1, err := workflow.Load(api.mustDB(), api.Cache, proj, "test_1", u, workflow.LoadOptions{})
	test.NoError(t, err)

	_, _, err = workflow.ManualRun(nil, api.mustDB(), api.mustDB(), api.Cache, proj, w1, &sdk.WorkflowNodeRunManual{
		User: *u,
	}, nil)
	test.NoError(t, err)

	//Prepare request
	vars := map[string]string{
		"key":              proj.Key,
		"permWorkflowName": w1.Name,
	}
	uri := router.GetRoute("GET", api.getWorkflowRunsHandler, vars)
	test.NotEmpty(t, uri)
	req := assets.NewAuthentifiedRequest(t, u, pass, "GET", uri, vars)
	q := req.URL.Query()
	q.Set("triggered_by", u.Username)
	req.URL.RawQuery = q.Encode()

	//Do the request
	rec := httptest.NewRecorder()
	router.Mux.ServeHTTP(rec, req)
	assert.Equal(t, 200, rec.Code)
	assert.Equal(t, "0-10/1", rec.Header().Get("Content-Range"))

	t.Log(rec.Body.String())
}

func Test_getLatestWorkflowRunHandler(t *testing.T) {
	api, db, router := newTestAPI(t, bootstrap.InitiliazeDB)
	u, pass := assets.InsertAdminUser(api.mustDB())
	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, api.Cache, key, key, u)

	//First pipeline
	pip := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip1",
		Type:       sdk.BuildPipeline,
	}
	test.NoError(t, pipeline.InsertPipeline(api.mustDB(), api.Cache, proj, &pip, u))

	s := sdk.NewStage("stage 1")
	s.Enabled = true
	s.PipelineID = pip.ID
	pipeline.InsertStage(api.mustDB(), s)
	j := &sdk.Job{
		Enabled: true,
		Action: sdk.Action{
			Enabled: true,
		},
	}
	pipeline.InsertJob(api.mustDB(), j, s.ID, &pip)
	s.Jobs = append(s.Jobs, *j)

	pip.Stages = append(pip.Stages, *s)

	//Second pipeline
	pip2 := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip2",
		Type:       sdk.BuildPipeline,
	}
	test.NoError(t, pipeline.InsertPipeline(api.mustDB(), api.Cache, proj, &pip2, u))
	s = sdk.NewStage("stage 1")
	s.Enabled = true
	s.PipelineID = pip2.ID
	pipeline.InsertStage(api.mustDB(), s)
	j = &sdk.Job{
		Enabled: true,
		Action: sdk.Action{
			Enabled: true,
		},
	}
	pipeline.InsertJob(api.mustDB(), j, s.ID, &pip2)
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

	proj2, errP := project.Load(api.mustDB(), api.Cache, proj.Key, u, project.LoadOptions.WithPipelines, project.LoadOptions.WithGroups, project.LoadOptions.WithPlatforms)
	test.NoError(t, errP)

	test.NoError(t, workflow.Insert(api.mustDB(), api.Cache, &w, proj2, u))
	w1, err := workflow.Load(api.mustDB(), api.Cache, proj, "test_1", u, workflow.LoadOptions{})
	test.NoError(t, err)

	for i := 0; i < 10; i++ {
		_, _, err = workflow.ManualRun(nil, api.mustDB(), api.mustDB(), api.Cache, proj, w1, &sdk.WorkflowNodeRunManual{
			User: *u,
			Payload: map[string]string{
				"git.branch": "master",
				"git.hash":   fmt.Sprintf("%d", i),
			},
		}, nil)
		test.NoError(t, err)
	}

	//Prepare request
	vars := map[string]string{
		"key":              proj.Key,
		"permWorkflowName": w1.Name,
	}
	uri := router.GetRoute("GET", api.getLatestWorkflowRunHandler, vars)
	test.NotEmpty(t, uri)
	req := assets.NewAuthentifiedRequest(t, u, pass, "GET", uri, vars)

	//Do the request
	rec := httptest.NewRecorder()
	router.Mux.ServeHTTP(rec, req)
	assert.Equal(t, 200, rec.Code)

	wr := &sdk.WorkflowRun{}
	test.NoError(t, json.Unmarshal(rec.Body.Bytes(), wr))
	assert.Equal(t, int64(10), wr.Number)

	//Test getWorkflowRunTagsHandler
	uri = router.GetRoute("GET", api.getWorkflowRunTagsHandler, vars)
	test.NotEmpty(t, uri)
	req = assets.NewAuthentifiedRequest(t, u, pass, "GET", uri, vars)
	//Do the request
	rec = httptest.NewRecorder()
	router.Mux.ServeHTTP(rec, req)
	assert.Equal(t, 200, rec.Code)

	tags := map[string][]string{}
	test.NoError(t, json.Unmarshal(rec.Body.Bytes(), &tags))
	assert.Len(t, tags, 3)
	assert.Len(t, tags["git.branch"], 1)
	assert.Len(t, tags["git.hash"], 10)

}

func Test_getWorkflowRunHandler(t *testing.T) {
	api, db, router := newTestAPI(t, bootstrap.InitiliazeDB)
	u, pass := assets.InsertAdminUser(api.mustDB())
	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, api.Cache, key, key, u)

	//First pipeline
	pip := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip1",
		Type:       sdk.BuildPipeline,
	}
	test.NoError(t, pipeline.InsertPipeline(api.mustDB(), api.Cache, proj, &pip, u))

	s := sdk.NewStage("stage 1")
	s.Enabled = true
	s.PipelineID = pip.ID
	pipeline.InsertStage(api.mustDB(), s)
	j := &sdk.Job{
		Enabled: true,
		Action: sdk.Action{
			Enabled: true,
		},
	}
	pipeline.InsertJob(api.mustDB(), j, s.ID, &pip)
	s.Jobs = append(s.Jobs, *j)

	pip.Stages = append(pip.Stages, *s)

	//Second pipeline
	pip2 := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip2",
		Type:       sdk.BuildPipeline,
	}
	test.NoError(t, pipeline.InsertPipeline(api.mustDB(), api.Cache, proj, &pip2, u))
	s = sdk.NewStage("stage 1")
	s.Enabled = true
	s.PipelineID = pip2.ID
	pipeline.InsertStage(api.mustDB(), s)
	j = &sdk.Job{
		Enabled: true,
		Action: sdk.Action{
			Enabled: true,
		},
	}
	pipeline.InsertJob(api.mustDB(), j, s.ID, &pip2)
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

	proj2, errP := project.Load(api.mustDB(), api.Cache, proj.Key, u, project.LoadOptions.WithPipelines, project.LoadOptions.WithGroups, project.LoadOptions.WithPlatforms)
	test.NoError(t, errP)

	test.NoError(t, workflow.Insert(api.mustDB(), api.Cache, &w, proj2, u))
	w1, err := workflow.Load(api.mustDB(), api.Cache, proj, "test_1", u, workflow.LoadOptions{})
	test.NoError(t, err)

	for i := 0; i < 10; i++ {
		_, _, err = workflow.ManualRun(nil, api.mustDB(), api.mustDB(), api.Cache, proj, w1, &sdk.WorkflowNodeRunManual{
			User: *u,
		}, nil)
		test.NoError(t, err)
	}

	//Prepare request
	vars := map[string]string{
		"key":              proj.Key,
		"permWorkflowName": w1.Name,
		"number":           "9",
	}
	uri := router.GetRoute("GET", api.getWorkflowRunHandler, vars)
	test.NotEmpty(t, uri)
	req := assets.NewAuthentifiedRequest(t, u, pass, "GET", uri, vars)

	//Do the request
	rec := httptest.NewRecorder()
	router.Mux.ServeHTTP(rec, req)
	assert.Equal(t, 200, rec.Code)

	wr := &sdk.WorkflowRun{}
	test.NoError(t, json.Unmarshal(rec.Body.Bytes(), wr))
	assert.Equal(t, int64(9), wr.Number)
}

func Test_getWorkflowNodeRunHandler(t *testing.T) {
	api, db, router := newTestAPI(t, bootstrap.InitiliazeDB)
	u, pass := assets.InsertAdminUser(api.mustDB())
	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, api.Cache, key, key, u)

	//First pipeline
	pip := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip1",
		Type:       sdk.BuildPipeline,
	}
	test.NoError(t, pipeline.InsertPipeline(api.mustDB(), api.Cache, proj, &pip, u))

	s := sdk.NewStage("stage 1")
	s.Enabled = true
	s.PipelineID = pip.ID
	pipeline.InsertStage(api.mustDB(), s)
	j := &sdk.Job{
		Enabled: true,
		Action: sdk.Action{
			Enabled: true,
		},
	}
	pipeline.InsertJob(api.mustDB(), j, s.ID, &pip)
	s.Jobs = append(s.Jobs, *j)

	pip.Stages = append(pip.Stages, *s)

	//Second pipeline
	pip2 := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip2",
		Type:       sdk.BuildPipeline,
	}
	test.NoError(t, pipeline.InsertPipeline(api.mustDB(), api.Cache, proj, &pip2, u))
	s = sdk.NewStage("stage 1")
	s.Enabled = true
	s.PipelineID = pip2.ID
	pipeline.InsertStage(api.mustDB(), s)
	j = &sdk.Job{
		Enabled: true,
		Action: sdk.Action{
			Enabled: true,
		},
	}
	pipeline.InsertJob(api.mustDB(), j, s.ID, &pip2)
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

	proj2, errP := project.Load(api.mustDB(), api.Cache, proj.Key, u, project.LoadOptions.WithPipelines, project.LoadOptions.WithGroups, project.LoadOptions.WithPlatforms)
	test.NoError(t, errP)

	test.NoError(t, workflow.Insert(api.mustDB(), api.Cache, &w, proj2, u))
	w1, err := workflow.Load(api.mustDB(), api.Cache, proj, "test_1", u, workflow.LoadOptions{})
	test.NoError(t, err)

	_, _, err = workflow.ManualRun(nil, api.mustDB(), api.mustDB(), api.Cache, proj, w1, &sdk.WorkflowNodeRunManual{
		User: *u,
	}, nil)
	test.NoError(t, err)

	lastrun, err := workflow.LoadLastRun(api.mustDB(), proj.Key, w1.Name, workflow.LoadRunOptions{WithArtifacts: true, WithTests: true})
	test.NoError(t, err)

	//Prepare request
	vars := map[string]string{
		"key":              proj.Key,
		"permWorkflowName": w1.Name,
		"number":           fmt.Sprintf("%d", lastrun.Number),
		"nodeRunID":        fmt.Sprintf("%d", lastrun.WorkflowNodeRuns[w1.RootID][0].ID),
	}
	uri := router.GetRoute("GET", api.getWorkflowNodeRunHandler, vars)
	test.NotEmpty(t, uri)
	req := assets.NewAuthentifiedRequest(t, u, pass, "GET", uri, vars)

	//Do the request
	rec := httptest.NewRecorder()
	router.Mux.ServeHTTP(rec, req)
	assert.Equal(t, 200, rec.Code)
}

func Test_resyncWorkflowRunHandler(t *testing.T) {
	api, db, router := newTestAPI(t, bootstrap.InitiliazeDB)
	u, pass := assets.InsertAdminUser(db)
	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, api.Cache, key, key, u)

	//First pipeline
	pip := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip1",
		Type:       sdk.BuildPipeline,
	}
	test.NoError(t, pipeline.InsertPipeline(db, api.Cache, proj, &pip, u))

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

	pipeline.InsertStage(db, s)
	j = &sdk.Job{
		Enabled: true,
		Action: sdk.Action{
			Enabled: true,
		},
	}
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

	proj2, errP := project.Load(api.mustDB(), api.Cache, proj.Key, u, project.LoadOptions.WithPipelines, project.LoadOptions.WithGroups, project.LoadOptions.WithPlatforms)
	test.NoError(t, errP)

	test.NoError(t, workflow.Insert(db, api.Cache, &w, proj2, u))
	w1, err := workflow.Load(db, api.Cache, proj2, "test_1", u, workflow.LoadOptions{})
	test.NoError(t, err)

	//Prepare request
	vars := map[string]string{
		"key":              proj.Key,
		"permWorkflowName": w1.Name,
	}
	uri := router.GetRoute("POST", api.postWorkflowRunHandler, vars)
	test.NotEmpty(t, uri)

	opts := &sdk.WorkflowRunPostHandlerOption{}
	req := assets.NewAuthentifiedRequest(t, u, pass, "POST", uri, opts)

	//Do the request
	rec := httptest.NewRecorder()
	router.Mux.ServeHTTP(rec, req)
	assert.Equal(t, 202, rec.Code)

	wr := &sdk.WorkflowRun{}
	test.NoError(t, json.Unmarshal(rec.Body.Bytes(), wr))
	assert.Equal(t, int64(1), wr.Number)
	assert.Equal(t, "stage 1", wr.Workflow.Root.Pipeline.Stages[0].Name)

	pip.Stages[0].Name = "New awesome stage"
	errS := pipeline.UpdateStage(db, &pip.Stages[0])
	test.NoError(t, errS)

	//Prepare request
	vars = map[string]string{
		"key":              proj.Key,
		"permWorkflowName": w1.Name,
		"number":           fmt.Sprintf("%d", wr.Number),
	}
	uri = router.GetRoute("POST", api.resyncWorkflowRunHandler, vars)
	test.NotEmpty(t, uri)
	req = assets.NewAuthentifiedRequest(t, u, pass, "POST", uri, nil)

	//Do the request
	rec = httptest.NewRecorder()
	router.Mux.ServeHTTP(rec, req)
	assert.Equal(t, 200, rec.Code)

	workflowRun, errWR := workflow.LoadRunByID(db, wr.ID, workflow.LoadRunOptions{WithArtifacts: true})
	test.NoError(t, errWR)

	assert.Equal(t, "New awesome stage", workflowRun.Workflow.Root.Pipeline.Stages[0].Name)
}

func Test_postWorkflowRunHandler(t *testing.T) {
	api, db, router := newTestAPI(t, bootstrap.InitiliazeDB)
	u, pass := assets.InsertAdminUser(api.mustDB())
	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, api.Cache, key, key, u)

	//First pipeline
	pip := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip1",
		Type:       sdk.BuildPipeline,
	}
	test.NoError(t, pipeline.InsertPipeline(api.mustDB(), api.Cache, proj, &pip, u))

	s := sdk.NewStage("stage 1")
	s.Enabled = true
	s.PipelineID = pip.ID
	pipeline.InsertStage(api.mustDB(), s)
	j := &sdk.Job{
		Enabled: true,
		Action: sdk.Action{
			Enabled: true,
		},
	}
	pipeline.InsertJob(api.mustDB(), j, s.ID, &pip)
	s.Jobs = append(s.Jobs, *j)

	pip.Stages = append(pip.Stages, *s)

	//Second pipeline
	pip2 := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip2",
		Type:       sdk.BuildPipeline,
	}
	test.NoError(t, pipeline.InsertPipeline(api.mustDB(), api.Cache, proj, &pip2, u))
	s = sdk.NewStage("stage 1")
	s.Enabled = true
	s.PipelineID = pip2.ID
	pipeline.InsertStage(api.mustDB(), s)
	j = &sdk.Job{
		Enabled: true,
		Action: sdk.Action{
			Enabled: true,
		},
	}
	pipeline.InsertJob(api.mustDB(), j, s.ID, &pip2)
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

	proj2, errP := project.Load(api.mustDB(), api.Cache, proj.Key, u, project.LoadOptions.WithPipelines, project.LoadOptions.WithGroups, project.LoadOptions.WithPlatforms)
	test.NoError(t, errP)

	test.NoError(t, workflow.Insert(api.mustDB(), api.Cache, &w, proj2, u))
	w1, err := workflow.Load(api.mustDB(), api.Cache, proj, "test_1", u, workflow.LoadOptions{})
	test.NoError(t, err)

	//Prepare request
	vars := map[string]string{
		"key":              proj.Key,
		"permWorkflowName": w1.Name,
	}
	uri := router.GetRoute("POST", api.postWorkflowRunHandler, vars)
	test.NotEmpty(t, uri)

	opts := &sdk.WorkflowRunPostHandlerOption{}
	req := assets.NewAuthentifiedRequest(t, u, pass, "POST", uri, opts)

	//Do the request
	rec := httptest.NewRecorder()
	router.Mux.ServeHTTP(rec, req)
	assert.Equal(t, 202, rec.Code)

	wr := &sdk.WorkflowRun{}
	test.NoError(t, json.Unmarshal(rec.Body.Bytes(), wr))
	assert.Equal(t, int64(1), wr.Number)
}

func Test_postWorkflowRunHandlerWithoutRightOnEnvironment(t *testing.T) {
	api, db, router := newTestAPI(t, bootstrap.InitiliazeDB)
	u, _ := assets.InsertAdminUser(api.mustDB())
	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, api.Cache, key, key, u)

	//First pipeline
	pip := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip1",
		Type:       sdk.BuildPipeline,
	}
	test.NoError(t, pipeline.InsertPipeline(api.mustDB(), api.Cache, proj, &pip, u))

	s := sdk.NewStage("stage 1")
	s.Enabled = true
	s.PipelineID = pip.ID
	pipeline.InsertStage(api.mustDB(), s)
	j := &sdk.Job{
		Enabled: true,
		Action: sdk.Action{
			Enabled: true,
		},
	}
	pipeline.InsertJob(api.mustDB(), j, s.ID, &pip)
	s.Jobs = append(s.Jobs, *j)

	pip.Stages = append(pip.Stages, *s)

	//Second pipeline
	pip2 := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip2",
		Type:       sdk.BuildPipeline,
	}
	test.NoError(t, pipeline.InsertPipeline(api.mustDB(), api.Cache, proj, &pip2, u))
	s = sdk.NewStage("stage 1")
	s.Enabled = true
	s.PipelineID = pip2.ID
	pipeline.InsertStage(api.mustDB(), s)
	j = &sdk.Job{
		Enabled: true,
		Action: sdk.Action{
			Enabled: true,
		},
	}
	pipeline.InsertJob(api.mustDB(), j, s.ID, &pip2)
	s.Jobs = append(s.Jobs, *j)
	env := sdk.Environment{
		Name:       "envtest",
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
	}
	test.NoError(t, environment.InsertEnvironment(api.mustDB(), &env))
	gr := sdk.Group{
		Name: sdk.RandomString(10),
	}
	_, _, errG := group.AddGroup(api.mustDB(), &gr)
	test.NoError(t, errG)

	uLambda, pass := assets.InsertLambdaUser(api.mustDB(), &gr)

	grs := []sdk.GroupPermission{
		{Group: gr, Permission: permission.PermissionRead},
	}
	test.NoError(t, group.InsertGroupsInEnvironment(db, grs, env.ID))

	w := sdk.Workflow{
		Name:       "test_1",
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Root: &sdk.WorkflowNode{
			Pipeline: pip,
			Context: &sdk.WorkflowNodeContext{
				EnvironmentID: env.ID,
				Environment:   &env,
			},
			Triggers: []sdk.WorkflowNodeTrigger{
				sdk.WorkflowNodeTrigger{
					WorkflowDestNode: sdk.WorkflowNode{
						Pipeline: pip,
					},
				},
			},
		},
	}

	proj2, errP := project.Load(api.mustDB(), api.Cache, proj.Key, u, project.LoadOptions.WithPipelines, project.LoadOptions.WithGroups, project.LoadOptions.WithEnvironments)
	test.NoError(t, errP)

	test.NoError(t, workflow.Insert(api.mustDB(), api.Cache, &w, proj2, u))
	w1, err := workflow.Load(api.mustDB(), api.Cache, proj2, "test_1", u, workflow.LoadOptions{})
	test.NoError(t, err)

	//Prepare request
	vars := map[string]string{
		"key":              proj.Key,
		"permWorkflowName": w1.Name,
	}
	uri := router.GetRoute("POST", api.postWorkflowRunHandler, vars)
	test.NotEmpty(t, uri)

	opts := &sdk.WorkflowRunPostHandlerOption{}
	req := assets.NewAuthentifiedRequest(t, uLambda, pass, "POST", uri, opts)

	//Do the request
	rec := httptest.NewRecorder()
	router.Mux.ServeHTTP(rec, req)
	assert.Equal(t, 403, rec.Code)
}

func Test_postWorkflowAsCodeRunDisabledHandler(t *testing.T) {
	api, db, router := newTestAPI(t, bootstrap.InitiliazeDB)
	u, pass := assets.InsertAdminUser(api.mustDB())
	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, api.Cache, key, key, u)

	c, _ := izanami.New("", "clientID", "secret")
	feature.SetClient(c)

	api.Cache.Set("feature:"+proj.Key, feature.ProjectFeatures{
		Key: proj.Key,
		Features: map[string]bool{
			feature.FeatWorkflowAsCode: false,
		},
	})

	//First pipeline
	pip := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip1",
		Type:       sdk.BuildPipeline,
	}
	test.NoError(t, pipeline.InsertPipeline(api.mustDB(), api.Cache, proj, &pip, u))

	w := sdk.Workflow{
		Name:       "test_1",
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Root: &sdk.WorkflowNode{
			Pipeline: pip,
		},
		FromRepository: "ovh/cds",
	}

	proj2, errP := project.Load(api.mustDB(), api.Cache, proj.Key, u, project.LoadOptions.WithPipelines, project.LoadOptions.WithGroups, project.LoadOptions.WithPlatforms)
	test.NoError(t, errP)

	test.NoError(t, workflow.Insert(api.mustDB(), api.Cache, &w, proj2, u))
	w1, err := workflow.Load(api.mustDB(), api.Cache, proj, "test_1", u, workflow.LoadOptions{})
	test.NoError(t, err)
	t.Logf(">>>>>>>>%s", w1.FromRepository)

	//Prepare request
	vars := map[string]string{
		"key":              proj.Key,
		"permWorkflowName": w1.Name,
	}
	uri := router.GetRoute("POST", api.postWorkflowRunHandler, vars)
	test.NotEmpty(t, uri)

	opts := &sdk.WorkflowRunPostHandlerOption{}
	req := assets.NewAuthentifiedRequest(t, u, pass, "POST", uri, opts)

	//Do the request
	rec := httptest.NewRecorder()
	router.Mux.ServeHTTP(rec, req)
	assert.Equal(t, 403, rec.Code)
}

func Test_postWorkflowRunHandler_Forbidden(t *testing.T) {
	api, db, router := newTestAPI(t, bootstrap.InitiliazeDB)
	u, pass := assets.InsertAdminUser(api.mustDB())
	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, api.Cache, key, key, u)

	gr := &sdk.Group{
		Name: sdk.RandomString(10),
	}
	test.NoError(t, group.InsertGroup(db, gr))
	test.NoError(t, group.InsertGroupInProject(api.mustDB(), proj.ID, gr.ID, 7))

	//First pipeline
	pip := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip1",
		Type:       sdk.BuildPipeline,
	}
	test.NoError(t, pipeline.InsertPipeline(api.mustDB(), api.Cache, proj, &pip, u))

	env := &sdk.Environment{
		Name:       sdk.RandomString(10),
		ProjectKey: proj.Key,
		ProjectID:  proj.ID,
	}
	test.NoError(t, environment.InsertEnvironment(api.mustDB(), env))

	group.InsertGroupInEnvironment(db, env.ID, proj.ProjectGroups[0].Group.ID, 4)
	group.InsertGroupInEnvironment(db, env.ID, gr.ID, 4)

	proj2, errp := project.Load(api.mustDB(), api.Cache, proj.Key, u, project.LoadOptions.WithPipelines, project.LoadOptions.WithEnvironments)
	test.NoError(t, errp)

	w := sdk.Workflow{
		Name:       "test_1",
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Root: &sdk.WorkflowNode{
			Pipeline: pip,
			Context: &sdk.WorkflowNodeContext{
				Environment: env,
			},
		},
	}

	test.NoError(t, workflow.Insert(api.mustDB(), api.Cache, &w, proj2, u))

	u.Admin = false
	test.NoError(t, user.UpdateUser(api.mustDB(), *u))

	//Prepare request
	vars := map[string]string{
		"key":              proj.Key,
		"permWorkflowName": w.Name,
	}
	uri := router.GetRoute("POST", api.postWorkflowRunHandler, vars)
	test.NotEmpty(t, uri)

	opts := &sdk.WorkflowRunPostHandlerOption{}
	req := assets.NewAuthentifiedRequest(t, u, pass, "POST", uri, opts)

	//Do the request
	rec := httptest.NewRecorder()
	router.Mux.ServeHTTP(rec, req)
	assert.Equal(t, 403, rec.Code)
}

func Test_getWorkflowNodeRunJobStepHandler(t *testing.T) {
	api, db, router := newTestAPI(t)
	u, pass := assets.InsertAdminUser(api.mustDB())
	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, api.Cache, key, key, u)

	//First pipeline
	pip := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip1",
		Type:       sdk.BuildPipeline,
	}
	test.NoError(t, pipeline.InsertPipeline(api.mustDB(), api.Cache, proj, &pip, u))

	s := sdk.NewStage("stage 1")
	s.Enabled = true
	s.PipelineID = pip.ID
	pipeline.InsertStage(api.mustDB(), s)
	j := &sdk.Job{
		Enabled: true,
		Action: sdk.Action{
			Enabled: true,
		},
	}
	pipeline.InsertJob(api.mustDB(), j, s.ID, &pip)
	s.Jobs = append(s.Jobs, *j)

	pip.Stages = append(pip.Stages, *s)

	//Second pipeline
	pip2 := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip2",
		Type:       sdk.BuildPipeline,
	}
	test.NoError(t, pipeline.InsertPipeline(api.mustDB(), api.Cache, proj, &pip2, u))
	s = sdk.NewStage("stage 1")
	s.Enabled = true
	s.PipelineID = pip2.ID
	pipeline.InsertStage(api.mustDB(), s)
	j = &sdk.Job{
		Enabled: true,
		Action: sdk.Action{
			Enabled: true,
		},
	}
	pipeline.InsertJob(api.mustDB(), j, s.ID, &pip2)
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

	proj2, errP := project.Load(api.mustDB(), api.Cache, proj.Key, u, project.LoadOptions.WithPipelines, project.LoadOptions.WithGroups, project.LoadOptions.WithPlatforms)
	test.NoError(t, errP)

	test.NoError(t, workflow.Insert(api.mustDB(), api.Cache, &w, proj2, u))
	w1, err := workflow.Load(api.mustDB(), api.Cache, proj, "test_1", u, workflow.LoadOptions{
		DeepPipeline: true,
	})
	test.NoError(t, err)

	_, _, err = workflow.ManualRun(nil, api.mustDB(), api.mustDB(), api.Cache, proj, w1, &sdk.WorkflowNodeRunManual{
		User: *u,
	}, nil)
	test.NoError(t, err)

	lastrun, err := workflow.LoadLastRun(api.mustDB(), proj.Key, w1.Name, workflow.LoadRunOptions{WithArtifacts: true})
	test.NoError(t, err)

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
	errUJ := workflow.UpdateNodeRun(api.mustDB(), &lastrun.WorkflowNodeRuns[w1.RootID][0])
	test.NoError(t, errUJ)

	// Add log
	errAL := workflow.AddLog(api.mustDB(), jobRun, log)
	test.NoError(t, errAL)

	//Prepare request
	vars := map[string]string{
		"key":              proj.Key,
		"permWorkflowName": w1.Name,
		"number":           fmt.Sprintf("%d", lastrun.Number),
		"nodeRunID":        fmt.Sprintf("%d", lastrun.WorkflowNodeRuns[w1.RootID][0].ID),
		"runJobId":         fmt.Sprintf("%d", jobRun.ID),
		"stepOrder":        "1",
	}
	uri := router.GetRoute("GET", api.getWorkflowNodeRunJobStepHandler, vars)
	test.NotEmpty(t, uri)
	req := assets.NewAuthentifiedRequest(t, u, pass, "GET", uri, vars)

	//Do the request
	rec := httptest.NewRecorder()
	router.Mux.ServeHTTP(rec, req)

	stepState := &sdk.BuildState{}
	json.Unmarshal(rec.Body.Bytes(), stepState)
	assert.Equal(t, 200, rec.Code)
	assert.Equal(t, "My Log", stepState.StepLogs.Val)
	assert.Equal(t, sdk.StatusBuilding, stepState.Status)
}
