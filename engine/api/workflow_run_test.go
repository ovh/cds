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
	"github.com/stretchr/testify/require"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/authentication"
	"github.com/ovh/cds/engine/api/bootstrap"
	"github.com/ovh/cds/engine/api/environment"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/repositoriesmanager"
	"github.com/ovh/cds/engine/api/services"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/engine/api/user"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/sdk"
)

func Test_getWorkflowNodeRunHistoryHandler(t *testing.T) {
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
	require.NoError(t, pipeline.InsertPipeline(db, api.Cache, proj, &pip))

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
	}
	require.NoError(t, pipeline.InsertPipeline(db, api.Cache, proj, &pip2))
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
	require.NoError(t, errP)

	require.NoError(t, workflow.Insert(context.TODO(), db, api.Cache, &w, proj2))
	w1, err := workflow.Load(context.TODO(), db, api.Cache, proj, "test_1", workflow.LoadOptions{})
	require.NoError(t, err)

	wrCreate, err := workflow.CreateRun(db, w1, nil, u)
	assert.NoError(t, err)
	wrCreate.Workflow = *w1
	_, errMR := workflow.StartWorkflowRun(context.TODO(), db, api.Cache, proj, wrCreate, &sdk.WorkflowRunPostHandlerOption{
		Manual: &sdk.WorkflowNodeRunManual{
			Username: u.GetUsername(),
		},
	}, consumer, nil)
	if errMR != nil {
		require.NoError(t, errMR)
	}

	_, errMR2 := workflow.StartWorkflowRun(context.TODO(), db, api.Cache, proj, wrCreate, &sdk.WorkflowRunPostHandlerOption{
		Manual: &sdk.WorkflowNodeRunManual{
			Username: u.GetUsername(),
		},
		FromNodeIDs: []int64{wrCreate.Workflow.WorkflowData.Node.ID},
	}, consumer, nil)
	assert.NoError(t, errMR2)

	//Prepare request
	vars := map[string]string{
		"key":              proj.Key,
		"permWorkflowName": w1.Name,
		"number":           fmt.Sprintf("%d", wrCreate.Number),
		"nodeID":           fmt.Sprintf("%d", wrCreate.Workflow.WorkflowData.Node.ID),
	}
	uri := router.GetRoute("GET", api.getWorkflowNodeRunHistoryHandler, vars)
	test.NotEmpty(t, uri)
	req := assets.NewAuthentifiedRequest(t, u, pass, "GET", uri, vars)

	//Do the request
	rec := httptest.NewRecorder()
	router.Mux.ServeHTTP(rec, req)
	assert.Equal(t, 200, rec.Code)

	history := []sdk.WorkflowNodeRun{}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &history))
	assert.Equal(t, 2, len(history))
	assert.Equal(t, int64(1), history[0].SubNumber)
	assert.Equal(t, int64(0), history[1].SubNumber)
}

func Test_getWorkflowRunsHandler(t *testing.T) {
	api, db, router, end := newTestAPI(t, bootstrap.InitiliazeDB)
	defer end()
	u, pass := assets.InsertAdminUser(t, api.mustDB())
	consumer, _ := authentication.LoadConsumerByTypeAndUserID(context.TODO(), db, sdk.ConsumerLocal, u.ID, authentication.LoadConsumerOptions.WithAuthentifiedUser)
	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, api.Cache, key, key)

	//First pipeline
	pip := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip1",
	}
	require.NoError(t, pipeline.InsertPipeline(api.mustDB(), api.Cache, proj, &pip))

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
	}
	require.NoError(t, pipeline.InsertPipeline(api.mustDB(), api.Cache, proj, &pip2))
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
	require.NoError(t, errP)

	require.NoError(t, workflow.Insert(context.TODO(), api.mustDB(), api.Cache, &w, proj2))
	w1, err := workflow.Load(context.TODO(), api.mustDB(), api.Cache, proj, "test_1", workflow.LoadOptions{})
	require.NoError(t, err)

	for i := 0; i < 10; i++ {
		wr, err := workflow.CreateRun(db, w1, nil, u)
		assert.NoError(t, err)
		wr.Workflow = *w1
		_, err = workflow.StartWorkflowRun(context.TODO(), db, api.Cache, proj, wr, &sdk.WorkflowRunPostHandlerOption{
			Manual: &sdk.WorkflowNodeRunManual{
				Username: u.GetUsername(),
			},
		}, consumer, nil)
		require.NoError(t, err)
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
	assert.Equal(t, 200, rec.Code)
	assert.Equal(t, "0-100/10", rec.Header().Get("Content-Range"))
	runs := make([]sdk.WorkflowRun, 0)
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &runs))
	assert.Equal(t, 10, len(runs))
}

func Test_getWorkflowRunsHandlerWithFilter(t *testing.T) {
	api, db, router, end := newTestAPI(t, bootstrap.InitiliazeDB)
	defer end()
	u, pass := assets.InsertAdminUser(t, api.mustDB())
	consumer, _ := authentication.LoadConsumerByTypeAndUserID(context.TODO(), db, sdk.ConsumerLocal, u.ID, authentication.LoadConsumerOptions.WithAuthentifiedUser)

	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, api.Cache, key, key)

	//First pipeline
	pip := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip1",
	}
	require.NoError(t, pipeline.InsertPipeline(api.mustDB(), api.Cache, proj, &pip))

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
	}
	require.NoError(t, pipeline.InsertPipeline(api.mustDB(), api.Cache, proj, &pip2))
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
	require.NoError(t, errP)

	require.NoError(t, workflow.Insert(context.TODO(), api.mustDB(), api.Cache, &w, proj2))
	w1, err := workflow.Load(context.TODO(), api.mustDB(), api.Cache, proj, "test_1", workflow.LoadOptions{})
	require.NoError(t, err)

	wr, err := workflow.CreateRun(db, w1, nil, u)
	assert.NoError(t, err)
	wr.Workflow = *w1
	_, err = workflow.StartWorkflowRun(context.TODO(), db, api.Cache, proj, wr, &sdk.WorkflowRunPostHandlerOption{
		Manual: &sdk.WorkflowNodeRunManual{
			Username: u.GetUsername(),
		},
	}, consumer, nil)
	require.NoError(t, err)

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
	api, db, router, end := newTestAPI(t, bootstrap.InitiliazeDB)
	defer end()
	u, pass := assets.InsertAdminUser(t, api.mustDB())
	consumer, _ := authentication.LoadConsumerByTypeAndUserID(context.TODO(), db, sdk.ConsumerLocal, u.ID, authentication.LoadConsumerOptions.WithAuthentifiedUser)

	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, api.Cache, key, key)

	//First pipeline
	pip := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip1",
	}
	require.NoError(t, pipeline.InsertPipeline(api.mustDB(), api.Cache, proj, &pip))

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
	}
	require.NoError(t, pipeline.InsertPipeline(api.mustDB(), api.Cache, proj, &pip2))
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
	require.NoError(t, errP)

	require.NoError(t, workflow.Insert(context.TODO(), api.mustDB(), api.Cache, &w, proj2))
	w1, err := workflow.Load(context.TODO(), api.mustDB(), api.Cache, proj, "test_1", workflow.LoadOptions{})
	require.NoError(t, err)

	for i := 0; i < 10; i++ {
		wr, err := workflow.CreateRun(db, w1, nil, u)
		wr.Workflow = *w1
		assert.NoError(t, err)
		_, err = workflow.StartWorkflowRun(context.TODO(), db, api.Cache, proj, wr, &sdk.WorkflowRunPostHandlerOption{
			Manual: &sdk.WorkflowNodeRunManual{
				Username: u.GetUsername(),
				Payload: map[string]string{
					"git.branch": "master",
					"git.hash":   fmt.Sprintf("%d", i),
				},
			},
		}, consumer, nil)
		require.NoError(t, err)
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
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), wr))
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
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &tags))
	assert.Len(t, tags, 1)

}

func Test_getWorkflowRunHandler(t *testing.T) {
	api, db, router, end := newTestAPI(t, bootstrap.InitiliazeDB)
	defer end()
	u, pass := assets.InsertAdminUser(t, api.mustDB())
	consumer, _ := authentication.LoadConsumerByTypeAndUserID(context.TODO(), db, sdk.ConsumerLocal, u.ID, authentication.LoadConsumerOptions.WithAuthentifiedUser)

	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, api.Cache, key, key)

	//First pipeline
	pip := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip1",
	}
	require.NoError(t, pipeline.InsertPipeline(api.mustDB(), api.Cache, proj, &pip))

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
	}
	require.NoError(t, pipeline.InsertPipeline(api.mustDB(), api.Cache, proj, &pip2))
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
	require.NoError(t, errP)

	require.NoError(t, workflow.Insert(context.TODO(), api.mustDB(), api.Cache, &w, proj2))
	w1, err := workflow.Load(context.TODO(), api.mustDB(), api.Cache, proj, "test_1", workflow.LoadOptions{})
	require.NoError(t, err)

	for i := 0; i < 10; i++ {
		wr, err := workflow.CreateRun(db, w1, nil, u)
		assert.NoError(t, err)
		wr.Workflow = *w1
		_, err = workflow.StartWorkflowRun(context.TODO(), db, api.Cache, proj, wr, &sdk.WorkflowRunPostHandlerOption{
			Manual: &sdk.WorkflowNodeRunManual{
				Username: u.GetUsername(),
			},
		}, consumer, nil)
		require.NoError(t, err)
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
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), wr))
	assert.Equal(t, int64(9), wr.Number)
}

func Test_getWorkflowNodeRunHandler(t *testing.T) {
	api, db, router, end := newTestAPI(t, bootstrap.InitiliazeDB)
	defer end()
	u, pass := assets.InsertAdminUser(t, api.mustDB())
	consumer, _ := authentication.LoadConsumerByTypeAndUserID(context.TODO(), db, sdk.ConsumerLocal, u.ID, authentication.LoadConsumerOptions.WithAuthentifiedUser)

	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, api.Cache, key, key)

	// Application
	app := sdk.Application{
		ProjectID: proj.ID,
		Name:      "app",
	}
	require.NoError(t, application.Insert(db, api.Cache, proj, &app))

	//First pipeline
	pip := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip1",
	}
	require.NoError(t, pipeline.InsertPipeline(api.mustDB(), api.Cache, proj, &pip))

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
	}
	require.NoError(t, pipeline.InsertPipeline(api.mustDB(), api.Cache, proj, &pip2))
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
		WorkflowData: &sdk.WorkflowData{
			Node: sdk.Node{
				Name: "root",
				Type: sdk.NodeTypePipeline,
				Context: &sdk.NodeContext{
					ApplicationID: app.ID,
					PipelineID:    pip.ID,
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

	proj2, errP := project.Load(api.mustDB(), api.Cache, proj.Key, project.LoadOptions.WithPipelines, project.LoadOptions.WithGroups, project.LoadOptions.WithIntegrations, project.LoadOptions.WithApplications)
	require.NoError(t, errP)

	require.NoError(t, workflow.Insert(context.TODO(), api.mustDB(), api.Cache, &w, proj2))
	w1, err := workflow.Load(context.TODO(), api.mustDB(), api.Cache, proj, "test_1", workflow.LoadOptions{})
	require.NoError(t, err)

	wr, err := workflow.CreateRun(db, w1, nil, u)
	assert.NoError(t, err)
	wr.Workflow = *w1
	_, err = workflow.StartWorkflowRun(context.TODO(), db, api.Cache, proj2, wr, &sdk.WorkflowRunPostHandlerOption{
		Manual: &sdk.WorkflowNodeRunManual{
			Username: u.GetUsername(),
		},
	}, consumer, nil)
	require.NoError(t, err)

	lastrun, err := workflow.LoadLastRun(api.mustDB(), proj.Key, w1.Name, workflow.LoadRunOptions{WithArtifacts: true, WithTests: true})
	require.NoError(t, err)

	vuln := sdk.Vulnerability{
		ApplicationID: app.ID,
		Ignored:       false,
		Component:     "lodash",
		CVE:           "",
		Description:   "",
		FixIn:         "",
		Origin:        "",
		Severity:      "high",
		Title:         "",
		Version:       "",
		Link:          "",
	}
	report := sdk.WorkflowNodeRunVulnerabilityReport{
		ApplicationID:     app.ID,
		WorkflowRunID:     lastrun.ID,
		WorkflowNodeRunID: lastrun.WorkflowNodeRuns[w1.WorkflowData.Node.ID][0].ID,
		Num:               lastrun.Number,
		Report: sdk.WorkflowNodeRunVulnerability{
			Vulnerabilities: []sdk.Vulnerability{vuln},
		},
	}
	assert.NoError(t, workflow.InsertVulnerabilityReport(db, report))

	//Prepare request
	vars := map[string]string{
		"key":              proj.Key,
		"permWorkflowName": w1.Name,
		"number":           fmt.Sprintf("%d", lastrun.Number),
		"nodeRunID":        fmt.Sprintf("%d", lastrun.WorkflowNodeRuns[w1.WorkflowData.Node.ID][0].ID),
	}
	uri := router.GetRoute("GET", api.getWorkflowNodeRunHandler, vars)
	test.NotEmpty(t, uri)
	req := assets.NewAuthentifiedRequest(t, u, pass, "GET", uri, vars)

	//Do the request
	rec := httptest.NewRecorder()
	router.Mux.ServeHTTP(rec, req)
	assert.Equal(t, 200, rec.Code)

	var nr sdk.WorkflowNodeRun
	assert.NoError(t, json.Unmarshal(rec.Body.Bytes(), &nr))
	assert.Equal(t, 1, len(nr.VulnerabilitiesReport.Report.Vulnerabilities))
}

func Test_resyncWorkflowRunHandler(t *testing.T) {
	api, db, router, end := newTestAPI(t, bootstrap.InitiliazeDB)
	defer end()
	u, pass := assets.InsertAdminUser(t, db)
	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, api.Cache, key, key)

	//First pipeline
	pip := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip1",
	}
	require.NoError(t, pipeline.InsertPipeline(db, api.Cache, proj, &pip))

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
	require.NoError(t, errP)

	require.NoError(t, workflow.Insert(context.TODO(), db, api.Cache, &w, proj2))
	w1, err := workflow.Load(context.TODO(), db, api.Cache, proj2, "test_1", workflow.LoadOptions{})
	require.NoError(t, err)

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
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), wr))
	assert.Equal(t, int64(1), wr.Number)

	cpt := 0
	for {
		varsGet := map[string]string{
			"key":              proj.Key,
			"permWorkflowName": w1.Name,
			"number":           "1",
		}
		uriGet := router.GetRoute("GET", api.getWorkflowRunHandler, varsGet)
		reqGet := assets.NewAuthentifiedRequest(t, u, pass, "GET", uriGet, nil)
		recGet := httptest.NewRecorder()
		router.Mux.ServeHTTP(recGet, reqGet)

		var wrGet sdk.WorkflowRun
		assert.NoError(t, json.Unmarshal(recGet.Body.Bytes(), &wrGet))

		if wrGet.Status != sdk.StatusPending {
			assert.Equal(t, sdk.StatusBuilding, wrGet.Status)
			break
		}
		cpt++
		if cpt > 10 {
			break
		}
	}

	pip.Stages[0].Name = "New awesome stage"
	errS := pipeline.UpdateStage(db, &pip.Stages[0])
	require.NoError(t, errS)

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
	require.NoError(t, errWR)

	assert.Equal(t, "New awesome stage", workflowRun.Workflow.Pipelines[pip.ID].Stages[0].Name)
}

func Test_postWorkflowRunHandler(t *testing.T) {
	api, db, router, end := newTestAPI(t, bootstrap.InitiliazeDB)
	defer end()
	u, pass := assets.InsertAdminUser(t, api.mustDB())
	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, api.Cache, key, key)

	//First pipeline
	pip := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip1",
	}
	require.NoError(t, pipeline.InsertPipeline(api.mustDB(), api.Cache, proj, &pip))

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
	}
	require.NoError(t, pipeline.InsertPipeline(api.mustDB(), api.Cache, proj, &pip2))
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
	require.NoError(t, errP)

	require.NoError(t, workflow.Insert(context.TODO(), api.mustDB(), api.Cache, &w, proj2))
	w1, err := workflow.Load(context.TODO(), api.mustDB(), api.Cache, proj, "test_1", workflow.LoadOptions{})
	require.NoError(t, err)

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
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), wr))
	assert.Equal(t, int64(1), wr.Number)
}

func Test_postWorkflowRunAsyncFailedHandler(t *testing.T) {
	api, db, router, end := newTestAPI(t, bootstrap.InitiliazeDB)
	defer end()
	u, pass := assets.InsertAdminUser(t, api.mustDB())
	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, api.Cache, key, key)

	assert.NoError(t, repositoriesmanager.InsertForProject(db, proj, &sdk.ProjectVCSServer{
		Name: "github",
		Data: map[string]string{
			"token":  "foo",
			"secret": "bar",
		},
	}))

	//First pipeline
	pip := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip1",
	}
	require.NoError(t, pipeline.InsertPipeline(api.mustDB(), api.Cache, proj, &pip))

	// Create Application
	app := sdk.Application{
		Name:               sdk.RandomString(10),
		ProjectID:          proj.ID,
		RepositoryFullname: "foo/myrepo",
		VCSServer:          "github",
	}
	assert.NoError(t, application.Insert(db, api.Cache, proj, &app))
	assert.NoError(t, repositoriesmanager.InsertForApplication(db, &app, proj.Key))

	w := sdk.Workflow{
		Name:       "test_1",
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		WorkflowData: &sdk.WorkflowData{
			Node: sdk.Node{
				Name: "root",
				Type: sdk.NodeTypePipeline,
				Context: &sdk.NodeContext{
					PipelineID:    pip.ID,
					ApplicationID: app.ID,
				},
			},
		},
	}

	proj2, errP := project.Load(api.mustDB(), api.Cache, proj.Key, project.LoadOptions.WithApplications, project.LoadOptions.WithPipelines, project.LoadOptions.WithGroups, project.LoadOptions.WithIntegrations)
	require.NoError(t, errP)

	require.NoError(t, workflow.Insert(context.TODO(), api.mustDB(), api.Cache, &w, proj2))
	w1, err := workflow.Load(context.TODO(), api.mustDB(), api.Cache, proj, "test_1", workflow.LoadOptions{})
	require.NoError(t, err)

	allSrv, err := services.LoadAll(context.TODO(), db)
	for _, s := range allSrv {
		if err := services.Delete(db, &s); err != nil {
			t.Fatalf("unable to delete service: %v", err)
		}
	}

	// Prepare VCS Mock
	mockVCSSservice, _ := assets.InsertService(t, db, "Test_postWorkflowRunAsyncFailedHandlerVCS", services.TypeVCS)
	defer func() {
		_ = services.Delete(db, mockVCSSservice) // nolint
	}()

	mockRepoService, _ := assets.InsertService(t, db, "Test_postWorkflowRunAsyncFailedHandlerRepo", services.TypeRepositories)
	defer func() {
		_ = services.Delete(db, mockRepoService) // nolint
	}()

	mockHookService, _ := assets.InsertService(t, db, "Test_postWorkflowRunAsyncFailedHandlerHook", services.TypeHooks)
	defer func() {
		_ = services.Delete(db, mockHookService) // nolint
	}()

	//This is a mock for the repositories service
	services.HTTPClient = mock(
		func(r *http.Request) (*http.Response, error) {
			t.Logf("[MOCK] %s %v", r.Method, r.URL)
			body := new(bytes.Buffer)
			w := new(http.Response)
			enc := json.NewEncoder(body)
			w.Body = ioutil.NopCloser(body)

			switch r.URL.String() {
			case "/operations/123":
				ope := sdk.Operation{
					UUID:   "123",
					Status: sdk.OperationStatusDone,
				}
				if err := enc.Encode(ope); err != nil {
					return writeError(w, err)
				}
			case "/vcs/github/webhooks":
				res := struct {
					WebhooksSupported bool   `json:"webhooks_supported"`
					WebhooksDisabled  bool   `json:"webhooks_disabled"`
					WebhooksIcon      string `json:"webhooks_icon"`
				}{
					WebhooksDisabled:  false,
					WebhooksIcon:      "",
					WebhooksSupported: true,
				}
				if err := enc.Encode(res); err != nil {
					return writeError(w, err)
				}
			case "/vcs/github/repos/foo/myrepo/hooks":
				h := sdk.VCSHook{}
				h.Name = "hook"
				if err := enc.Encode(h); err != nil {
					return writeError(w, err)
				}
			case "/vcs/github/repos/foo/myrepo/pullrequests":
				pr := sdk.VCSPullRequest{
					Title: "blabla",
				}
				if err := enc.Encode(pr); err != nil {
					return writeError(w, err)
				}
			case "/vcs/github/repos/foo/myrepo/pullrequests/0":
				return writeError(w, fmt.Errorf("error for test"))

			case "/task/bulk":
				hooks := map[string]sdk.NodeHook{}
				hooks["123"] = sdk.NodeHook{
					UUID: "123",
				}
				if err := enc.Encode(hooks); err != nil {
					return writeError(w, err)
				}
			default:
				return writeError(w, fmt.Errorf("route %s must not be called", r.URL.String()))
			}
			return w, nil
		},
	)

	// Migrate as code
	ope := sdk.Operation{
		UUID: "123",
		Setup: sdk.OperationSetup{
			Push: sdk.OperationPush{},
		},
	}
	workflow.UpdateWorkflowAsCodeResult(context.TODO(), api.mustDB(), api.Cache, proj, &ope, w1, u)

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

	cpt := 0
	for {
		t.Logf("Attempt getWorkflowRunHandler #%d", cpt)
		varsGet := map[string]string{
			"key":              proj.Key,
			"permWorkflowName": w1.Name,
			"number":           "1",
		}
		uriGet := router.GetRoute("GET", api.getWorkflowRunHandler, varsGet)
		reqGet := assets.NewAuthentifiedRequest(t, u, pass, "GET", uriGet, nil)
		recGet := httptest.NewRecorder()
		router.Mux.ServeHTTP(recGet, reqGet)

		var wrGet sdk.WorkflowRun
		recGetBody := recGet.Body.Bytes()
		t.Logf("getWorkflowRunHandler response: %s", string(recGetBody))
		assert.NoError(t, json.Unmarshal(recGetBody, &wrGet))

		if wrGet.Status != sdk.StatusPending {
			assert.Equal(t, sdk.StatusFail, wrGet.Status)
			assert.Equal(t, 1, len(wrGet.Infos))
			assert.Equal(t, wrGet.Infos[0].Message.ID, sdk.MsgWorkflowError.ID)
			return
		}
		cpt++
		if cpt > 10 {
			break
		}
	}
	t.FailNow()
}

func Test_postWorkflowRunHandlerWithoutRightOnEnvironment(t *testing.T) {
	api, db, router, end := newTestAPI(t, bootstrap.InitiliazeDB)
	defer end()
	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, api.Cache, key, key)

	//First pipeline
	pip := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip1",
	}
	require.NoError(t, pipeline.InsertPipeline(api.mustDB(), api.Cache, proj, &pip))

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
	}
	require.NoError(t, pipeline.InsertPipeline(api.mustDB(), api.Cache, proj, &pip2))
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
	require.NoError(t, environment.InsertEnvironment(api.mustDB(), &env))
	gr := sdk.Group{
		Name: sdk.RandomString(10),
	}
	require.NoError(t, group.Insert(api.mustDB(), &gr))

	uLambda, pass := assets.InsertLambdaUser(t, api.mustDB(), &gr)

	w := sdk.Workflow{
		Name:       "test_1",
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		WorkflowData: &sdk.WorkflowData{
			Node: sdk.Node{
				Name: "root",
				Type: sdk.NodeTypePipeline,
				Context: &sdk.NodeContext{
					PipelineID:    pip.ID,
					EnvironmentID: env.ID,
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

	proj2, errP := project.Load(api.mustDB(), api.Cache, proj.Key, project.LoadOptions.WithPipelines, project.LoadOptions.WithGroups, project.LoadOptions.WithEnvironments)
	require.NoError(t, errP)

	require.NoError(t, workflow.Insert(context.TODO(), api.mustDB(), api.Cache, &w, proj2))
	w1, err := workflow.Load(context.TODO(), api.mustDB(), api.Cache, proj2, "test_1", workflow.LoadOptions{})
	require.NoError(t, err)

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

func Test_postWorkflowRunHandler_Forbidden(t *testing.T) {
	api, db, router, end := newTestAPI(t, bootstrap.InitiliazeDB)
	defer end()
	u, pass := assets.InsertAdminUser(t, api.mustDB())
	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, api.Cache, key, key)

	gr := &sdk.Group{
		Name: sdk.RandomString(10),
	}
	require.NoError(t, group.Insert(db, gr))
	require.NoError(t, group.InsertLinkGroupProject(api.mustDB(), &group.LinkGroupProject{
		GroupID:   gr.ID,
		ProjectID: proj.ID,
		Role:      7,
	}))

	//First pipeline
	pip := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip1",
	}
	require.NoError(t, pipeline.InsertPipeline(api.mustDB(), api.Cache, proj, &pip))

	env := &sdk.Environment{
		Name:       sdk.RandomString(10),
		ProjectKey: proj.Key,
		ProjectID:  proj.ID,
	}
	require.NoError(t, environment.InsertEnvironment(api.mustDB(), env))

	proj2, errp := project.Load(api.mustDB(), api.Cache, proj.Key, project.LoadOptions.WithPipelines, project.LoadOptions.WithEnvironments)
	require.NoError(t, errp)

	w := sdk.Workflow{
		Name:       "test_1",
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		WorkflowData: &sdk.WorkflowData{
			Node: sdk.Node{
				Name: "root",
				Type: sdk.NodeTypePipeline,
				Context: &sdk.NodeContext{
					PipelineID:    pip.ID,
					EnvironmentID: env.ID,
				},
			},
		},
	}

	require.NoError(t, workflow.Insert(context.TODO(), api.mustDB(), api.Cache, &w, proj2))

	u.Ring = ""
	require.NoError(t, user.Update(api.mustDB(), u))

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
func Test_postWorkflowRunHandler_BadPayload(t *testing.T) {
	api, db, router, end := newTestAPI(t, bootstrap.InitiliazeDB)
	defer end()
	u, pass := assets.InsertAdminUser(t, api.mustDB())
	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, api.Cache, key, key)

	gr := &sdk.Group{
		Name: sdk.RandomString(10),
	}
	require.NoError(t, group.Insert(db, gr))
	require.NoError(t, group.InsertLinkGroupProject(api.mustDB(), &group.LinkGroupProject{
		GroupID:   gr.ID,
		ProjectID: proj.ID,
		Role:      7,
	}))

	//First pipeline
	pip := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip1",
	}
	require.NoError(t, pipeline.InsertPipeline(api.mustDB(), api.Cache, proj, &pip))

	env := &sdk.Environment{
		Name:       sdk.RandomString(10),
		ProjectKey: proj.Key,
		ProjectID:  proj.ID,
	}
	require.NoError(t, environment.InsertEnvironment(api.mustDB(), env))

	proj2, errp := project.Load(api.mustDB(), api.Cache, proj.Key, project.LoadOptions.WithPipelines, project.LoadOptions.WithEnvironments)
	require.NoError(t, errp)

	w := sdk.Workflow{
		Name:       "test_1",
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		WorkflowData: &sdk.WorkflowData{
			Node: sdk.Node{
				Name: "root",
				Type: sdk.NodeTypePipeline,
				Context: &sdk.NodeContext{
					PipelineID:    pip.ID,
					EnvironmentID: env.ID,
				},
			},
		},
	}

	require.NoError(t, workflow.Insert(context.TODO(), api.mustDB(), api.Cache, &w, proj2))

	require.NoError(t, user.Update(api.mustDB(), u))

	//Prepare request
	vars := map[string]string{
		"key":              proj.Key,
		"permWorkflowName": w.Name,
	}
	uri := router.GetRoute("POST", api.postWorkflowRunHandler, vars)
	test.NotEmpty(t, uri)

	opts := &sdk.WorkflowRunPostHandlerOption{
		Manual: &sdk.WorkflowNodeRunManual{
			Payload: map[string]string{"cds.test": "test"},
		},
	}
	req := assets.NewAuthentifiedRequest(t, u, pass, "POST", uri, opts)

	//Do the request
	rec := httptest.NewRecorder()
	router.Mux.ServeHTTP(rec, req)
	assert.Equal(t, 400, rec.Code)
}

func initGetWorkflowNodeRunJobTest(t *testing.T, api *API, db *gorp.DbMap) (*sdk.AuthentifiedUser, string, *sdk.Project, *sdk.Workflow, *sdk.WorkflowRun, *sdk.WorkflowNodeJobRun) {
	u, pass := assets.InsertAdminUser(t, api.mustDB())
	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, api.Cache, key, key)

	//First pipeline
	pip := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip1",
	}
	require.NoError(t, pipeline.InsertPipeline(api.mustDB(), api.Cache, proj, &pip))

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
	}
	require.NoError(t, pipeline.InsertPipeline(api.mustDB(), api.Cache, proj, &pip2))
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
	require.NoError(t, errP)
	consumer, _ := authentication.LoadConsumerByTypeAndUserID(context.TODO(), db, sdk.ConsumerLocal, u.ID, authentication.LoadConsumerOptions.WithAuthentifiedUser)

	require.NoError(t, workflow.Insert(context.TODO(), api.mustDB(), api.Cache, &w, proj2))
	w1, err := workflow.Load(context.TODO(), api.mustDB(), api.Cache, proj, "test_1", workflow.LoadOptions{
		DeepPipeline: true,
	})
	require.NoError(t, err)

	wr, err := workflow.CreateRun(db, w1, nil, u)
	assert.NoError(t, err)
	wr.Workflow = *w1
	_, err = workflow.StartWorkflowRun(context.TODO(), db, api.Cache, proj, wr, &sdk.WorkflowRunPostHandlerOption{
		Manual: &sdk.WorkflowNodeRunManual{
			Username: u.GetUsername(),
		},
	}, consumer, nil)
	require.NoError(t, err)

	lastRun, err := workflow.LoadLastRun(api.mustDB(), proj.Key, w1.Name, workflow.LoadRunOptions{WithArtifacts: true})
	require.NoError(t, err)

	// Update step status
	jobRun := &lastRun.WorkflowNodeRuns[w1.WorkflowData.Node.ID][0].Stages[0].RunJobs[0]
	jobRun.Job.StepStatus = []sdk.StepStatus{
		{
			StepOrder: 1,
			Status:    sdk.StatusBuilding,
		},
	}

	// Update node job run
	errUJ := workflow.UpdateNodeRun(api.mustDB(), &lastRun.WorkflowNodeRuns[w1.WorkflowData.Node.ID][0])
	require.NoError(t, errUJ)

	// Add log
	require.NoError(t, workflow.AddLog(api.mustDB(), jobRun, &sdk.Log{
		StepOrder: 1,
		Val:       "1234567890",
	}, 15))

	// Add truncated log
	require.NoError(t, workflow.AddLog(api.mustDB(), jobRun, &sdk.Log{
		StepOrder: 1,
		Val:       "1234567890",
	}, 15))

	// Add service log
	require.NoError(t, workflow.AddServiceLog(api.mustDB(), jobRun, &sdk.ServiceLog{
		Val: "0987654321",
	}, 15))

	// Add truncated service log
	require.NoError(t, workflow.AddServiceLog(api.mustDB(), jobRun, &sdk.ServiceLog{
		Val: "0987654321",
	}, 15))

	return u, pass, proj, w1, lastRun, jobRun
}

func Test_getWorkflowNodeRunJobStepHandler(t *testing.T) {
	api, db, router, end := newTestAPI(t)
	defer end()

	u, pass, proj, w1, lastRun, jobRun := initGetWorkflowNodeRunJobTest(t, api, db)

	//Prepare request
	vars := map[string]string{
		"key":              proj.Key,
		"permWorkflowName": w1.Name,
		"number":           fmt.Sprintf("%d", lastRun.Number),
		"nodeRunID":        fmt.Sprintf("%d", lastRun.WorkflowNodeRuns[w1.WorkflowData.Node.ID][0].ID),
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
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), stepState))
	assert.Equal(t, 200, rec.Code)
	assert.Equal(t, "123456789012345... truncated\n", stepState.StepLogs.Val)
	assert.Equal(t, sdk.StatusBuilding, stepState.Status)
}

func Test_getWorkflowNodeRunJobServiceLogsHandler(t *testing.T) {
	api, db, router, end := newTestAPI(t)
	defer end()

	u, pass, proj, w1, lastRun, jobRun := initGetWorkflowNodeRunJobTest(t, api, db)

	//Prepare request
	vars := map[string]string{
		"key":              proj.Key,
		"permWorkflowName": w1.Name,
		"number":           fmt.Sprintf("%d", lastRun.Number),
		"nodeRunID":        fmt.Sprintf("%d", lastRun.WorkflowNodeRuns[w1.WorkflowData.Node.ID][0].ID),
		"runJobId":         fmt.Sprintf("%d", jobRun.ID),
	}
	uri := router.GetRoute("GET", api.getWorkflowNodeRunJobServiceLogsHandler, vars)
	test.NotEmpty(t, uri)
	req := assets.NewAuthentifiedRequest(t, u, pass, "GET", uri, vars)

	//Do the request
	rec := httptest.NewRecorder()
	router.Mux.ServeHTTP(rec, req)

	var logs []sdk.ServiceLog
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &logs))
	assert.Equal(t, 200, rec.Code)
	assert.Equal(t, "098765432109876... truncated\n", logs[0].Val)
}

func Test_deleteWorkflowRunsBranchHandler(t *testing.T) {
	api, db, router, end := newTestAPI(t, bootstrap.InitiliazeDB)
	defer end()
	u, pass := assets.InsertAdminUser(t, api.mustDB())
	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, api.Cache, key, key)

	//First pipeline
	pip := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip1",
	}
	require.NoError(t, pipeline.InsertPipeline(api.mustDB(), api.Cache, proj, &pip))

	s := sdk.NewStage("stage 1")
	s.Enabled = true
	s.PipelineID = pip.ID
	require.NoError(t, pipeline.InsertStage(api.mustDB(), s))
	j := &sdk.Job{
		Enabled: true,
		Action: sdk.Action{
			Enabled: true,
		},
	}
	require.NoError(t, pipeline.InsertJob(api.mustDB(), j, s.ID, &pip))
	s.Jobs = append(s.Jobs, *j)

	pip.Stages = append(pip.Stages, *s)

	//Second pipeline
	pip2 := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip2",
	}
	require.NoError(t, pipeline.InsertPipeline(api.mustDB(), api.Cache, proj, &pip2))
	s = sdk.NewStage("stage 1")
	s.Enabled = true
	s.PipelineID = pip2.ID
	require.NoError(t, pipeline.InsertStage(api.mustDB(), s))
	j = &sdk.Job{
		Enabled: true,
		Action: sdk.Action{
			Enabled: true,
		},
	}
	require.NoError(t, pipeline.InsertJob(api.mustDB(), j, s.ID, &pip2))
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

	consumer, _ := authentication.LoadConsumerByTypeAndUserID(context.TODO(), db, sdk.ConsumerLocal, u.ID, authentication.LoadConsumerOptions.WithAuthentifiedUser)

	proj2, errP := project.Load(api.mustDB(), api.Cache, proj.Key, project.LoadOptions.WithPipelines, project.LoadOptions.WithGroups, project.LoadOptions.WithIntegrations)
	require.NoError(t, errP)

	require.NoError(t, workflow.Insert(context.TODO(), api.mustDB(), api.Cache, &w, proj2))
	w1, err := workflow.Load(context.TODO(), api.mustDB(), api.Cache, proj, "test_1", workflow.LoadOptions{})
	require.NoError(t, err)

	wr, err := workflow.CreateRun(db, w1, nil, u)
	assert.NoError(t, err)
	wr.Workflow = *w1
	wr.Tag("git.branch", "master")
	assert.NoError(t, workflow.UpdateWorkflowRun(context.TODO(), api.mustDB(), wr))
	_, err = workflow.StartWorkflowRun(context.TODO(), db, api.Cache, proj, wr, &sdk.WorkflowRunPostHandlerOption{
		Manual: &sdk.WorkflowNodeRunManual{
			Username: u.GetUsername(),
			Payload:  `{"git.branch": "master"}`,
		},
	}, consumer, nil)
	require.NoError(t, err)

	mockHookService, _ := assets.InsertService(t, db, "Test_deleteWorkflowRunsBranchHandler", services.TypeHooks, sdk.AuthConsumerScopeRun)
	defer func() {
		_ = services.Delete(db, mockHookService) // nolint
	}()

	serviceConsumer, err := authentication.LoadConsumerByID(context.TODO(), db, *mockHookService.ConsumerID)
	require.NoError(t, err)

	session, err := authentication.NewSession(db, serviceConsumer, 5*time.Minute, false)
	require.NoError(t, err)

	jwt, err := authentication.NewSessionJWT(session)
	require.NoError(t, err)

	//Prepare request
	vars := map[string]string{
		"key":              proj.Key,
		"permWorkflowName": w1.Name,
		"branch":           "master",
	}
	uri := router.GetRoute("DELETE", api.deleteWorkflowRunsBranchHandler, vars)
	test.NotEmpty(t, uri)
	req := assets.NewAuthentifiedRequest(t, nil, jwt, "DELETE", uri, vars)

	//Do the request
	rec := httptest.NewRecorder()
	router.Mux.ServeHTTP(rec, req)
	require.Equal(t, 200, rec.Code)

	//Prepare request
	vars = map[string]string{
		"key":              proj.Key,
		"permWorkflowName": w1.Name,
	}
	uri = router.GetRoute("GET", api.getWorkflowRunsHandler, vars)
	test.NotEmpty(t, uri)
	req = assets.NewAuthentifiedRequest(t, u, pass, "GET", uri, vars)

	//Do the request
	rec = httptest.NewRecorder()
	router.Mux.ServeHTTP(rec, req)
	require.Equal(t, 200, rec.Code)

	var wfRuns []sdk.WorkflowRun
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &wfRuns))
	require.Equal(t, 0, len(wfRuns))
}

func Test_deleteWorkflowRunHandler(t *testing.T) {
	api, db, router, end := newTestAPI(t, bootstrap.InitiliazeDB)
	defer end()
	u, pass := assets.InsertAdminUser(t, api.mustDB())
	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, api.Cache, key, key)

	//First pipeline
	pip := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip1",
	}
	require.NoError(t, pipeline.InsertPipeline(api.mustDB(), api.Cache, proj, &pip))

	s := sdk.NewStage("stage 1")
	s.Enabled = true
	s.PipelineID = pip.ID
	require.NoError(t, pipeline.InsertStage(api.mustDB(), s))
	j := &sdk.Job{
		Enabled: true,
		Action: sdk.Action{
			Enabled: true,
		},
	}
	require.NoError(t, pipeline.InsertJob(api.mustDB(), j, s.ID, &pip))
	s.Jobs = append(s.Jobs, *j)

	pip.Stages = append(pip.Stages, *s)

	//Second pipeline
	pip2 := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip2",
	}
	require.NoError(t, pipeline.InsertPipeline(api.mustDB(), api.Cache, proj, &pip2))
	s = sdk.NewStage("stage 1")
	s.Enabled = true
	s.PipelineID = pip2.ID
	require.NoError(t, pipeline.InsertStage(api.mustDB(), s))
	j = &sdk.Job{
		Enabled: true,
		Action: sdk.Action{
			Enabled: true,
		},
	}
	require.NoError(t, pipeline.InsertJob(api.mustDB(), j, s.ID, &pip2))
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
	require.NoError(t, errP)

	require.NoError(t, workflow.Insert(context.TODO(), api.mustDB(), api.Cache, &w, proj2))
	w1, err := workflow.Load(context.TODO(), api.mustDB(), api.Cache, proj, "test_1", workflow.LoadOptions{})
	require.NoError(t, err)

	wr, err := workflow.CreateRun(db, w1, nil, u)
	assert.NoError(t, err)
	//Prepare request
	vars := map[string]string{
		"key":              proj.Key,
		"permWorkflowName": w1.Name,
		"number":           fmt.Sprintf("%d", wr.Number),
	}
	uri := router.GetRoute("DELETE", api.deleteWorkflowRunHandler, vars)
	test.NotEmpty(t, uri)
	req := assets.NewAuthentifiedRequest(t, u, pass, "DELETE", uri, vars)

	//Do the request
	rec := httptest.NewRecorder()
	router.Mux.ServeHTTP(rec, req)
	assert.Equal(t, 202, rec.Code)

	//Prepare request
	vars = map[string]string{
		"key":              proj.Key,
		"permWorkflowName": w1.Name,
		"number":           fmt.Sprintf("%d", wr.Number),
	}
	uri = router.GetRoute("GET", api.getWorkflowRunHandler, vars)
	test.NotEmpty(t, uri)
	req = assets.NewAuthentifiedRequest(t, u, pass, "GET", uri, vars)

	//Do the request
	rec = httptest.NewRecorder()
	router.Mux.ServeHTTP(rec, req)
	assert.Equal(t, 404, rec.Code)
}
