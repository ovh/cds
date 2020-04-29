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

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/ascode"
	"github.com/ovh/cds/engine/api/authentication"
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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_getWorkflowNodeRunHistoryHandler(t *testing.T) {
	api, db, router, end := newTestAPI(t)
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
	require.NoError(t, pipeline.InsertPipeline(db, &pip))

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
	require.NoError(t, pipeline.InsertPipeline(db, &pip2))
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

	proj2, errP := project.Load(api.mustDB(), proj.Key, project.LoadOptions.WithPipelines, project.LoadOptions.WithGroups, project.LoadOptions.WithIntegrations)
	require.NoError(t, errP)

	require.NoError(t, workflow.Insert(context.TODO(), db, api.Cache, *proj2, &w))
	w1, err := workflow.Load(context.TODO(), db, api.Cache, *proj, "test_1", workflow.LoadOptions{})
	require.NoError(t, err)

	wrCreate, err := workflow.CreateRun(db, w1, nil, u)
	assert.NoError(t, err)
	wrCreate.Workflow = *w1
	_, errMR := workflow.StartWorkflowRun(context.TODO(), db, api.Cache, *proj, wrCreate, &sdk.WorkflowRunPostHandlerOption{
		Manual: &sdk.WorkflowNodeRunManual{
			Username: u.GetUsername(),
		},
	}, consumer, nil)
	if errMR != nil {
		require.NoError(t, errMR)
	}

	_, errMR2 := workflow.StartWorkflowRun(context.TODO(), db, api.Cache, *proj, wrCreate, &sdk.WorkflowRunPostHandlerOption{
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
	api, db, router, end := newTestAPI(t)
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
	require.NoError(t, pipeline.InsertPipeline(api.mustDB(), &pip))

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
	require.NoError(t, pipeline.InsertPipeline(api.mustDB(), &pip2))
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

	proj2, errP := project.Load(api.mustDB(), proj.Key, project.LoadOptions.WithPipelines, project.LoadOptions.WithGroups, project.LoadOptions.WithIntegrations)
	require.NoError(t, errP)

	require.NoError(t, workflow.Insert(context.TODO(), api.mustDB(), api.Cache, *proj2, &w))
	w1, err := workflow.Load(context.TODO(), api.mustDB(), api.Cache, *proj, "test_1", workflow.LoadOptions{})
	require.NoError(t, err)

	for i := 0; i < 10; i++ {
		wr, err := workflow.CreateRun(db, w1, nil, u)
		assert.NoError(t, err)
		wr.Workflow = *w1
		_, err = workflow.StartWorkflowRun(context.TODO(), db, api.Cache, *proj, wr, &sdk.WorkflowRunPostHandlerOption{
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
	api, db, router, end := newTestAPI(t)
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
	require.NoError(t, pipeline.InsertPipeline(api.mustDB(), &pip))

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
	require.NoError(t, pipeline.InsertPipeline(api.mustDB(), &pip2))
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

	proj2, errP := project.Load(api.mustDB(), proj.Key, project.LoadOptions.WithPipelines, project.LoadOptions.WithGroups, project.LoadOptions.WithIntegrations)
	require.NoError(t, errP)

	require.NoError(t, workflow.Insert(context.TODO(), api.mustDB(), api.Cache, *proj2, &w))
	w1, err := workflow.Load(context.TODO(), api.mustDB(), api.Cache, *proj, "test_1", workflow.LoadOptions{})
	require.NoError(t, err)

	wr, err := workflow.CreateRun(db, w1, nil, u)
	assert.NoError(t, err)
	wr.Workflow = *w1
	_, err = workflow.StartWorkflowRun(context.TODO(), db, api.Cache, *proj, wr, &sdk.WorkflowRunPostHandlerOption{
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
	api, db, router, end := newTestAPI(t)
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
	require.NoError(t, pipeline.InsertPipeline(api.mustDB(), &pip))

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
	require.NoError(t, pipeline.InsertPipeline(api.mustDB(), &pip2))
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

	proj2, errP := project.Load(api.mustDB(), proj.Key, project.LoadOptions.WithPipelines, project.LoadOptions.WithGroups, project.LoadOptions.WithIntegrations)
	require.NoError(t, errP)

	require.NoError(t, workflow.Insert(context.TODO(), api.mustDB(), api.Cache, *proj2, &w))
	w1, err := workflow.Load(context.TODO(), api.mustDB(), api.Cache, *proj, "test_1", workflow.LoadOptions{})
	require.NoError(t, err)

	for i := 0; i < 10; i++ {
		wr, err := workflow.CreateRun(db, w1, nil, u)
		wr.Workflow = *w1
		assert.NoError(t, err)
		_, err = workflow.StartWorkflowRun(context.TODO(), db, api.Cache, *proj, wr, &sdk.WorkflowRunPostHandlerOption{
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
	api, db, router, end := newTestAPI(t)
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
	require.NoError(t, pipeline.InsertPipeline(api.mustDB(), &pip))

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
	require.NoError(t, pipeline.InsertPipeline(api.mustDB(), &pip2))
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

	proj2, errP := project.Load(api.mustDB(), proj.Key, project.LoadOptions.WithPipelines, project.LoadOptions.WithGroups, project.LoadOptions.WithIntegrations)
	require.NoError(t, errP)

	require.NoError(t, workflow.Insert(context.TODO(), api.mustDB(), api.Cache, *proj2, &w))
	w1, err := workflow.Load(context.TODO(), api.mustDB(), api.Cache, *proj, "test_1", workflow.LoadOptions{})
	require.NoError(t, err)

	for i := 0; i < 10; i++ {
		wr, err := workflow.CreateRun(db, w1, nil, u)
		assert.NoError(t, err)
		wr.Workflow = *w1
		_, err = workflow.StartWorkflowRun(context.TODO(), db, api.Cache, *proj, wr, &sdk.WorkflowRunPostHandlerOption{
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
	api, db, router, end := newTestAPI(t)
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
	require.NoError(t, application.Insert(db, *proj, &app))

	//First pipeline
	pip := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip1",
	}
	require.NoError(t, pipeline.InsertPipeline(api.mustDB(), &pip))

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
	require.NoError(t, pipeline.InsertPipeline(api.mustDB(), &pip2))
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
		WorkflowData: sdk.WorkflowData{
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

	proj2, errP := project.Load(api.mustDB(), proj.Key, project.LoadOptions.WithPipelines, project.LoadOptions.WithGroups, project.LoadOptions.WithIntegrations, project.LoadOptions.WithApplications)
	require.NoError(t, errP)

	require.NoError(t, workflow.Insert(context.TODO(), api.mustDB(), api.Cache, *proj2, &w))
	w1, err := workflow.Load(context.TODO(), api.mustDB(), api.Cache, *proj, "test_1", workflow.LoadOptions{})
	require.NoError(t, err)

	wr, err := workflow.CreateRun(db, w1, nil, u)
	assert.NoError(t, err)
	wr.Workflow = *w1
	_, err = workflow.StartWorkflowRun(context.TODO(), db, api.Cache, *proj2, wr, &sdk.WorkflowRunPostHandlerOption{
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

func Test_postWorkflowRunHandler(t *testing.T) {
	api, db, router, end := newTestAPI(t)
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
	require.NoError(t, pipeline.InsertPipeline(api.mustDB(), &pip))

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
	require.NoError(t, pipeline.InsertPipeline(api.mustDB(), &pip2))
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

	proj2, errP := project.Load(api.mustDB(), proj.Key, project.LoadOptions.WithPipelines, project.LoadOptions.WithGroups, project.LoadOptions.WithIntegrations)
	require.NoError(t, errP)

	require.NoError(t, workflow.Insert(context.TODO(), api.mustDB(), api.Cache, *proj2, &w))
	w1, err := workflow.Load(context.TODO(), api.mustDB(), api.Cache, *proj, "test_1", workflow.LoadOptions{})
	require.NoError(t, err)

	//Prepare request
	vars := map[string]string{
		"key":              proj.Key,
		"permWorkflowName": w1.Name,
	}
	uri := router.GetRoute("POST", api.postWorkflowRunHandler, vars)
	test.NotEmpty(t, uri)

	opts := &sdk.WorkflowRunPostHandlerOption{
		Manual: &sdk.WorkflowNodeRunManual{
			Payload: map[string]string{
				"test": "hereismytest",
			},
		},
	}
	req := assets.NewAuthentifiedRequest(t, u, pass, "POST", uri, opts)

	//Do the request
	rec := httptest.NewRecorder()
	router.Mux.ServeHTTP(rec, req)
	assert.Equal(t, 202, rec.Code)

	wr := &sdk.WorkflowRun{}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), wr))
	assert.Equal(t, int64(1), wr.Number)

	// wait for the workflow to finish crafting
	assert.NoError(t, waitCraftinWorkflow(t, db, wr.ID))

	lastRun, err := workflow.LoadLastRun(api.mustDB(), proj.Key, w1.Name, workflow.LoadRunOptions{})
	test.NoError(t, err)
	assert.NotNil(t, lastRun.RootRun())
	payloadCount := 0
	testFound := false
	for _, param := range lastRun.RootRun().BuildParameters {
		if param.Name == "payload" {
			payloadCount++
		} else if param.Name == "test" {
			testFound = true
		}
	}

	assert.Equal(t, 1, payloadCount)
	assert.True(t, testFound, "should find 'test' in build parameters")
}

func waitCraftinWorkflow(t *testing.T, db gorp.SqlExecutor, id int64) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	tick := time.NewTicker(100 * time.Millisecond)
	defer tick.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-tick.C:
			w, _ := workflow.LoadRunByID(db, id, workflow.LoadRunOptions{})
			if w == nil {
				continue
			}
			if w.Status == sdk.StatusPending {
				continue
			}
			return nil
		}
	}

}

/**
 * This test does
 * 1. Create worklow
 * 2. Migrate as code => this will create PR.id = 1
 * 3. Run workflow :  Must fail on getting PR.id = 1
 */
func Test_postWorkflowRunAsyncFailedHandler(t *testing.T) {
	api, db, router, end := newTestAPI(t)
	defer end()
	u, pass := assets.InsertAdminUser(t, api.mustDB())
	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, api.Cache, key, key)

	// Clean ascode event
	evts, _ := ascode.LoadAsCodeEventByRepo(context.TODO(), db, "ssh:/cloneurl")
	for _, e := range evts {
		_ = ascode.DeleteAsCodeEvent(db, e) // nolint
	}

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
	require.NoError(t, pipeline.InsertPipeline(api.mustDB(), &pip))

	// Create Application
	app := sdk.Application{
		Name:               sdk.RandomString(10),
		ProjectID:          proj.ID,
		RepositoryFullname: "foo/myrepo",
		VCSServer:          "github",
		RepositoryStrategy: sdk.RepositoryStrategy{
			ConnectionType: "ssh",
		},
	}
	assert.NoError(t, application.Insert(db, *proj, &app))
	assert.NoError(t, repositoriesmanager.InsertForApplication(db, &app, proj.Key))

	w := sdk.Workflow{
		Name:       "test_1",
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		WorkflowData: sdk.WorkflowData{
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

	proj2, errP := project.Load(api.mustDB(), proj.Key, project.LoadOptions.WithApplications, project.LoadOptions.WithPipelines, project.LoadOptions.WithGroups, project.LoadOptions.WithIntegrations)
	require.NoError(t, errP)

	require.NoError(t, workflow.Insert(context.TODO(), api.mustDB(), api.Cache, *proj2, &w))
	w1, err := workflow.Load(context.TODO(), api.mustDB(), api.Cache, *proj, "test_1", workflow.LoadOptions{})
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
			case "/vcs/github/repos/foo/myrepo":
				r := sdk.VCSRepo{
					SSHCloneURL:  "ssh:/cloneurl",
					HTTPCloneURL: "http:/cloneurl",
				}
				if err := enc.Encode(r); err != nil {
					return writeError(w, err)
				}
			case "/vcs/github/repos/foo/myrepo/hooks":
				h := sdk.VCSHook{}
				h.Name = "hook"
				if err := enc.Encode(h); err != nil {
					return writeError(w, err)
				}
			case "/vcs/github/repos/foo/myrepo/pullrequests":
				if r.Method == http.MethodGet {
					vcsPRs := []sdk.VCSPullRequest{}
					if err := enc.Encode(vcsPRs); err != nil {
						return writeError(w, err)
					}
				} else {
					pr := sdk.VCSPullRequest{
						Title: "blabla",
						URL:   "myurl",
						ID:    1,
					}
					if err := enc.Encode(pr); err != nil {
						return writeError(w, err)
					}
				}
			case "/vcs/github/repos/foo/myrepo/pullrequests/1":
				return writeError(w, fmt.Errorf("error for test"))

			case "/task/bulk":
				hooks := map[string]sdk.NodeHook{}
				hooks["123"] = sdk.NodeHook{
					UUID: "123",
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
				return writeError(w, fmt.Errorf("route %s must not be called", r.URL.String()))
			}
			return w, nil
		},
	)

	// Migrate as code
	ope := sdk.Operation{
		UUID: "123",
		Setup: sdk.OperationSetup{
			Push: sdk.OperationPush{
				Update: false,
			},
		},
	}
	ed := ascode.EntityData{
		FromRepo:      "ssh:/cloneurl",
		Name:          w1.Name,
		ID:            w1.ID,
		Type:          ascode.WorkflowEvent,
		OperationUUID: ope.UUID,
	}

	x := ascode.UpdateAsCodeResult(context.TODO(), api.mustDB(), api.Cache, *proj, app, ed, u)
	require.NotNil(t, x, "ascodeEvent should not be nil, but it was")

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
		assert.NoError(t, json.Unmarshal(recGetBody, &wrGet))

		if sdk.StatusIsTerminated(wrGet.Status) {
			t.Logf("%+v", wrGet)
			assert.Equal(t, sdk.StatusFail, wrGet.Status)
			assert.Equal(t, 1, len(wrGet.Infos))
			if len(wrGet.Infos) == 1 {
				assert.Equal(t, wrGet.Infos[0].Message.ID, sdk.MsgWorkflowError.ID)
			}
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
	api, db, router, end := newTestAPI(t)
	defer end()
	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, api.Cache, key, key)

	//First pipeline
	pip := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip1",
	}
	require.NoError(t, pipeline.InsertPipeline(api.mustDB(), &pip))

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
	require.NoError(t, pipeline.InsertPipeline(api.mustDB(), &pip2))
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
	require.NoError(t, group.Insert(context.TODO(), api.mustDB(), &gr))

	uLambda, pass := assets.InsertLambdaUser(t, api.mustDB(), &gr)

	w := sdk.Workflow{
		Name:       "test_1",
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		WorkflowData: sdk.WorkflowData{
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

	proj2, errP := project.Load(api.mustDB(), proj.Key, project.LoadOptions.WithPipelines, project.LoadOptions.WithGroups, project.LoadOptions.WithEnvironments)
	require.NoError(t, errP)

	require.NoError(t, workflow.Insert(context.TODO(), api.mustDB(), api.Cache, *proj2, &w))
	w1, err := workflow.Load(context.TODO(), api.mustDB(), api.Cache, *proj2, "test_1", workflow.LoadOptions{})
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

func Test_postWorkflowRunHandlerWithoutRightConditionsOnHook(t *testing.T) {
	api, db, router, end := newTestAPI(t)
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
	test.NoError(t, pipeline.InsertPipeline(api.mustDB(), &pip))

	s := sdk.NewStage("stage 1")
	s.Enabled = true
	s.PipelineID = pip.ID
	test.NoError(t, pipeline.InsertStage(api.mustDB(), s))
	j := &sdk.Job{
		Enabled: true,
		Action: sdk.Action{
			Enabled: true,
		},
	}
	test.NoError(t, pipeline.InsertJob(api.mustDB(), j, s.ID, &pip))
	s.Jobs = append(s.Jobs, *j)

	pip.Stages = append(pip.Stages, *s)

	//Second pipeline
	pip2 := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip2",
	}
	test.NoError(t, pipeline.InsertPipeline(api.mustDB(), &pip2))
	s = sdk.NewStage("stage 1")
	s.Enabled = true
	s.PipelineID = pip2.ID
	test.NoError(t, pipeline.InsertStage(api.mustDB(), s))
	j = &sdk.Job{
		Enabled: true,
		Action: sdk.Action{
			Enabled: true,
		},
	}
	test.NoError(t, pipeline.InsertJob(api.mustDB(), j, s.ID, &pip2))
	s.Jobs = append(s.Jobs, *j)

	mockHookService, _ := assets.InsertService(t, db, "Test_postWorkflowRunHandlerWithoutRightConditionsOnHook", services.TypeHooks)
	defer func() {
		_ = services.Delete(db, mockHookService) // nolint
	}()

	//This is a mock for the repositories service
	services.HTTPClient = mock(
		func(r *http.Request) (*http.Response, error) {
			body := new(bytes.Buffer)
			w := new(http.Response)
			enc := json.NewEncoder(body)
			w.Body = ioutil.NopCloser(body)

			switch r.URL.String() {
			case "/task/bulk":
				hooks := map[string]sdk.NodeHook{}
				hooks["1cbf3792-126b-4111-884f-077bdee9523c"] = sdk.NodeHook{
					Conditions: sdk.WorkflowNodeConditions{
						LuaScript: "return false",
					},
					HookModelName: sdk.WebHookModel.Name,
					Config:        sdk.WebHookModel.DefaultConfig.Clone(),
					UUID:          "1cbf3792-126b-4111-884f-077bdee9523c",
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

	_, errDb := db.Exec("DELETE FROM w_node_hook WHERE uuid = $1", "1cbf3792-126b-4111-884f-077bdee9523c")
	test.NoError(t, errDb)

	w := sdk.Workflow{
		Name:       "test_1",
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		HookModels: map[int64]sdk.WorkflowHookModel{
			1: sdk.WebHookModel,
		},
		WorkflowData: sdk.WorkflowData{
			Node: sdk.Node{
				Name: "root",
				Type: sdk.NodeTypePipeline,
				Context: &sdk.NodeContext{
					PipelineID: pip.ID,
				},
				Hooks: []sdk.NodeHook{
					{
						Conditions: sdk.WorkflowNodeConditions{
							LuaScript: "return false",
						},
						HookModelName: sdk.WebHookModel.Name,
						Config:        sdk.WebHookModel.DefaultConfig.Clone(),
						UUID:          "1cbf3792-126b-4111-884f-077bdee9523c",
					},
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

	proj2, errP := project.Load(api.mustDB(), proj.Key, project.LoadOptions.WithPipelines, project.LoadOptions.WithGroups, project.LoadOptions.WithEnvironments)
	test.NoError(t, errP)

	test.NoError(t, workflow.Insert(context.TODO(), api.mustDB(), api.Cache, *proj2, &w))
	w1, err := workflow.Load(context.TODO(), api.mustDB(), api.Cache, *proj2, "test_1", workflow.LoadOptions{})
	test.NoError(t, err)

	//Prepare request
	vars := map[string]string{
		"key":              proj.Key,
		"permWorkflowName": w1.Name,
	}
	uri := router.GetRoute("POST", api.postWorkflowRunHandler, vars)
	test.NotEmpty(t, uri)

	opts := &sdk.WorkflowRunPostHandlerOption{
		Hook: &sdk.WorkflowNodeRunHookEvent{
			Payload:              nil,
			WorkflowNodeHookUUID: "1cbf3792-126b-4111-884f-077bdee9523c",
		},
	}
	req := assets.NewAuthentifiedRequest(t, u, pass, "POST", uri, opts)

	//Do the request
	rec := httptest.NewRecorder()
	router.Mux.ServeHTTP(rec, req)
	var body []byte
	_, err = req.Body.Read(body)
	test.NoError(t, err)
	defer req.Body.Close()
	assert.Equal(t, 400, rec.Code)
}

func Test_postWorkflowRunHandlerHookWithMutex(t *testing.T) {
	api, db, router, end := newTestAPI(t)
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
	test.NoError(t, pipeline.InsertPipeline(api.mustDB(), &pip))

	s := sdk.NewStage("stage 1")
	s.Enabled = true
	s.PipelineID = pip.ID
	test.NoError(t, pipeline.InsertStage(api.mustDB(), s))
	j := &sdk.Job{
		Enabled: true,
		Action: sdk.Action{
			Enabled: true,
		},
	}
	test.NoError(t, pipeline.InsertJob(api.mustDB(), j, s.ID, &pip))
	s.Jobs = append(s.Jobs, *j)

	pip.Stages = append(pip.Stages, *s)

	//Second pipeline
	pip2 := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip2",
	}
	test.NoError(t, pipeline.InsertPipeline(api.mustDB(), &pip2))
	s = sdk.NewStage("stage 1")
	s.Enabled = true
	s.PipelineID = pip2.ID
	test.NoError(t, pipeline.InsertStage(api.mustDB(), s))
	j = &sdk.Job{
		Enabled: true,
		Action: sdk.Action{
			Enabled: true,
		},
	}
	test.NoError(t, pipeline.InsertJob(api.mustDB(), j, s.ID, &pip2))
	s.Jobs = append(s.Jobs, *j)

	mockServiceHook, _ := assets.InsertService(t, db, "Test_postWorkflowRunHandlerHookWithMutex", services.TypeHooks)
	defer func() {
		_ = services.Delete(db, mockServiceHook) // nolint
	}()

	//This is a mock for the repositories service
	services.HTTPClient = mock(
		func(r *http.Request) (*http.Response, error) {
			body := new(bytes.Buffer)
			w := new(http.Response)
			enc := json.NewEncoder(body)
			w.Body = ioutil.NopCloser(body)

			switch r.URL.String() {
			case "/task/bulk":
				hooks := map[string]sdk.NodeHook{}
				hooks["1cbf3792-126b-4111-884f-077bdee9523d"] = sdk.NodeHook{
					HookModelName: sdk.WebHookModel.Name,
					Config:        sdk.WebHookModel.DefaultConfig.Clone(),
					UUID:          "1cbf3792-126b-4111-884f-077bdee9523d",
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

	_, errDb := db.Exec("DELETE FROM w_node_hook WHERE uuid = $1", "1cbf3792-126b-4111-884f-077bdee9523d")
	test.NoError(t, errDb)

	w := sdk.Workflow{
		Name:       "test_1",
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		HookModels: map[int64]sdk.WorkflowHookModel{
			1: sdk.WebHookModel,
		},
		WorkflowData: sdk.WorkflowData{
			Node: sdk.Node{
				Name: "root",
				Type: sdk.NodeTypePipeline,
				Context: &sdk.NodeContext{
					PipelineID: pip.ID,
					Mutex:      true,
				},
				Hooks: []sdk.NodeHook{
					{
						HookModelName: sdk.WebHookModel.Name,
						Config:        sdk.WebHookModel.DefaultConfig.Clone(),
						UUID:          "1cbf3792-126b-4111-884f-077bdee9523d",
					},
				},
			},
		},
	}

	proj2, errP := project.Load(api.mustDB(), proj.Key, project.LoadOptions.WithPipelines, project.LoadOptions.WithGroups, project.LoadOptions.WithEnvironments)
	test.NoError(t, errP)

	test.NoError(t, workflow.Insert(context.TODO(), api.mustDB(), api.Cache, *proj2, &w))
	w1, err := workflow.Load(context.TODO(), api.mustDB(), api.Cache, *proj2, "test_1", workflow.LoadOptions{})
	test.NoError(t, err)

	//Prepare request
	vars := map[string]string{
		"key":              proj.Key,
		"permWorkflowName": w1.Name,
	}
	uri := router.GetRoute("POST", api.postWorkflowRunHandler, vars)
	test.NotEmpty(t, uri)

	opts := &sdk.WorkflowRunPostHandlerOption{
		Hook: &sdk.WorkflowNodeRunHookEvent{
			Payload:              nil,
			WorkflowNodeHookUUID: "1cbf3792-126b-4111-884f-077bdee9523d",
		},
	}
	req := assets.NewAuthentifiedRequest(t, u, pass, "POST", uri, opts)

	//Do the request, start first workflow
	rec := httptest.NewRecorder()
	router.Mux.ServeHTTP(rec, req)
	var body []byte
	_, err = req.Body.Read(body)
	test.NoError(t, err)
	defer req.Body.Close()
	assert.Equal(t, 202, rec.Code)

	req2 := assets.NewAuthentifiedRequest(t, u, pass, "POST", uri, opts)

	//Do the request, start a new run
	rec2 := httptest.NewRecorder()
	router.Mux.ServeHTTP(rec2, req2)
	var body2 []byte
	_, err = req2.Body.Read(body2)
	test.NoError(t, err)
	defer req2.Body.Close()
	assert.Equal(t, 202, rec2.Code)

	// it's an async call, wait a bit the let cds take care of the previous request
	time.Sleep(3 * time.Second)

	lastRun, err := workflow.LoadLastRun(api.mustDB(), proj.Key, w1.Name, workflow.LoadRunOptions{})
	test.NoError(t, err)
	assert.Equal(t, int64(2), lastRun.Number)
	assert.Equal(t, sdk.StatusBuilding, lastRun.Status)
}

func Test_postWorkflowRunHandlerHook(t *testing.T) {
	api, db, router, end := newTestAPI(t)
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
	test.NoError(t, pipeline.InsertPipeline(api.mustDB(), &pip))

	s := sdk.NewStage("stage 1")
	s.Enabled = true
	s.PipelineID = pip.ID
	test.NoError(t, pipeline.InsertStage(api.mustDB(), s))
	j := &sdk.Job{
		Enabled: true,
		Action: sdk.Action{
			Enabled: true,
		},
	}
	test.NoError(t, pipeline.InsertJob(api.mustDB(), j, s.ID, &pip))
	s.Jobs = append(s.Jobs, *j)

	pip.Stages = append(pip.Stages, *s)

	//Second pipeline
	pip2 := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip2",
	}
	test.NoError(t, pipeline.InsertPipeline(api.mustDB(), &pip2))
	s = sdk.NewStage("stage 1")
	s.Enabled = true
	s.PipelineID = pip2.ID
	test.NoError(t, pipeline.InsertStage(api.mustDB(), s))
	j = &sdk.Job{
		Enabled: true,
		Action: sdk.Action{
			Enabled: true,
		},
	}
	test.NoError(t, pipeline.InsertJob(api.mustDB(), j, s.ID, &pip2))
	s.Jobs = append(s.Jobs, *j)

	mockServiceHook, _ := assets.InsertService(t, db, "Test_postWorkflowRunHandlerHookWithMutex", services.TypeHooks)
	defer func() {
		_ = services.Delete(db, mockServiceHook) // nolint
	}()

	//This is a mock for the hook service
	services.HTTPClient = mock(
		func(r *http.Request) (*http.Response, error) {
			body := new(bytes.Buffer)
			w := new(http.Response)
			enc := json.NewEncoder(body)
			w.Body = ioutil.NopCloser(body)

			switch r.URL.String() {
			case "/task/bulk":
				hooks := map[string]sdk.NodeHook{}
				hooks["1cbf3792-126b-4111-884f-077bdee9523d"] = sdk.NodeHook{
					HookModelName: sdk.WebHookModel.Name,
					Config:        sdk.WebHookModel.DefaultConfig.Clone(),
					UUID:          "1cbf3792-126b-4111-884f-077bdee9523d",
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

	_, errDb := db.Exec("DELETE FROM w_node_hook WHERE uuid = $1", "1cbf3792-126b-4111-884f-077bdee9523d")
	test.NoError(t, errDb)

	w := sdk.Workflow{
		Name:       "test_1",
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		HookModels: map[int64]sdk.WorkflowHookModel{
			1: sdk.WebHookModel,
		},
		WorkflowData: sdk.WorkflowData{
			Node: sdk.Node{
				Name: "root",
				Type: sdk.NodeTypePipeline,
				Context: &sdk.NodeContext{
					PipelineID: pip.ID,
				},
				Hooks: []sdk.NodeHook{
					{
						HookModelName: sdk.WebHookModel.Name,
						Config:        sdk.WebHookModel.DefaultConfig.Clone(),
						UUID:          "1cbf3792-126b-4111-884f-077bdee9523d",
					},
				},
			},
		},
	}

	proj2, errP := project.Load(api.mustDB(), proj.Key, project.LoadOptions.WithPipelines, project.LoadOptions.WithGroups, project.LoadOptions.WithEnvironments)
	test.NoError(t, errP)

	test.NoError(t, workflow.Insert(context.TODO(), api.mustDB(), api.Cache, *proj2, &w))
	w1, err := workflow.Load(context.TODO(), api.mustDB(), api.Cache, *proj2, "test_1", workflow.LoadOptions{})
	test.NoError(t, err)

	//Prepare request
	vars := map[string]string{
		"key":              proj.Key,
		"permWorkflowName": w1.Name,
	}
	uri := router.GetRoute("POST", api.postWorkflowRunHandler, vars)
	test.NotEmpty(t, uri)

	opts := &sdk.WorkflowRunPostHandlerOption{
		Hook: &sdk.WorkflowNodeRunHookEvent{
			Payload: map[string]string{
				"test":    "mypayload",
				"payload": `{"raw": "value"}`,
			},
			WorkflowNodeHookUUID: "1cbf3792-126b-4111-884f-077bdee9523d",
		},
	}
	req := assets.NewAuthentifiedRequest(t, u, pass, "POST", uri, opts)

	//Do the request, start first workflow
	rec := httptest.NewRecorder()
	router.Mux.ServeHTTP(rec, req)
	var body []byte
	_, err = req.Body.Read(body)
	test.NoError(t, err)
	defer req.Body.Close()
	assert.Equal(t, 202, rec.Code)
	wr := &sdk.WorkflowRun{}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), wr))
	assert.Equal(t, int64(1), wr.Number)

	assert.NoError(t, waitCraftinWorkflow(t, db, wr.ID))
	lastRun, err := workflow.LoadLastRun(api.mustDB(), proj.Key, w1.Name, workflow.LoadRunOptions{})
	test.NoError(t, err)
	assert.NotNil(t, lastRun.RootRun())
	payloadCount := 0
	rawFound := false
	testFound := false
	for _, param := range lastRun.RootRun().BuildParameters {
		if param.Name == "payload" {
			payloadCount++
		} else if param.Name == "raw" {
			rawFound = true
		} else if param.Name == "test" {
			testFound = true
		}
	}

	assert.Equal(t, 1, payloadCount)
	assert.False(t, rawFound, "should not find 'raw' in build parameters")
	assert.True(t, testFound, "should find 'test' in build parameters")
}

func Test_postWorkflowRunHandler_Forbidden(t *testing.T) {
	api, db, router, end := newTestAPI(t)
	defer end()
	u, pass := assets.InsertAdminUser(t, api.mustDB())
	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, api.Cache, key, key)

	gr := &sdk.Group{
		Name: sdk.RandomString(10),
	}
	require.NoError(t, group.Insert(context.TODO(), db, gr))
	require.NoError(t, group.InsertLinkGroupProject(context.TODO(), api.mustDB(), &group.LinkGroupProject{
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
	require.NoError(t, pipeline.InsertPipeline(api.mustDB(), &pip))

	env := &sdk.Environment{
		Name:       sdk.RandomString(10),
		ProjectKey: proj.Key,
		ProjectID:  proj.ID,
	}
	require.NoError(t, environment.InsertEnvironment(api.mustDB(), env))

	proj2, errp := project.Load(api.mustDB(), proj.Key, project.LoadOptions.WithPipelines, project.LoadOptions.WithEnvironments)
	require.NoError(t, errp)

	w := sdk.Workflow{
		Name:       "test_1",
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		WorkflowData: sdk.WorkflowData{
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

	require.NoError(t, workflow.Insert(context.TODO(), api.mustDB(), api.Cache, *proj2, &w))

	u.Ring = ""
	require.NoError(t, user.Update(context.TODO(), api.mustDB(), u))

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

func Test_postWorkflowRunHandler_ConditionNotOK(t *testing.T) {
	api, db, router, end := newTestAPI(t)
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
	test.NoError(t, pipeline.InsertPipeline(api.mustDB(), &pip))

	env := &sdk.Environment{
		Name:       sdk.RandomString(10),
		ProjectKey: proj.Key,
		ProjectID:  proj.ID,
	}
	test.NoError(t, environment.InsertEnvironment(api.mustDB(), env))

	proj2, errp := project.Load(api.mustDB(), proj.Key, project.LoadOptions.WithPipelines, project.LoadOptions.WithEnvironments)
	test.NoError(t, errp)

	w := sdk.Workflow{
		Name:       "test_1",
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		WorkflowData: sdk.WorkflowData{
			Node: sdk.Node{
				Name: "root",
				Type: sdk.NodeTypePipeline,
				Context: &sdk.NodeContext{
					PipelineID:    pip.ID,
					EnvironmentID: env.ID,
					Conditions: sdk.WorkflowNodeConditions{
						LuaScript: "return false",
					},
				},
			},
		},
	}

	test.NoError(t, workflow.Insert(context.TODO(), api.mustDB(), api.Cache, *proj2, &w))

	//Prepare request
	vars := map[string]string{
		"key":              proj.Key,
		"permWorkflowName": w.Name,
	}
	uri := router.GetRoute("POST", api.postWorkflowRunHandler, vars)
	test.NotEmpty(t, uri)

	opts := &sdk.WorkflowRunPostHandlerOption{
		Manual: &sdk.WorkflowNodeRunManual{
			Payload: map[string]string{"foo": "bar"},
		},
	}
	req := assets.NewAuthentifiedRequest(t, u, pass, "POST", uri, opts)

	//Do the request
	rec := httptest.NewRecorder()
	router.Mux.ServeHTTP(rec, req)

	assert.Equal(t, 202, rec.Code)

	// it's an async call, wait a bit the let cds take care of the previous request
	time.Sleep(3 * time.Second)

	lastRun, err := workflow.LoadLastRun(api.mustDB(), proj.Key, w.Name, workflow.LoadRunOptions{})
	test.NoError(t, err)
	assert.Equal(t, int64(1), lastRun.Number)
	assert.Equal(t, sdk.StatusNeverBuilt, lastRun.Status)
	// check "Run conditions aren't ok" info
	var found bool
	for _, info := range lastRun.Infos {
		if info.Message.ID == sdk.MsgWorkflowConditionError.ID {
			found = true
		}
	}
	assert.Equal(t, true, found)
}

func Test_postWorkflowRunHandler_BadPayload(t *testing.T) {
	api, db, router, end := newTestAPI(t)
	defer end()
	u, pass := assets.InsertAdminUser(t, api.mustDB())
	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, api.Cache, key, key)

	gr := &sdk.Group{
		Name: sdk.RandomString(10),
	}
	require.NoError(t, group.Insert(context.TODO(), db, gr))
	require.NoError(t, group.InsertLinkGroupProject(context.TODO(), api.mustDB(), &group.LinkGroupProject{
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
	require.NoError(t, pipeline.InsertPipeline(api.mustDB(), &pip))

	env := &sdk.Environment{
		Name:       sdk.RandomString(10),
		ProjectKey: proj.Key,
		ProjectID:  proj.ID,
	}
	require.NoError(t, environment.InsertEnvironment(api.mustDB(), env))

	proj2, errp := project.Load(api.mustDB(), proj.Key, project.LoadOptions.WithPipelines, project.LoadOptions.WithEnvironments)
	require.NoError(t, errp)

	w := sdk.Workflow{
		Name:       "test_1",
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		WorkflowData: sdk.WorkflowData{
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

	require.NoError(t, workflow.Insert(context.TODO(), api.mustDB(), api.Cache, *proj2, &w))

	require.NoError(t, user.Update(context.TODO(), api.mustDB(), u))

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
	require.NoError(t, pipeline.InsertPipeline(api.mustDB(), &pip))

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
	require.NoError(t, pipeline.InsertPipeline(api.mustDB(), &pip2))
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

	proj2, errP := project.Load(api.mustDB(), proj.Key, project.LoadOptions.WithPipelines, project.LoadOptions.WithGroups, project.LoadOptions.WithIntegrations)
	require.NoError(t, errP)
	consumer, _ := authentication.LoadConsumerByTypeAndUserID(context.TODO(), db, sdk.ConsumerLocal, u.ID, authentication.LoadConsumerOptions.WithAuthentifiedUser)

	require.NoError(t, workflow.Insert(context.TODO(), api.mustDB(), api.Cache, *proj2, &w))
	w1, err := workflow.Load(context.TODO(), api.mustDB(), api.Cache, *proj, "test_1", workflow.LoadOptions{
		DeepPipeline: true,
	})
	require.NoError(t, err)

	wr, err := workflow.CreateRun(db, w1, nil, u)
	assert.NoError(t, err)
	wr.Workflow = *w1
	_, err = workflow.StartWorkflowRun(context.TODO(), db, api.Cache, *proj, wr, &sdk.WorkflowRunPostHandlerOption{
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
	api, db, router, end := newTestAPI(t)
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
	require.NoError(t, pipeline.InsertPipeline(api.mustDB(), &pip))

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
	require.NoError(t, pipeline.InsertPipeline(api.mustDB(), &pip2))
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

	consumer, _ := authentication.LoadConsumerByTypeAndUserID(context.TODO(), db, sdk.ConsumerLocal, u.ID, authentication.LoadConsumerOptions.WithAuthentifiedUser)

	proj2, errP := project.Load(api.mustDB(), proj.Key, project.LoadOptions.WithPipelines, project.LoadOptions.WithGroups, project.LoadOptions.WithIntegrations)
	require.NoError(t, errP)

	require.NoError(t, workflow.Insert(context.TODO(), api.mustDB(), api.Cache, *proj2, &w))
	w1, err := workflow.Load(context.TODO(), api.mustDB(), api.Cache, *proj, "test_1", workflow.LoadOptions{})
	require.NoError(t, err)

	wr, err := workflow.CreateRun(db, w1, nil, u)
	assert.NoError(t, err)
	wr.Workflow = *w1
	wr.Tag("git.branch", "master")
	assert.NoError(t, workflow.UpdateWorkflowRun(context.TODO(), api.mustDB(), wr))
	_, err = workflow.StartWorkflowRun(context.TODO(), db, api.Cache, *proj, wr, &sdk.WorkflowRunPostHandlerOption{
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

	session, err := authentication.NewSession(context.TODO(), db, serviceConsumer, 5*time.Minute, false)
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
	api, db, router, end := newTestAPI(t)
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
	require.NoError(t, pipeline.InsertPipeline(api.mustDB(), &pip))

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
	require.NoError(t, pipeline.InsertPipeline(api.mustDB(), &pip2))
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

	proj2, errP := project.Load(api.mustDB(), proj.Key, project.LoadOptions.WithPipelines, project.LoadOptions.WithGroups, project.LoadOptions.WithIntegrations)
	require.NoError(t, errP)

	require.NoError(t, workflow.Insert(context.TODO(), api.mustDB(), api.Cache, *proj2, &w))
	w1, err := workflow.Load(context.TODO(), api.mustDB(), api.Cache, *proj, "test_1", workflow.LoadOptions{})
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

func Test_postWorkflowRunHandlerBadResyncOptions(t *testing.T) {
	api, db, router, end := newTestAPI(t)
	defer end()

	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, api.Cache, key, key)
	w := assets.InsertTestWorkflow(t, db, api.Cache, proj, sdk.RandomString(10))
	u, pass := assets.InsertLambdaUser(t, api.mustDB(), &proj.ProjectGroups[0].Group)

	//Prepare request
	vars := map[string]string{
		"key":              proj.Key,
		"permWorkflowName": w.Name,
	}
	uri := router.GetRoute("POST", api.postWorkflowRunHandler, vars)
	test.NotEmpty(t, uri)

	opts := &sdk.WorkflowRunPostHandlerOption{
		Manual: &sdk.WorkflowNodeRunManual{
			OnlyFailedJobs: true,
			Resync:         true,
		},
	}
	req := assets.NewAuthentifiedRequest(t, u, pass, "POST", uri, opts)

	//Do the request
	rec := httptest.NewRecorder()
	router.Mux.ServeHTTP(rec, req)
	assert.Equal(t, 400, rec.Code)
}

func Test_postWorkflowRunHandlerRestartOnlyFailed(t *testing.T) {
	api, db, router, end := newTestAPI(t)
	defer end()
	u, pass := assets.InsertAdminUser(t, api.mustDB())
	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, api.Cache, key, key)

	pip := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip1",
	}
	require.NoError(t, pipeline.InsertPipeline(api.mustDB(), &pip))
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

	j2 := &sdk.Job{
		Enabled: true,
		Action: sdk.Action{
			Enabled: true,
		},
	}
	pipeline.InsertJob(api.mustDB(), j2, s.ID, &pip)
	s.Jobs = append(s.Jobs, *j2)

	pip.Stages = append(pip.Stages, *s)

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
			},
		},
	}

	proj2, errP := project.Load(api.mustDB(), proj.Key, project.LoadOptions.WithPipelines, project.LoadOptions.WithGroups, project.LoadOptions.WithIntegrations)
	require.NoError(t, errP)

	require.NoError(t, workflow.Insert(context.TODO(), api.mustDB(), api.Cache, *proj2, &w))
	w1, err := workflow.Load(context.TODO(), api.mustDB(), api.Cache, *proj, "test_1", workflow.LoadOptions{})
	require.NoError(t, err)

	//Prepare request
	vars := map[string]string{
		"key":              proj.Key,
		"permWorkflowName": w1.Name,
	}
	uri := router.GetRoute("POST", api.postWorkflowRunHandler, vars)
	test.NotEmpty(t, uri)

	opts := &sdk.WorkflowRunPostHandlerOption{
		Manual: &sdk.WorkflowNodeRunManual{
			OnlyFailedJobs: false,
			Resync:         false,
		},
	}
	req := assets.NewAuthentifiedRequest(t, u, pass, "POST", uri, opts)

	//Do the request
	rec := httptest.NewRecorder()
	router.Mux.ServeHTTP(rec, req)
	assert.Equal(t, 202, rec.Code)

	var wr sdk.WorkflowRun
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &wr))
	assert.Equal(t, int64(1), wr.Number)

	// wait for the workflow to finish crafting
	assert.NoError(t, waitCraftinWorkflow(t, db, wr.ID))

	wrr, _ := workflow.LoadRun(context.TODO(), db, proj2.Key, w1.Name, 1, workflow.LoadRunOptions{})
	assert.Equal(t, sdk.StatusBuilding, wrr.Status)

	// Update WORKFLOW RUN
	wrr.Status = sdk.StatusFail
	assert.NoError(t, workflow.UpdateWorkflowRun(context.TODO(), db, wrr))

	// Update WORKFLOW NODE RUN
	nr, err := workflow.LoadNodeRunByID(db, wrr.WorkflowNodeRuns[w1.WorkflowData.Node.ID][0].ID, workflow.LoadRunOptions{})
	assert.NoError(t, err)

	assert.NoError(t, workflow.DeleteNodeJobRuns(db, nr.ID))

	firstJobEnd := time.Now()
	nr.Status = sdk.StatusFail
	nr.Stages[0].Status = sdk.StatusFail
	nr.Stages[0].RunJobs = make([]sdk.WorkflowNodeJobRun, 2)
	nr.Stages[0].RunJobs[0] = sdk.WorkflowNodeJobRun{
		Start:  firstJobEnd,
		Done:   firstJobEnd,
		Status: sdk.StatusSuccess,
		Job: sdk.ExecutedJob{
			Job: pip.Stages[0].Jobs[0],
		},
	}

	nr.Stages[0].RunJobs[1] = sdk.WorkflowNodeJobRun{
		Start:  firstJobEnd,
		Done:   firstJobEnd,
		Status: sdk.StatusFail,
		Job: sdk.ExecutedJob{
			Job: pip.Stages[0].Jobs[1],
		},
	}
	assert.NoError(t, workflow.UpdateNodeRun(db, nr))

	opts = &sdk.WorkflowRunPostHandlerOption{
		Manual: &sdk.WorkflowNodeRunManual{
			OnlyFailedJobs: true,
			Resync:         false,
		},
		FromNodeIDs: []int64{w1.WorkflowData.Node.ID},
		Number:      &wrr.Number,
	}
	api.initWorkflowRun(context.TODO(), proj2.Key, &wrr.Workflow, wrr, opts, &sdk.AuthConsumer{
		AuthentifiedUser: u,
	})

	wrr, _ = workflow.LoadRun(context.TODO(), db, proj2.Key, w1.Name, 1, workflow.LoadRunOptions{})

	assert.Equal(t, sdk.StatusBuilding, wrr.Status)
	assert.Equal(t, firstJobEnd.Unix(), wrr.WorkflowNodeRuns[wrr.Workflow.WorkflowData.Node.ID][0].Stages[0].RunJobs[0].Start.Unix())
	assert.NotEqual(t, firstJobEnd, wrr.WorkflowNodeRuns[wrr.Workflow.WorkflowData.Node.ID][0].Stages[0].RunJobs[1].Start)
	assert.Equal(t, sdk.StatusSuccess, wrr.WorkflowNodeRuns[wrr.Workflow.WorkflowData.Node.ID][0].Stages[0].RunJobs[0].Status)
	assert.Equal(t, sdk.StatusWaiting, wrr.WorkflowNodeRuns[wrr.Workflow.WorkflowData.Node.ID][0].Stages[0].RunJobs[1].Status)
}

func Test_postWorkflowRunHandlerRestartResync(t *testing.T) {
	api, db, router, end := newTestAPI(t)
	defer end()
	u, pass := assets.InsertAdminUser(t, api.mustDB())
	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, api.Cache, key, key)

	pip := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip1",
	}
	require.NoError(t, pipeline.InsertPipeline(api.mustDB(), &pip))
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

	j2 := &sdk.Job{
		Enabled: true,
		Action: sdk.Action{
			Enabled: true,
		},
	}
	pipeline.InsertJob(api.mustDB(), j2, s.ID, &pip)
	s.Jobs = append(s.Jobs, *j2)

	pip.Stages = append(pip.Stages, *s)

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
			},
		},
	}

	proj2, errP := project.Load(api.mustDB(), proj.Key, project.LoadOptions.WithPipelines, project.LoadOptions.WithGroups, project.LoadOptions.WithIntegrations)
	require.NoError(t, errP)

	require.NoError(t, workflow.Insert(context.TODO(), api.mustDB(), api.Cache, *proj2, &w))
	w1, err := workflow.Load(context.TODO(), api.mustDB(), api.Cache, *proj, "test_1", workflow.LoadOptions{})
	require.NoError(t, err)

	//Prepare request
	vars := map[string]string{
		"key":              proj.Key,
		"permWorkflowName": w1.Name,
	}
	uri := router.GetRoute("POST", api.postWorkflowRunHandler, vars)
	test.NotEmpty(t, uri)

	opts := &sdk.WorkflowRunPostHandlerOption{
		Manual: &sdk.WorkflowNodeRunManual{
			OnlyFailedJobs: false,
			Resync:         false,
		},
	}
	req := assets.NewAuthentifiedRequest(t, u, pass, "POST", uri, opts)

	//Do the request
	rec := httptest.NewRecorder()
	router.Mux.ServeHTTP(rec, req)
	assert.Equal(t, 202, rec.Code)

	var wr sdk.WorkflowRun
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &wr))
	assert.Equal(t, int64(1), wr.Number)

	// wait for the workflow to finish crafting
	assert.NoError(t, waitCraftinWorkflow(t, db, wr.ID))

	wrr, _ := workflow.LoadRun(context.TODO(), db, proj2.Key, w1.Name, 1, workflow.LoadRunOptions{})
	assert.Equal(t, sdk.StatusBuilding, wrr.Status)
	assert.Equal(t, 2, len(wrr.Workflow.Pipelines[pip.ID].Stages[0].Jobs))

	// Update WORKFLOW RUN
	wrr.Status = sdk.StatusFail
	assert.NoError(t, workflow.UpdateWorkflowRun(context.TODO(), db, wrr))

	// Update WORKFLOW NODE RUN
	nr, err := workflow.LoadNodeRunByID(db, wrr.WorkflowNodeRuns[w1.WorkflowData.Node.ID][0].ID, workflow.LoadRunOptions{})
	assert.NoError(t, err)

	assert.NoError(t, workflow.DeleteNodeJobRuns(db, nr.ID))

	firstJobEnd := time.Now()
	nr.Status = sdk.StatusFail
	nr.Stages[0].Status = sdk.StatusFail
	nr.Stages[0].RunJobs = make([]sdk.WorkflowNodeJobRun, 2)
	nr.Stages[0].RunJobs[0] = sdk.WorkflowNodeJobRun{
		Start:  firstJobEnd,
		Done:   firstJobEnd,
		Status: sdk.StatusSuccess,
		Job: sdk.ExecutedJob{
			Job: pip.Stages[0].Jobs[0],
		},
	}

	nr.Stages[0].RunJobs[1] = sdk.WorkflowNodeJobRun{
		Start:  firstJobEnd,
		Done:   firstJobEnd,
		Status: sdk.StatusFail,
		Job: sdk.ExecutedJob{
			Job: pip.Stages[0].Jobs[1],
		},
	}
	assert.NoError(t, workflow.UpdateNodeRun(db, nr))

	// Update pipeline
	j3 := &sdk.Job{
		Enabled: true,
		Action: sdk.Action{
			Enabled: true,
		},
	}
	pipeline.InsertJob(api.mustDB(), j3, s.ID, &pip)

	uri = router.GetRoute("POST", api.postWorkflowRunHandler, vars)
	test.NotEmpty(t, uri)

	opts = &sdk.WorkflowRunPostHandlerOption{
		Manual: &sdk.WorkflowNodeRunManual{
			OnlyFailedJobs: false,
			Resync:         true,
		},
		Number:      &wrr.Number,
		FromNodeIDs: []int64{w1.WorkflowData.Node.ID},
	}
	req = assets.NewAuthentifiedRequest(t, u, pass, "POST", uri, opts)

	//Do the request
	rec = httptest.NewRecorder()
	router.Mux.ServeHTTP(rec, req)
	assert.Equal(t, 202, rec.Code)

	var wrResync sdk.WorkflowRun
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &wrResync))

	wrrResyncDB, err := workflow.LoadRun(context.TODO(), db, proj2.Key, w1.Name, wrResync.Number, workflow.LoadRunOptions{})
	assert.NoError(t, err)
	assert.Equal(t, 3, len(wrrResyncDB.Workflow.Pipelines[pip.ID].Stages[0].Jobs))
}
