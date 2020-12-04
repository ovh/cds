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
	"github.com/ovh/cds/engine/api/ascode"
	"github.com/ovh/cds/engine/api/authentication"
	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/api/environment"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/integration"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/plugin"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/purge"
	"github.com/ovh/cds/engine/api/repositoriesmanager"
	"github.com/ovh/cds/engine/api/services"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/engine/api/user"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/engine/featureflipping"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
)

func Test_getWorkflowNodeRunHistoryHandler(t *testing.T) {
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

	proj2, errP := project.Load(context.TODO(), api.mustDB(), proj.Key, project.LoadOptions.WithPipelines, project.LoadOptions.WithGroups, project.LoadOptions.WithIntegrations)
	require.NoError(t, errP)

	require.NoError(t, workflow.Insert(context.TODO(), db, api.Cache, *proj2, &w))
	w1, err := workflow.Load(context.TODO(), db, api.Cache, *proj, "test_1", workflow.LoadOptions{})
	require.NoError(t, err)

	wrCreate, err := workflow.CreateRun(api.mustDB(), w1, sdk.WorkflowRunPostHandlerOption{AuthConsumerID: consumer.ID})
	assert.NoError(t, err)
	wrCreate.Workflow = *w1
	_, errMR := workflow.StartWorkflowRun(context.TODO(), db, api.Cache, *proj, wrCreate, &sdk.WorkflowRunPostHandlerOption{
		Manual: &sdk.WorkflowNodeRunManual{
			Username: u.GetUsername(),
		},
	}, *consumer, nil)
	if errMR != nil {
		require.NoError(t, errMR)
	}

	_, errMR2 := workflow.StartWorkflowRun(context.TODO(), db, api.Cache, *proj, wrCreate, &sdk.WorkflowRunPostHandlerOption{
		Manual: &sdk.WorkflowNodeRunManual{
			Username: u.GetUsername(),
		},
		FromNodeIDs: []int64{wrCreate.Workflow.WorkflowData.Node.ID},
	}, *consumer, nil)
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

	proj2, errP := project.Load(context.TODO(), api.mustDB(), proj.Key, project.LoadOptions.WithPipelines, project.LoadOptions.WithGroups, project.LoadOptions.WithIntegrations)
	require.NoError(t, errP)

	require.NoError(t, workflow.Insert(context.TODO(), db, api.Cache, *proj2, &w))
	w1, err := workflow.Load(context.TODO(), api.mustDB(), api.Cache, *proj, "test_1", workflow.LoadOptions{})
	require.NoError(t, err)

	for i := 0; i < 10; i++ {
		wr, err := workflow.CreateRun(api.mustDB(), w1, sdk.WorkflowRunPostHandlerOption{AuthConsumerID: consumer.ID})
		assert.NoError(t, err)
		wr.Workflow = *w1
		_, err = workflow.StartWorkflowRun(context.TODO(), db, api.Cache, *proj, wr, &sdk.WorkflowRunPostHandlerOption{
			Manual: &sdk.WorkflowNodeRunManual{
				Username: u.GetUsername(),
			},
		}, *consumer, nil)
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

	proj2, errP := project.Load(context.TODO(), api.mustDB(), proj.Key, project.LoadOptions.WithPipelines, project.LoadOptions.WithGroups, project.LoadOptions.WithIntegrations)
	require.NoError(t, errP)

	require.NoError(t, workflow.Insert(context.TODO(), db, api.Cache, *proj2, &w))
	w1, err := workflow.Load(context.TODO(), api.mustDB(), api.Cache, *proj, "test_1", workflow.LoadOptions{})
	require.NoError(t, err)

	wr, err := workflow.CreateRun(api.mustDB(), w1, sdk.WorkflowRunPostHandlerOption{AuthConsumerID: consumer.ID})
	assert.NoError(t, err)
	wr.Workflow = *w1
	_, err = workflow.StartWorkflowRun(context.TODO(), db, api.Cache, *proj, wr, &sdk.WorkflowRunPostHandlerOption{
		Manual: &sdk.WorkflowNodeRunManual{
			Username: u.GetUsername(),
		},
	}, *consumer, nil)
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

	proj2, errP := project.Load(context.TODO(), api.mustDB(), proj.Key, project.LoadOptions.WithPipelines, project.LoadOptions.WithGroups, project.LoadOptions.WithIntegrations)
	require.NoError(t, errP)

	require.NoError(t, workflow.Insert(context.TODO(), db, api.Cache, *proj2, &w))
	w1, err := workflow.Load(context.TODO(), api.mustDB(), api.Cache, *proj, "test_1", workflow.LoadOptions{})
	require.NoError(t, err)

	for i := 0; i < 10; i++ {
		wr, err := workflow.CreateRun(api.mustDB(), w1, sdk.WorkflowRunPostHandlerOption{AuthConsumerID: consumer.ID})
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
		}, *consumer, nil)
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

	proj2, errP := project.Load(context.TODO(), api.mustDB(), proj.Key, project.LoadOptions.WithPipelines, project.LoadOptions.WithGroups, project.LoadOptions.WithIntegrations)
	require.NoError(t, errP)

	require.NoError(t, workflow.Insert(context.TODO(), db, api.Cache, *proj2, &w))
	w1, err := workflow.Load(context.TODO(), api.mustDB(), api.Cache, *proj, "test_1", workflow.LoadOptions{})
	require.NoError(t, err)

	for i := 0; i < 10; i++ {
		wr, err := workflow.CreateRun(api.mustDB(), w1, sdk.WorkflowRunPostHandlerOption{AuthConsumerID: consumer.ID})
		assert.NoError(t, err)
		wr.Workflow = *w1
		_, err = workflow.StartWorkflowRun(context.TODO(), db, api.Cache, *proj, wr, &sdk.WorkflowRunPostHandlerOption{
			Manual: &sdk.WorkflowNodeRunManual{
				Username: u.GetUsername(),
			},
		}, *consumer, nil)
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
	api, db, router := newTestAPI(t)

	u, pass := assets.InsertAdminUser(t, db)
	consumer, _ := authentication.LoadConsumerByTypeAndUserID(context.TODO(), db, sdk.ConsumerLocal, u.ID, authentication.LoadConsumerOptions.WithAuthentifiedUser)

	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, api.Cache, key, key)

	// Application
	app := sdk.Application{
		ProjectID: proj.ID,
		Name:      "app",
	}
	require.NoError(t, application.Insert(db, proj.ID, &app))

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

	proj2, errP := project.Load(context.TODO(), api.mustDB(), proj.Key, project.LoadOptions.WithPipelines, project.LoadOptions.WithGroups, project.LoadOptions.WithIntegrations, project.LoadOptions.WithApplications)
	require.NoError(t, errP)

	require.NoError(t, workflow.Insert(context.TODO(), db, api.Cache, *proj2, &w))
	w1, err := workflow.Load(context.TODO(), api.mustDB(), api.Cache, *proj, "test_1", workflow.LoadOptions{})
	require.NoError(t, err)

	wr, err := workflow.CreateRun(api.mustDB(), w1, sdk.WorkflowRunPostHandlerOption{AuthConsumerID: consumer.ID})
	assert.NoError(t, err)
	wr.Workflow = *w1
	_, err = workflow.StartWorkflowRun(context.TODO(), db, api.Cache, *proj2, wr, &sdk.WorkflowRunPostHandlerOption{
		Manual: &sdk.WorkflowNodeRunManual{
			Username: u.GetUsername(),
		},
	}, *consumer, nil)
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
	assert.NoError(t, workflow.InsertVulnerabilityReport(db, &report))

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
	api, db, router := newTestAPI(t)

	u, pass := assets.InsertAdminUser(t, db)
	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, api.Cache, key, key)

	projKey := sdk.ProjectKey{
		Name:      "proj-sshkey",
		Type:      sdk.KeySSHParameter,
		Public:    "publicssh-proj",
		Private:   "privatessh-proj",
		Builtin:   false,
		ProjectID: proj.ID,
		KeyID:     "key-id-proj",
	}
	require.NoError(t, project.InsertKey(db, &projKey))

	pwdProject := sdk.ProjectVariable{
		Name:  "projvar",
		Type:  sdk.SecretVariable,
		Value: "myprojpassword",
	}
	require.NoError(t, project.InsertVariable(db, proj.ID, &pwdProject, u))

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

	modelIntegration := sdk.IntegrationModel{
		Name:       sdk.RandomString(10),
		Deployment: true,
	}
	require.NoError(t, integration.InsertModel(db, &modelIntegration))
	projInt := sdk.ProjectIntegration{
		Config: sdk.IntegrationConfig{
			"test": sdk.IntegrationConfigValue{
				Description: "here is a test",
				Type:        sdk.IntegrationConfigTypeString,
				Value:       "test",
			},
			"mypassword": sdk.IntegrationConfigValue{
				Description: "here isa password",
				Type:        sdk.IntegrationConfigTypePassword,
				Value:       "mypassword",
			},
		},
		Name:               sdk.RandomString(10),
		ProjectID:          proj.ID,
		Model:              modelIntegration,
		IntegrationModelID: modelIntegration.ID,
	}
	require.NoError(t, integration.InsertIntegration(db, &projInt))
	t.Logf("### Integration %s created with id: %d\n", projInt.Name, projInt.ID)

	p := sdk.GRPCPlugin{
		Author:             "unitTest",
		Description:        "desc",
		Name:               sdk.RandomString(10),
		Type:               sdk.GRPCPluginDeploymentIntegration,
		IntegrationModelID: &modelIntegration.ID,
		Integration:        modelIntegration.Name,
		Binaries: []sdk.GRPCPluginBinary{
			{
				OS:   "linux",
				Arch: "adm64",
				Name: "blabla",
			},
		},
	}

	require.NoError(t, plugin.Insert(db, &p))
	assert.NotEqual(t, 0, p.ID)

	app := sdk.Application{
		ProjectID: proj.ID,
		Name:      sdk.RandomString(10),
		Variables: []sdk.ApplicationVariable{
			{
				Name:  "app-password",
				Type:  sdk.SecretVariable,
				Value: "apppassword",
			},
			{
				Name:  "app-clear",
				Type:  sdk.StringVariable,
				Value: "apppassword",
			},
		},
		Keys: []sdk.ApplicationKey{
			{
				Type:    sdk.KeySSHParameter,
				Name:    "app-sshkey",
				Private: "private-key",
				Public:  "public-key",
				KeyID:   "id",
			},
		},
		DeploymentStrategies: map[string]sdk.IntegrationConfig{
			projInt.Name: map[string]sdk.IntegrationConfigValue{
				"token": {
					Type:        "password",
					Value:       "app-token",
					Description: "token",
				},
				"notoken": {
					Type:        "string",
					Value:       "app-token",
					Description: "token",
				},
			},
		},
	}
	require.NoError(t, application.Insert(db, proj.ID, &app))
	require.NoError(t, application.InsertVariable(db, app.ID, &app.Variables[0], u))
	app.Keys[0].ApplicationID = app.ID
	require.NoError(t, application.InsertKey(db, &app.Keys[0]))
	require.NoError(t, application.SetDeploymentStrategy(db, proj.ID, app.ID, modelIntegration.ID, projInt.Name, app.DeploymentStrategies[projInt.Name]))

	env := sdk.Environment{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       sdk.RandomString(10),
		Variables: []sdk.EnvironmentVariable{
			{
				Name:  "env-password",
				Type:  sdk.SecretVariable,
				Value: "envpassword",
			},
			{
				Name:  "env-data",
				Type:  sdk.StringVariable,
				Value: "coucou",
			},
		},
		Keys: []sdk.EnvironmentKey{
			{
				Type:    sdk.KeySSHParameter,
				Name:    "env-sshkey",
				Private: "private-key-env",
				Public:  "public-key-env",
				KeyID:   "id-env",
			},
		},
	}
	require.NoError(t, environment.InsertEnvironment(db, &env))
	require.NoError(t, environment.InsertVariable(db, env.ID, &env.Variables[0], u))
	env.Keys[0].EnvironmentID = env.ID
	require.NoError(t, environment.InsertKey(db, &env.Keys[0]))

	proj2, errP := project.Load(context.TODO(), api.mustDB(), key,
		project.LoadOptions.WithApplicationWithDeploymentStrategies,
		project.LoadOptions.WithPipelines,
		project.LoadOptions.WithEnvironments,
		project.LoadOptions.WithGroups,
		project.LoadOptions.WithIntegrations,
	)
	require.NoError(t, errP)

	w := sdk.Workflow{
		Name:       "test_1",
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		WorkflowData: sdk.WorkflowData{
			Node: sdk.Node{
				Name: "root",
				Type: sdk.NodeTypePipeline,
				Context: &sdk.NodeContext{
					PipelineID:           pip.ID,
					ApplicationID:        app.ID,
					EnvironmentID:        env.ID,
					ProjectIntegrationID: proj2.Integrations[0].ID,
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

	proj2, errP = project.Load(context.TODO(), api.mustDB(), proj.Key, project.LoadOptions.WithPipelines, project.LoadOptions.WithGroups, project.LoadOptions.WithIntegrations)
	require.NoError(t, errP)

	require.NoError(t, workflow.Insert(context.TODO(), db, api.Cache, *proj2, &w))
	w1, err := workflow.Load(context.TODO(), api.mustDB(), api.Cache, *proj2, "test_1", workflow.LoadOptions{})
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
	assert.NoError(t, waitCraftinWorkflow(t, api, db, wr.ID))

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

	secrets, err := workflow.LoadDecryptSecrets(context.TODO(), db, lastRun, lastRun.RootRun())
	require.NoError(t, err)

	t.Logf("%+v", secrets)

	// Proj key
	require.NotNil(t, sdk.VariableFind(secrets, "cds.key.proj-sshkey.priv"))
	// Project password
	require.NotNil(t, sdk.VariableFind(secrets, "cds.proj.projvar"))

	// Proj Integration
	require.NotNil(t, sdk.VariableFind(secrets, "cds.integration.mypassword"))

	// Application variable
	require.Nil(t, sdk.VariableFind(secrets, "cds.app.app-clear"))
	require.NotNil(t, sdk.VariableFind(secrets, "cds.app.app-password"))
	// Application key
	require.NotNil(t, sdk.VariableFind(secrets, "cds.key.app-sshkey.priv"))
	// Application integration
	require.NotNil(t, sdk.VariableFind(secrets, "cds.integration.token"))
	require.Nil(t, sdk.VariableFind(secrets, "cds.integration.notoken"))

	// Env variable
	require.NotNil(t, sdk.VariableFind(secrets, "cds.env.env-password"))
	require.Nil(t, sdk.VariableFind(secrets, "cds.env.env-data"))
	// En  key
	require.NotNil(t, sdk.VariableFind(secrets, "cds.key.env-sshkey.priv"))

	// Check public and id key in node run param
	require.NotNil(t, sdk.ParameterFind(lastRun.RootRun().BuildParameters, "cds.key.proj-sshkey.pub"))
	require.NotNil(t, sdk.ParameterFind(lastRun.RootRun().BuildParameters, "cds.key.proj-sshkey.id"))
	require.NotNil(t, sdk.ParameterFind(lastRun.RootRun().BuildParameters, "cds.key.app-sshkey.pub"))
	require.NotNil(t, sdk.ParameterFind(lastRun.RootRun().BuildParameters, "cds.key.app-sshkey.id"))
	require.NotNil(t, sdk.ParameterFind(lastRun.RootRun().BuildParameters, "cds.key.env-sshkey.pub"))
	require.NotNil(t, sdk.ParameterFind(lastRun.RootRun().BuildParameters, "cds.key.env-sshkey.id"))

}

func waitCraftinWorkflow(t *testing.T, api *API, db gorp.SqlExecutor, id int64) error {
	t.Logf("(%v) waitCraftingWorkflow %d", time.Now(), id)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	go api.WorkflowRunCraft(ctx, 10*time.Millisecond)

	tick := time.NewTicker(1 * time.Second)
	defer tick.Stop()
	for {
		select {
		case <-ctx.Done():
			t.Logf("(%v) exiting waitCraftingWorkflow %d", time.Now(), id)
			return ctx.Err()
		case <-tick.C:
			w, _ := workflow.LoadRunByID(api.mustDB(), id, workflow.LoadRunOptions{})
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

func Test_workflowRunCraft(t *testing.T) {
	featureflipping.Init(gorpmapping.Mapper)
	api, db, _ := newTestAPI(t)
	key := sdk.RandomString(10)

	features, err := featureflipping.LoadAll(context.TODO(), gorpmapping.Mapper, db)
	require.NoError(t, err)
	for _, f := range features {
		_ = featureflipping.Delete(db, f.ID)
	}

	proj := assets.InsertTestProject(t, db, api.Cache, key, key)
	wf := assets.InsertTestWorkflow(t, db, api.Cache, proj, sdk.RandomString(10))

	require.NoError(t, workflow.UpdateMaxRunsByID(db, wf.ID, 1))

	wr, err := workflow.CreateRun(db.DbMap, wf, sdk.WorkflowRunPostHandlerOption{
		Hook: &sdk.WorkflowNodeRunHookEvent{},
	})
	require.NoError(t, err)
	wr.Status = sdk.StatusSuccess
	require.NoError(t, workflow.UpdateWorkflowRunStatus(db, wr))

	wrPending, err := workflow.CreateRun(db.DbMap, wf, sdk.WorkflowRunPostHandlerOption{
		Hook: &sdk.WorkflowNodeRunHookEvent{},
	})
	require.NoError(t, err)

	f := sdk.Feature{
		Name: purge.FeatureMaxRuns,
		Rule: "return true",
	}
	require.NoError(t, featureflipping.Insert(gorpmapping.Mapper, api.mustDB(), &f))

	require.NoError(t, api.workflowRunCraft(context.TODO(), wrPending.ID))

	wrDB, err := workflow.LoadRunByID(db, wrPending.ID, workflow.LoadRunOptions{})
	require.NoError(t, err)

	require.Len(t, wrDB.Infos, 1)
	require.Equal(t, sdk.MsgTooMuchWorkflowRun.ID, wrDB.Infos[0].Message.ID)
}

/**
 * This test does
 * 1. Create worklow
 * 2. Migrate as code => this will create PR.id = 1
 * 3. Run workflow :  Must fail on getting PR.id = 1
 */
func Test_postWorkflowRunAsyncFailedHandler(t *testing.T) {
	api, db, router := newTestAPI(t)

	u, pass := assets.InsertAdminUser(t, db)
	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, api.Cache, key, key)

	vcsServer := sdk.ProjectVCSServerLink{
		ProjectID: proj.ID,
		Name:      "github",
	}
	vcsServer.Set("token", "foo")
	vcsServer.Set("secret", "bar")
	require.NoError(t, repositoriesmanager.InsertProjectVCSServerLink(context.TODO(), db, &vcsServer))

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
	require.NoError(t, application.Insert(db, proj.ID, &app))
	require.NoError(t, repositoriesmanager.InsertForApplication(db, &app))

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

	proj2, err := project.Load(context.TODO(), api.mustDB(), proj.Key, project.LoadOptions.WithApplications, project.LoadOptions.WithPipelines, project.LoadOptions.WithGroups, project.LoadOptions.WithIntegrations)
	require.NoError(t, err)

	require.NoError(t, workflow.Insert(context.TODO(), db, api.Cache, *proj2, &w))
	w1, err := workflow.Load(context.TODO(), api.mustDB(), api.Cache, *proj, "test_1", workflow.LoadOptions{})
	require.NoError(t, err)

	allSrv, err := services.LoadAll(context.TODO(), db)
	for _, s := range allSrv {
		if err := services.Delete(db, &s); err != nil {
			t.Fatalf("unable to delete service: %v", err)
		}
	}

	// Prepare service mocks
	mockVCSSservice, _ := assets.InsertService(t, db, t.Name()+"_VCS", sdk.TypeVCS)
	defer func() { _ = services.Delete(db, mockVCSSservice) }()
	mockRepoService, _ := assets.InsertService(t, db, t.Name()+"_REPOSITORIES", sdk.TypeRepositories)
	defer func() { _ = services.Delete(db, mockRepoService) }()
	mockHookService, _ := assets.InsertService(t, db, t.Name()+"_HOOKS", sdk.TypeHooks)
	defer func() { _ = services.Delete(db, mockHookService) }()

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
			case "/vcs/github/repos/foo/myrepo/pullrequests?state=open":
				vcsPRs := []sdk.VCSPullRequest{}
				if err := enc.Encode(vcsPRs); err != nil {
					return writeError(w, err)
				}
			case "/vcs/github/repos/foo/myrepo/pullrequests":
				pr := sdk.VCSPullRequest{
					Title: "blabla",
					URL:   "myurl",
					ID:    1,
				}
				if err := enc.Encode(pr); err != nil {
					return writeError(w, err)
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
				return writeError(w, sdk.NewError(sdk.ErrServiceUnavailable,
					fmt.Errorf("route %s must not be called", r.URL.String()),
				))
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
		FromRepo:      "ssh://cloneurl",
		Name:          w1.Name,
		ID:            w1.ID,
		Type:          ascode.WorkflowEvent,
		OperationUUID: ope.UUID,
	}

	ascode.UpdateAsCodeResult(context.TODO(), api.mustDB(), api.Cache, sdk.NewGoRoutines(), *proj, *w1, app, ed, u)

	// Prepare request
	uri := router.GetRoute("POST", api.postWorkflowRunHandler, map[string]string{
		"key":              proj.Key,
		"permWorkflowName": w1.Name,
	})
	req := assets.NewAuthentifiedRequest(t, u, pass, "POST", uri, &sdk.WorkflowRunPostHandlerOption{})
	rec := httptest.NewRecorder()
	router.Mux.ServeHTTP(rec, req)
	require.Equal(t, 202, rec.Code)

	lastRun, err := workflow.LoadLastRun(api.mustDB(), proj.Key, w.Name, workflow.LoadRunOptions{})
	test.NoError(t, err)
	waitCraftinWorkflow(t, api, db, lastRun.ID)

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
		require.Equal(t, 200, recGet.Code)
		var wrGet sdk.WorkflowRun
		recGetBody := recGet.Body.Bytes()
		require.NoError(t, json.Unmarshal(recGetBody, &wrGet))

		if sdk.StatusIsTerminated(wrGet.Status) {
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
		time.Sleep(time.Second)
	}

	t.FailNow()
}

func Test_postWorkflowRunHandlerWithoutRightOnEnvironment(t *testing.T) {
	api, db, router := newTestAPI(t)

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
	require.NoError(t, group.Insert(context.TODO(), db, &gr))

	uLambda, pass := assets.InsertLambdaUser(t, db, &gr)

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

	proj2, errP := project.Load(context.TODO(), api.mustDB(), proj.Key, project.LoadOptions.WithPipelines, project.LoadOptions.WithGroups, project.LoadOptions.WithEnvironments)
	require.NoError(t, errP)

	require.NoError(t, workflow.Insert(context.TODO(), db, api.Cache, *proj2, &w))
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
	api, db, router := newTestAPI(t)

	u, pass := assets.InsertAdminUser(t, db)
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

	mockHookService, _ := assets.InsertService(t, db, "Test_postWorkflowRunHandlerWithoutRightConditionsOnHook", sdk.TypeHooks)
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

	proj2, errP := project.Load(context.TODO(), api.mustDB(), proj.Key, project.LoadOptions.WithPipelines, project.LoadOptions.WithGroups, project.LoadOptions.WithEnvironments)
	test.NoError(t, errP)

	require.NoError(t, workflow.Insert(context.TODO(), db, api.Cache, *proj2, &w))
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
	api, db, router := newTestAPI(t)

	u, pass := assets.InsertAdminUser(t, db)
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

	mockServiceHook, _ := assets.InsertService(t, db, "Test_postWorkflowRunHandlerHookWithMutex", sdk.TypeHooks)
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

	proj2, errP := project.Load(context.TODO(), api.mustDB(), proj.Key, project.LoadOptions.WithPipelines, project.LoadOptions.WithGroups, project.LoadOptions.WithEnvironments)
	test.NoError(t, errP)

	require.NoError(t, workflow.Insert(context.TODO(), db, api.Cache, *proj2, &w))
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

	lastRun, err := workflow.LoadLastRun(api.mustDB(), proj.Key, w.Name, workflow.LoadRunOptions{})
	test.NoError(t, err)
	waitCraftinWorkflow(t, api, db, lastRun.ID)

	req2 := assets.NewAuthentifiedRequest(t, u, pass, "POST", uri, opts)

	//Do the request, start a new run
	rec2 := httptest.NewRecorder()
	router.Mux.ServeHTTP(rec2, req2)
	var body2 []byte
	_, err = req2.Body.Read(body2)
	test.NoError(t, err)
	defer req2.Body.Close()
	assert.Equal(t, 202, rec2.Code)

	lastRun, err = workflow.LoadLastRun(api.mustDB(), proj.Key, w.Name, workflow.LoadRunOptions{})
	test.NoError(t, err)
	waitCraftinWorkflow(t, api, db, lastRun.ID)

	// it's an async call, wait a bit the let cds take care of the previous request
	time.Sleep(3 * time.Second)

	lastRun, err = workflow.LoadLastRun(api.mustDB(), proj.Key, w1.Name, workflow.LoadRunOptions{})
	test.NoError(t, err)
	assert.Equal(t, int64(2), lastRun.Number)
	assert.Equal(t, sdk.StatusBuilding, lastRun.Status)
}

func Test_postWorkflowRunHandlerMutexRelease(t *testing.T) {
	api, db, router := newTestAPI(t)

	u, jwt := assets.InsertAdminUser(t, db)

	// Init test pipeline with one stage and one job
	projKey := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, api.Cache, projKey, projKey)
	pip := sdk.Pipeline{ProjectID: proj.ID, ProjectKey: proj.Key, Name: sdk.RandomString(10)}
	require.NoError(t, pipeline.InsertPipeline(api.mustDB(), &pip))
	stage := sdk.Stage{PipelineID: pip.ID, Name: sdk.RandomString(10), Enabled: true}
	require.NoError(t, pipeline.InsertStage(api.mustDB(), &stage))
	job := &sdk.Job{Enabled: true, Action: sdk.Action{Enabled: true}}
	require.NoError(t, pipeline.InsertJob(api.mustDB(), job, stage.ID, &pip))

	// Init test workflow with one pipeline with mutex
	wkf := sdk.Workflow{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       sdk.RandomString(10),
		WorkflowData: sdk.WorkflowData{
			Node: sdk.Node{
				Name: "root",
				Type: sdk.NodeTypePipeline,
				Context: &sdk.NodeContext{
					PipelineID: pip.ID,
					Mutex:      true,
				},
			},
		},
	}
	require.NoError(t, workflow.Insert(context.TODO(), db, api.Cache, *proj, &wkf))

	// Run workflow 1
	uri := router.GetRoute("POST", api.postWorkflowRunHandler, map[string]string{
		"key":              proj.Key,
		"permWorkflowName": wkf.Name,
	})
	require.NotEmpty(t, uri)
	req := assets.NewAuthentifiedRequest(t, u, jwt, "POST", uri, sdk.WorkflowRunPostHandlerOption{})
	rec := httptest.NewRecorder()
	router.Mux.ServeHTTP(rec, req)
	require.Equal(t, 202, rec.Code)

	lastRun, err := workflow.LoadLastRun(api.mustDB(), proj.Key, wkf.Name, workflow.LoadRunOptions{})
	test.NoError(t, err)
	waitCraftinWorkflow(t, api, db, lastRun.ID)

	var try int
	for {
		if try > 10 {
			t.Logf("Maximum attempts reached on getWorkflowRunHandler for run 1")
			t.FailNow()
			return
		}
		try++
		t.Logf("Attempt #%d on getWorkflowRunHandler for run 1", try)
		uri := router.GetRoute("GET", api.getWorkflowRunHandler, map[string]string{
			"key":              proj.Key,
			"permWorkflowName": wkf.Name,
			"number":           "1",
		})
		req := assets.NewAuthentifiedRequest(t, u, jwt, "GET", uri, nil)
		rec := httptest.NewRecorder()
		router.Mux.ServeHTTP(rec, req)
		require.Equal(t, 200, rec.Code)

		var wkfRun sdk.WorkflowRun
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &wkfRun))
		if wkfRun.Status != sdk.StatusBuilding {
			t.Logf("Workflow run status: %s", wkfRun.Status)
			continue
		}

		require.Equal(t, sdk.StatusBuilding, wkfRun.Status)
		require.Equal(t, sdk.StatusWaiting, wkfRun.RootRun().Stages[0].Status)
		break
	}

	// Run workflow 2
	uri = router.GetRoute("POST", api.postWorkflowRunHandler, map[string]string{
		"key":              proj.Key,
		"permWorkflowName": wkf.Name,
	})
	require.NotEmpty(t, uri)
	req = assets.NewAuthentifiedRequest(t, u, jwt, "POST", uri, sdk.WorkflowRunPostHandlerOption{})
	rec = httptest.NewRecorder()
	router.Mux.ServeHTTP(rec, req)
	require.Equal(t, 202, rec.Code)

	lastRun, err = workflow.LoadLastRun(api.mustDB(), proj.Key, wkf.Name, workflow.LoadRunOptions{})
	test.NoError(t, err)
	waitCraftinWorkflow(t, api, db, lastRun.ID)

	try = 0
	for {
		if try > 10 {
			t.Logf("Maximum attempts reached on getWorkflowRunHandler for run 2")
			t.FailNow()
			return
		}
		try++
		t.Logf("Attempt #%d on getWorkflowRunHandler for run 2", try)
		uri := router.GetRoute("GET", api.getWorkflowRunHandler, map[string]string{
			"key":              proj.Key,
			"permWorkflowName": wkf.Name,
			"number":           "2",
		})
		req := assets.NewAuthentifiedRequest(t, u, jwt, "GET", uri, nil)
		rec := httptest.NewRecorder()
		router.Mux.ServeHTTP(rec, req)
		require.Equal(t, 200, rec.Code)

		var wkfRun sdk.WorkflowRun
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &wkfRun))
		if wkfRun.Status != sdk.StatusBuilding {
			t.Logf("Workflow run status: %s", wkfRun.Status)
			continue
		}

		require.Equal(t, sdk.StatusBuilding, wkfRun.Status)
		require.Equal(t, 2, len(wkfRun.Infos))
		require.Equal(t, sdk.MsgWorkflowStarting.ID, wkfRun.Infos[0].Message.ID)
		require.Equal(t, sdk.MsgWorkflowNodeMutex.ID, wkfRun.Infos[1].Message.ID)
		require.Equal(t, "", wkfRun.RootRun().Stages[0].Status)
		break
	}

	// Run workflow 3
	uri = router.GetRoute("POST", api.postWorkflowRunHandler, map[string]string{
		"key":              proj.Key,
		"permWorkflowName": wkf.Name,
	})
	require.NotEmpty(t, uri)
	req = assets.NewAuthentifiedRequest(t, u, jwt, "POST", uri, sdk.WorkflowRunPostHandlerOption{})
	rec = httptest.NewRecorder()
	router.Mux.ServeHTTP(rec, req)
	require.Equal(t, 202, rec.Code)

	lastRun, err = workflow.LoadLastRun(api.mustDB(), proj.Key, wkf.Name, workflow.LoadRunOptions{})
	test.NoError(t, err)
	waitCraftinWorkflow(t, api, db, lastRun.ID)

	try = 0
	for {
		if try > 10 {
			t.Logf("Maximum attempts reached on getWorkflowRunHandler for run 3")
			t.FailNow()
			return
		}
		try++
		t.Logf("Attempt #%d on getWorkflowRunHandler for run 3", try)
		uri := router.GetRoute("GET", api.getWorkflowRunHandler, map[string]string{
			"key":              proj.Key,
			"permWorkflowName": wkf.Name,
			"number":           "3",
		})
		req := assets.NewAuthentifiedRequest(t, u, jwt, "GET", uri, nil)
		rec := httptest.NewRecorder()
		router.Mux.ServeHTTP(rec, req)
		require.Equal(t, 200, rec.Code)

		var wkfRun sdk.WorkflowRun
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &wkfRun))
		if wkfRun.Status != sdk.StatusBuilding {
			t.Logf("Workflow run status: %s", wkfRun.Status)
			continue
		}

		require.Equal(t, sdk.StatusBuilding, wkfRun.Status)
		require.Equal(t, 2, len(wkfRun.Infos))
		require.Equal(t, sdk.MsgWorkflowStarting.ID, wkfRun.Infos[0].Message.ID)
		require.Equal(t, sdk.MsgWorkflowNodeMutex.ID, wkfRun.Infos[1].Message.ID)
		require.Equal(t, "", wkfRun.RootRun().Stages[0].Status)
		break
	}

	// Stop workflow 1
	uri = router.GetRoute("POST", api.stopWorkflowRunHandler, map[string]string{
		"key":              proj.Key,
		"permWorkflowName": wkf.Name,
		"number":           "1",
	})
	require.NotEmpty(t, uri)
	req = assets.NewAuthentifiedRequest(t, u, jwt, "POST", uri, nil)
	rec = httptest.NewRecorder()
	router.Mux.ServeHTTP(rec, req)
	require.Equal(t, 200, rec.Code)

	try = 0
	for {
		if try > 10 {
			t.Logf("Maximum attempts reached on getWorkflowRunHandler for run 1")
			t.FailNow()
			return
		}
		try++
		t.Logf("Attempt #%d on getWorkflowRunHandler for run 1", try)
		uri := router.GetRoute("GET", api.getWorkflowRunHandler, map[string]string{
			"key":              proj.Key,
			"permWorkflowName": wkf.Name,
			"number":           "1",
		})
		req := assets.NewAuthentifiedRequest(t, u, jwt, "GET", uri, nil)
		rec := httptest.NewRecorder()
		router.Mux.ServeHTTP(rec, req)
		require.Equal(t, 200, rec.Code)

		var wkfRun sdk.WorkflowRun
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &wkfRun))
		if wkfRun.Status != sdk.StatusStopped {
			t.Logf("Workflow run status: %s", wkfRun.Status)
			continue
		}

		require.Equal(t, sdk.StatusStopped, wkfRun.Status)
		require.Equal(t, sdk.StatusStopped, wkfRun.RootRun().Stages[0].Status)
		break
	}

	// Run 2 should be running
	try = 0
	for {
		if try > 10 {
			t.Logf("Maximum attempts reached on getWorkflowRunHandler for run 2")
			t.FailNow()
			return
		}
		try++
		t.Logf("Attempt #%d on getWorkflowRunHandler for run 2", try)
		uri := router.GetRoute("GET", api.getWorkflowRunHandler, map[string]string{
			"key":              proj.Key,
			"permWorkflowName": wkf.Name,
			"number":           "2",
		})
		req := assets.NewAuthentifiedRequest(t, u, jwt, "GET", uri, nil)
		rec := httptest.NewRecorder()
		router.Mux.ServeHTTP(rec, req)
		require.Equal(t, 200, rec.Code)

		var wkfRun sdk.WorkflowRun
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &wkfRun))
		if wkfRun.Status != sdk.StatusBuilding && wkfRun.RootRun().Stages[0].Status == sdk.StatusWaiting {
			t.Logf("Workflow run status: %s", wkfRun.Status)
			continue
		}

		require.Equal(t, sdk.StatusBuilding, wkfRun.Status)
		require.Equal(t, sdk.StatusWaiting, wkfRun.RootRun().Stages[0].Status, "Stop a previous workflow run should have release the mutex and trigger the second run, status of the stage should change for empty string to waiting")
		break
	}

	// Run 3 should still be locked
	try = 0
	for {
		if try > 10 {
			t.Logf("Maximum attempts reached on getWorkflowRunHandler for run 3")
			t.FailNow()
			return
		}
		try++
		t.Logf("Attempt #%d on getWorkflowRunHandler for run 3", try)
		uri := router.GetRoute("GET", api.getWorkflowRunHandler, map[string]string{
			"key":              proj.Key,
			"permWorkflowName": wkf.Name,
			"number":           "3",
		})
		req := assets.NewAuthentifiedRequest(t, u, jwt, "GET", uri, nil)
		rec := httptest.NewRecorder()
		router.Mux.ServeHTTP(rec, req)
		require.Equal(t, 200, rec.Code)

		var wkfRun sdk.WorkflowRun
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &wkfRun))
		if wkfRun.Status != sdk.StatusBuilding {
			t.Logf("Workflow run status: %s", wkfRun.Status)
			continue
		}

		require.Equal(t, sdk.StatusBuilding, wkfRun.Status)
		require.Equal(t, 2, len(wkfRun.Infos))
		require.Equal(t, sdk.MsgWorkflowStarting.ID, wkfRun.Infos[0].Message.ID)
		require.Equal(t, sdk.MsgWorkflowNodeMutex.ID, wkfRun.Infos[1].Message.ID)
		require.Equal(t, "", wkfRun.RootRun().Stages[0].Status)
		break
	}
}

func Test_postWorkflowRunHandlerHook(t *testing.T) {
	api, db, router := newTestAPI(t)

	u, pass := assets.InsertAdminUser(t, db)
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

	mockServiceHook, _ := assets.InsertService(t, db, "Test_postWorkflowRunHandlerHookWithMutex", sdk.TypeHooks)
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

	proj2, errP := project.Load(context.TODO(), api.mustDB(), proj.Key, project.LoadOptions.WithPipelines, project.LoadOptions.WithGroups, project.LoadOptions.WithEnvironments)
	test.NoError(t, errP)

	require.NoError(t, workflow.Insert(context.TODO(), db, api.Cache, *proj2, &w))
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

	assert.NoError(t, waitCraftinWorkflow(t, api, db, wr.ID))
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
	api, db, router := newTestAPI(t)

	u, pass := assets.InsertAdminUser(t, db)
	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, api.Cache, key, key)

	gr := &sdk.Group{
		Name: sdk.RandomString(10),
	}
	require.NoError(t, group.Insert(context.TODO(), db, gr))
	require.NoError(t, group.InsertLinkGroupProject(context.TODO(), db, &group.LinkGroupProject{
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

	proj2, errp := project.Load(context.TODO(), api.mustDB(), proj.Key, project.LoadOptions.WithPipelines, project.LoadOptions.WithEnvironments)
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

	require.NoError(t, workflow.Insert(context.TODO(), db, api.Cache, *proj2, &w))

	u.Ring = ""
	require.NoError(t, user.Update(context.TODO(), db, u))

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
	api, db, router := newTestAPI(t)

	u, pass := assets.InsertAdminUser(t, db)
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

	proj2, errp := project.Load(context.TODO(), api.mustDB(), proj.Key, project.LoadOptions.WithPipelines, project.LoadOptions.WithEnvironments)
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

	require.NoError(t, workflow.Insert(context.TODO(), db, api.Cache, *proj2, &w))

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

	lastRun, err := workflow.LoadLastRun(api.mustDB(), proj.Key, w.Name, workflow.LoadRunOptions{})
	test.NoError(t, err)
	waitCraftinWorkflow(t, api, db, lastRun.ID)

	// it's an async call, wait a bit the let cds take care of the previous request
	time.Sleep(3 * time.Second)

	lastRun, err = workflow.LoadLastRun(api.mustDB(), proj.Key, w.Name, workflow.LoadRunOptions{})
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
	api, db, router := newTestAPI(t)

	u, pass := assets.InsertAdminUser(t, db)
	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, api.Cache, key, key)

	gr := &sdk.Group{
		Name: sdk.RandomString(10),
	}
	require.NoError(t, group.Insert(context.TODO(), db, gr))
	require.NoError(t, group.InsertLinkGroupProject(context.TODO(), db, &group.LinkGroupProject{
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

	proj2, errp := project.Load(context.TODO(), api.mustDB(), proj.Key, project.LoadOptions.WithPipelines, project.LoadOptions.WithEnvironments)
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

	require.NoError(t, workflow.Insert(context.TODO(), db, api.Cache, *proj2, &w))

	require.NoError(t, user.Update(context.TODO(), db, u))

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

func initGetWorkflowNodeRunJobTest(t *testing.T, api *API, db gorpmapper.SqlExecutorWithTx) (*sdk.AuthentifiedUser, string, *sdk.Project, *sdk.Workflow, *sdk.WorkflowRun, *sdk.WorkflowNodeJobRun) {
	u, pass := assets.InsertAdminUser(t, db)
	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, api.Cache, key, key)

	//First pipeline
	pip := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip1",
	}
	require.NoError(t, pipeline.InsertPipeline(api.mustDB(), &pip))

	script := assets.GetBuiltinOrPluginActionByName(t, db, sdk.ScriptAction)

	s := sdk.NewStage("stage 1")
	s.Enabled = true
	s.PipelineID = pip.ID
	pipeline.InsertStage(api.mustDB(), s)
	j := &sdk.Job{
		Enabled: true,
		Action: sdk.Action{
			Enabled: true,
			Actions: []sdk.Action{
				assets.NewAction(script.ID, sdk.Parameter{Name: "script", Value: "echo lol"})},
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
			Actions: []sdk.Action{
				assets.NewAction(script.ID, sdk.Parameter{Name: "script", Value: "echo lol"}),
			},
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

	proj2, errP := project.Load(context.TODO(), api.mustDB(), proj.Key, project.LoadOptions.WithPipelines, project.LoadOptions.WithGroups, project.LoadOptions.WithIntegrations)
	require.NoError(t, errP)
	consumer, _ := authentication.LoadConsumerByTypeAndUserID(context.TODO(), db, sdk.ConsumerLocal, u.ID, authentication.LoadConsumerOptions.WithAuthentifiedUser)

	require.NoError(t, workflow.Insert(context.TODO(), db, api.Cache, *proj2, &w))
	w1, err := workflow.Load(context.TODO(), api.mustDB(), api.Cache, *proj, "test_1", workflow.LoadOptions{
		DeepPipeline: true,
	})
	require.NoError(t, err)

	wr, err := workflow.CreateRun(api.mustDB(), w1, sdk.WorkflowRunPostHandlerOption{AuthConsumerID: consumer.ID})
	assert.NoError(t, err)
	wr.Workflow = *w1
	_, err = workflow.StartWorkflowRun(context.TODO(), db, api.Cache, *proj, wr, &sdk.WorkflowRunPostHandlerOption{
		Manual: &sdk.WorkflowNodeRunManual{
			Username: u.GetUsername(),
		},
	}, *consumer, nil)
	require.NoError(t, err)

	lastRun, err := workflow.LoadLastRun(api.mustDB(), proj.Key, w1.Name, workflow.LoadRunOptions{WithArtifacts: true})
	require.NoError(t, err)

	// Update step status
	jobRun := &lastRun.WorkflowNodeRuns[w1.WorkflowData.Node.ID][0].Stages[0].RunJobs[0]
	jobRun.Job.StepStatus = []sdk.StepStatus{
		{
			StepOrder: 0,
			Status:    sdk.StatusBuilding,
		},
	}

	// Update node job run
	require.NoError(t, workflow.UpdateNodeRun(api.mustDB(), &lastRun.WorkflowNodeRuns[w1.WorkflowData.Node.ID][0]))

	// Add log
	require.NoError(t, workflow.AppendLog(api.mustDB(), jobRun.ID, jobRun.WorkflowNodeRunID, 0, "1234567890", 15))

	// Add truncated log
	require.NoError(t, workflow.AppendLog(api.mustDB(), jobRun.ID, jobRun.WorkflowNodeRunID, 0, "1234567890", 15))

	// Add service log
	require.NoError(t, workflow.AddServiceLog(api.mustDB(), &sdk.ServiceLog{
		WorkflowNodeRunID:      jobRun.WorkflowNodeRunID,
		WorkflowNodeJobRunID:   jobRun.ID,
		Val:                    "0987654321",
		ServiceRequirementName: "postgres",
	}, 15))

	// Add truncated service log
	require.NoError(t, workflow.AddServiceLog(api.mustDB(), &sdk.ServiceLog{
		WorkflowNodeRunID:      jobRun.WorkflowNodeRunID,
		WorkflowNodeJobRunID:   jobRun.ID,
		Val:                    "0987654321",
		ServiceRequirementName: "postgres",
	}, 15))

	return u, pass, proj, w1, lastRun, jobRun
}

func Test_deleteWorkflowRunsBranchHandler(t *testing.T) {
	api, db, router := newTestAPI(t)

	u, pass := assets.InsertAdminUser(t, db)
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

	proj2, errP := project.Load(context.TODO(), api.mustDB(), proj.Key, project.LoadOptions.WithPipelines, project.LoadOptions.WithGroups, project.LoadOptions.WithIntegrations)
	require.NoError(t, errP)

	require.NoError(t, workflow.Insert(context.TODO(), db, api.Cache, *proj2, &w))
	w1, err := workflow.Load(context.TODO(), api.mustDB(), api.Cache, *proj, "test_1", workflow.LoadOptions{})
	require.NoError(t, err)

	wr, err := workflow.CreateRun(api.mustDB(), w1, sdk.WorkflowRunPostHandlerOption{AuthConsumerID: consumer.ID})
	assert.NoError(t, err)
	wr.Workflow = *w1
	wr.Tag("git.branch", "master")
	assert.NoError(t, workflow.UpdateWorkflowRun(context.TODO(), api.mustDB(), wr))
	_, err = workflow.StartWorkflowRun(context.TODO(), db, api.Cache, *proj, wr, &sdk.WorkflowRunPostHandlerOption{
		Manual: &sdk.WorkflowNodeRunManual{
			Username: u.GetUsername(),
			Payload:  `{"git.branch": "master"}`,
		},
	}, *consumer, nil)
	require.NoError(t, err)

	mockHookService, _ := assets.InsertService(t, db, "Test_deleteWorkflowRunsBranchHandler", sdk.TypeHooks, sdk.AuthConsumerScopeRun)
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
	require.NotEmpty(t, uri)
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
	require.NotEmpty(t, uri)
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
	api, db, router := newTestAPI(t)

	u, pass := assets.InsertAdminUser(t, db)
	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, api.Cache, key, key)
	consumer, _ := authentication.LoadConsumerByTypeAndUserID(context.TODO(), db, sdk.ConsumerLocal, u.ID, authentication.LoadConsumerOptions.WithAuthentifiedUser)

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

	proj2, errP := project.Load(context.TODO(), api.mustDB(), proj.Key, project.LoadOptions.WithPipelines, project.LoadOptions.WithGroups, project.LoadOptions.WithIntegrations)
	require.NoError(t, errP)

	require.NoError(t, workflow.Insert(context.TODO(), db, api.Cache, *proj2, &w))
	w1, err := workflow.Load(context.TODO(), api.mustDB(), api.Cache, *proj, "test_1", workflow.LoadOptions{})
	require.NoError(t, err)

	wr, err := workflow.CreateRun(api.mustDB(), w1, sdk.WorkflowRunPostHandlerOption{AuthConsumerID: consumer.ID})
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
	api, db, router := newTestAPI(t)

	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, api.Cache, key, key)
	w := assets.InsertTestWorkflow(t, db, api.Cache, proj, sdk.RandomString(10))
	u, pass := assets.InsertLambdaUser(t, db, &proj.ProjectGroups[0].Group)

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
	api, db, router := newTestAPI(t)

	u, pass := assets.InsertAdminUser(t, db)
	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, api.Cache, key, key)
	consumer, _ := authentication.LoadConsumerByTypeAndUserID(context.TODO(), db, sdk.ConsumerLocal, u.ID, authentication.LoadConsumerOptions.WithAuthentifiedUser)

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

	proj2, errP := project.Load(context.TODO(), api.mustDB(), proj.Key, project.LoadOptions.WithPipelines, project.LoadOptions.WithGroups, project.LoadOptions.WithIntegrations)
	require.NoError(t, errP)

	require.NoError(t, workflow.Insert(context.TODO(), db, api.Cache, *proj2, &w))
	w1, err := workflow.Load(context.TODO(), api.mustDB(), api.Cache, *proj, "test_1", workflow.LoadOptions{})
	require.NoError(t, err)

	//Prepare request
	vars := map[string]string{
		"key":              proj.Key,
		"permWorkflowName": w1.Name,
	}
	uri := router.GetRoute("POST", api.postWorkflowRunHandler, vars)
	test.NotEmpty(t, uri)

	opts := sdk.WorkflowRunPostHandlerOption{
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
	assert.NoError(t, waitCraftinWorkflow(t, api, db, wr.ID))

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

	opts = sdk.WorkflowRunPostHandlerOption{
		Manual: &sdk.WorkflowNodeRunManual{
			OnlyFailedJobs: true,
			Resync:         false,
		},
		FromNodeIDs:    []int64{w1.WorkflowData.Node.ID},
		Number:         &wrr.Number,
		AuthConsumerID: consumer.ID,
	}
	api.initWorkflowRun(context.TODO(), proj2.Key, &wrr.Workflow, wrr, opts)

	wrr, _ = workflow.LoadRun(context.TODO(), db, proj2.Key, w1.Name, 1, workflow.LoadRunOptions{})

	assert.Equal(t, sdk.StatusBuilding, wrr.Status)
	assert.Equal(t, firstJobEnd.Unix(), wrr.WorkflowNodeRuns[wrr.Workflow.WorkflowData.Node.ID][0].Stages[0].RunJobs[0].Start.Unix())
	assert.NotEqual(t, firstJobEnd, wrr.WorkflowNodeRuns[wrr.Workflow.WorkflowData.Node.ID][0].Stages[0].RunJobs[1].Start)
	assert.Equal(t, sdk.StatusSuccess, wrr.WorkflowNodeRuns[wrr.Workflow.WorkflowData.Node.ID][0].Stages[0].RunJobs[0].Status)
	assert.Equal(t, sdk.StatusWaiting, wrr.WorkflowNodeRuns[wrr.Workflow.WorkflowData.Node.ID][0].Stages[0].RunJobs[1].Status)
}

func Test_postWorkflowRunHandlerRestartResync(t *testing.T) {
	api, db, router := newTestAPI(t)

	u, pass := assets.InsertAdminUser(t, db)
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

	proj2, errP := project.Load(context.TODO(), api.mustDB(), proj.Key, project.LoadOptions.WithPipelines, project.LoadOptions.WithGroups, project.LoadOptions.WithIntegrations)
	require.NoError(t, errP)

	require.NoError(t, workflow.Insert(context.TODO(), db, api.Cache, *proj2, &w))
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
	assert.NoError(t, waitCraftinWorkflow(t, api, db, wr.ID))

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
