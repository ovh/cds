package workflow_test

import (
	"context"
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"

	"github.com/ovh/cds/engine/api/bootstrap"
	"github.com/ovh/cds/engine/api/event"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/sdk"
)

func TestCanBeRun(t *testing.T) {
	wnrs := map[int64][]sdk.WorkflowNodeRun{}
	node1 := sdk.Node{ID: 25}
	nodeRoot := sdk.Node{
		ID:   10,
		Type: sdk.NodeTypePipeline,
		Triggers: []sdk.NodeTrigger{
			{
				ChildNode: node1,
			},
		},
	}
	wnrs[nodeRoot.ID] = []sdk.WorkflowNodeRun{
		{ID: 3, WorkflowNodeID: nodeRoot.ID, Status: sdk.StatusBuilding.String()},
	}
	wnrs[node1.ID] = []sdk.WorkflowNodeRun{
		{ID: 3, WorkflowNodeID: node1.ID, Status: sdk.StatusFail.String()},
	}
	wr := &sdk.WorkflowRun{
		Workflow: sdk.Workflow{
			Name:       "test_1",
			ProjectID:  1,
			ProjectKey: "key",
			RootID:     10,
			WorkflowData: &sdk.WorkflowData{
				Node: nodeRoot,
			},
		},
		WorkflowID:       2,
		WorkflowNodeRuns: wnrs,
	}

	wnr := &sdk.WorkflowNodeRun{
		WorkflowNodeID: node1.ID,
	}

	ts := []struct {
		status   string
		canBeRun bool
	}{
		{status: sdk.StatusBuilding.String(), canBeRun: false},
		{status: "", canBeRun: false},
		{status: sdk.StatusSuccess.String(), canBeRun: true},
	}

	for _, tc := range ts {
		wnrs[nodeRoot.ID][0].Status = tc.status
		test.Equal(t, workflow.CanBeRun(wr, wnr), tc.canBeRun)
	}
}

func TestPurgeWorkflowRun(t *testing.T) {
	db, cache, end := test.SetupPG(t, bootstrap.InitiliazeDB)
	defer end()
	event.Initialize(event.KafkaConfig{}, cache)

	u, _ := assets.InsertAdminUser(db)
	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, cache, key, key, u)

	//First pipeline
	pip := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip1",
	}
	test.NoError(t, pipeline.InsertPipeline(db, cache, proj, &pip, u))

	s := sdk.NewStage("stage 1")
	s.Enabled = true
	s.PipelineID = pip.ID
	test.NoError(t, pipeline.InsertStage(db, s))

	proj, _ = project.LoadByID(db, cache, proj.ID, u, project.LoadOptions.WithApplications, project.LoadOptions.WithPipelines, project.LoadOptions.WithEnvironments, project.LoadOptions.WithGroups)

	w := sdk.Workflow{
		Name:       "test_purge_1",
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		WorkflowData: &sdk.WorkflowData{
			Node: sdk.Node{
				Name: "node1",
				Ref:  "node1",
				Type: sdk.NodeTypePipeline,
				Context: &sdk.NodeContext{
					PipelineID: pip.ID,
				},
				Triggers: []sdk.NodeTrigger{
					{
						ChildNode: sdk.Node{
							Name: "node2",
							Ref:  "node2",
							Type: sdk.NodeTypePipeline,
							Context: &sdk.NodeContext{
								PipelineID: pip.ID,
							},
						},
					},
				},
			},
		},

		HistoryLength: 2,
		PurgeTags:     []string{"git.branch"},
	}

	(&w).RetroMigrate()

	test.NoError(t, workflow.Insert(db, cache, &w, proj, u))

	w1, err := workflow.Load(context.TODO(), db, cache, proj, "test_purge_1", u, workflow.LoadOptions{
		DeepPipeline: true,
	})
	test.NoError(t, err)

	for i := 0; i < 5; i++ {
		wr, errWR := workflow.CreateRun(db, w1, nil, u)
		assert.NoError(t, errWR)
		wr.Workflow = *w1
		_, errWr := workflow.StartWorkflowRun(context.TODO(), db, cache, proj, wr, &sdk.WorkflowRunPostHandlerOption{
			Manual: &sdk.WorkflowNodeRunManual{
				User: *u,
				Payload: map[string]string{
					"git.branch": "master",
					"git.author": "test",
				},
			},
		}, nil, nil)
		test.NoError(t, errWr)
	}

	errP := workflow.PurgeWorkflowRun(context.Background(), db, *w1, nil)
	test.NoError(t, errP)

	wruns, _, _, count, errRuns := workflow.LoadRuns(db, proj.Key, w1.Name, 0, 10, nil)
	test.NoError(t, errRuns)
	test.Equal(t, 5, count, "Number of workflow runs isn't correct")

	toDeleteNb := 0
	for _, wfRun := range wruns {
		if wfRun.ToDelete {
			toDeleteNb++
		}
	}

	test.Equal(t, 3, toDeleteNb, "Number of workflow runs to be purged isn't correct")
}

func TestPurgeWorkflowRunWithRunningStatus(t *testing.T) {
	db, cache, end := test.SetupPG(t, bootstrap.InitiliazeDB)
	defer end()
	event.Initialize(event.KafkaConfig{}, cache)

	u, _ := assets.InsertAdminUser(db)
	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, cache, key, key, u)

	//First pipeline
	pip := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip1",
	}
	test.NoError(t, pipeline.InsertPipeline(db, cache, proj, &pip, u))

	s := sdk.NewStage("stage 1")
	s.Enabled = true
	s.PipelineID = pip.ID
	test.NoError(t, pipeline.InsertStage(db, s))

	proj, _ = project.LoadByID(db, cache, proj.ID, u, project.LoadOptions.WithApplications, project.LoadOptions.WithPipelines, project.LoadOptions.WithEnvironments, project.LoadOptions.WithGroups)

	w := sdk.Workflow{
		Name:       "test_purge_1",
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		WorkflowData: &sdk.WorkflowData{
			Node: sdk.Node{
				Name: "node1",
				Ref:  "node1",
				Type: sdk.NodeTypePipeline,
				Context: &sdk.NodeContext{
					PipelineID: pip.ID,
				},
				Triggers: []sdk.NodeTrigger{
					{
						ChildNode: sdk.Node{
							Name: "node2",
							Ref:  "node2",
							Type: sdk.NodeTypePipeline,
							Context: &sdk.NodeContext{
								PipelineID: pip.ID,
							},
						},
					},
				},
			},
		},
		HistoryLength: 2,
		PurgeTags:     []string{"git.branch"},
	}

	(&w).RetroMigrate()

	test.NoError(t, workflow.Insert(db, cache, &w, proj, u))

	w1, err := workflow.Load(context.TODO(), db, cache, proj, "test_purge_1", u, workflow.LoadOptions{
		DeepPipeline: true,
	})
	test.NoError(t, err)

	for i := 0; i < 5; i++ {
		wfr, errWR := workflow.CreateRun(db, w1, nil, u)
		assert.NoError(t, errWR)
		wfr.Workflow = *w1
		_, errWr := workflow.StartWorkflowRun(context.TODO(), db, cache, proj, wfr, &sdk.WorkflowRunPostHandlerOption{
			Manual: &sdk.WorkflowNodeRunManual{
				User: *u,
				Payload: map[string]string{
					"git.branch": "master",
					"git.author": "test",
				},
			},
		}, nil, nil)
		test.NoError(t, errWr)
		wfr.Status = sdk.StatusBuilding.String()
		test.NoError(t, workflow.UpdateWorkflowRunStatus(db, wfr))
	}

	errP := workflow.PurgeWorkflowRun(context.Background(), db, *w1, nil)
	test.NoError(t, errP)

	wruns, _, _, count, errRuns := workflow.LoadRuns(db, proj.Key, w1.Name, 0, 10, nil)
	test.NoError(t, errRuns)
	test.Equal(t, 5, count, "Number of workflow runs isn't correct")

	toDeleteNb := 0
	for _, wfRun := range wruns {
		if wfRun.ToDelete {
			toDeleteNb++
		}
	}

	test.Equal(t, 0, toDeleteNb, "Number of workflow runs to be purged isn't correct")
}

func TestPurgeWorkflowRunWithOneSuccessWorkflowRun(t *testing.T) {
	db, cache, end := test.SetupPG(t, bootstrap.InitiliazeDB)
	defer end()
	event.Initialize(event.KafkaConfig{}, cache)

	u, _ := assets.InsertAdminUser(db)
	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, cache, key, key, u)

	//First pipeline
	pip := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip1",
	}
	test.NoError(t, pipeline.InsertPipeline(db, cache, proj, &pip, u))

	s := sdk.NewStage("stage 1")
	s.Enabled = true
	s.PipelineID = pip.ID
	test.NoError(t, pipeline.InsertStage(db, s))

	proj, _ = project.LoadByID(db, cache, proj.ID, u, project.LoadOptions.WithApplications, project.LoadOptions.WithPipelines, project.LoadOptions.WithEnvironments, project.LoadOptions.WithGroups)

	w := sdk.Workflow{
		Name:       "test_purge_1",
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		WorkflowData: &sdk.WorkflowData{
			Node: sdk.Node{
				Name: "node1",
				Ref:  "node1",
				Type: sdk.NodeTypePipeline,
				Context: &sdk.NodeContext{
					PipelineID: pip.ID,
				},
				Triggers: []sdk.NodeTrigger{
					{
						ChildNode: sdk.Node{
							Name: "node2",
							Ref:  "node2",
							Type: sdk.NodeTypePipeline,
							Context: &sdk.NodeContext{
								PipelineID: pip.ID,
							},
						},
					},
				},
			},
		},
		HistoryLength: 2,
		PurgeTags:     []string{"git.branch"},
	}

	(&w).RetroMigrate()

	test.NoError(t, workflow.Insert(db, cache, &w, proj, u))

	w1, err := workflow.Load(context.TODO(), db, cache, proj, "test_purge_1", u, workflow.LoadOptions{
		DeepPipeline: true,
	})
	test.NoError(t, err)

	wr, errWR := workflow.CreateRun(db, w1, nil, u)
	assert.NoError(t, errWR)
	wr.Workflow = *w1
	_, errWr := workflow.StartWorkflowRun(context.TODO(), db, cache, proj, wr, &sdk.WorkflowRunPostHandlerOption{
		Manual: &sdk.WorkflowNodeRunManual{
			User: *u,
			Payload: map[string]string{
				"git.branch": "master",
				"git.author": "test",
			},
		},
	}, nil, nil)
	test.NoError(t, errWr)

	for i := 0; i < 5; i++ {
		wfr, errWR := workflow.CreateRun(db, w1, nil, u)
		assert.NoError(t, errWR)
		wfr.Workflow = *w1
		_, errWr := workflow.StartWorkflowRun(context.TODO(), db, cache, proj, wfr, &sdk.WorkflowRunPostHandlerOption{
			Manual: &sdk.WorkflowNodeRunManual{
				User: *u,
				Payload: map[string]string{
					"git.branch": "master",
					"git.author": "test",
				},
			},
		}, nil, nil)
		test.NoError(t, errWr)

		wfr.Status = sdk.StatusFail.String()
		test.NoError(t, workflow.UpdateWorkflowRunStatus(db, wfr))
	}

	errP := workflow.PurgeWorkflowRun(context.Background(), db, *w1, nil)
	test.NoError(t, errP)

	wruns, _, _, count, errRuns := workflow.LoadRuns(db, proj.Key, w1.Name, 0, 10, nil)
	test.NoError(t, errRuns)
	test.Equal(t, 6, count, "Number of workflow runs isn't correct")
	toDeleteNb := 0
	wfInSuccess := false
	for _, wfRun := range wruns {
		if wfRun.ToDelete {
			toDeleteNb++
			if wfRun.Status == sdk.StatusSuccess.String() {
				wfInSuccess = true
			}
		}
	}

	test.Equal(t, 3, toDeleteNb, "Number of workflow runs to be purged isn't correct")
	test.Equal(t, false, wfInSuccess, "The workflow should keep at least one workflow run in success")
}

func TestPurgeWorkflowRunWithNoSuccessWorkflowRun(t *testing.T) {
	db, cache, end := test.SetupPG(t, bootstrap.InitiliazeDB)
	defer end()
	event.Initialize(event.KafkaConfig{}, cache)

	u, _ := assets.InsertAdminUser(db)
	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, cache, key, key, u)

	//First pipeline
	pip := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip1",
	}
	test.NoError(t, pipeline.InsertPipeline(db, cache, proj, &pip, u))

	s := sdk.NewStage("stage 1")
	s.Enabled = true
	s.PipelineID = pip.ID
	test.NoError(t, pipeline.InsertStage(db, s))

	proj, _ = project.LoadByID(db, cache, proj.ID, u, project.LoadOptions.WithApplications, project.LoadOptions.WithPipelines, project.LoadOptions.WithEnvironments, project.LoadOptions.WithGroups)

	w := sdk.Workflow{
		Name:       "test_purge_1",
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		WorkflowData: &sdk.WorkflowData{
			Node: sdk.Node{
				Name: "node1",
				Ref:  "node1",
				Type: sdk.NodeTypePipeline,
				Context: &sdk.NodeContext{
					PipelineID: pip.ID,
				},
				Triggers: []sdk.NodeTrigger{
					{
						ChildNode: sdk.Node{
							Name: "node2",
							Ref:  "node2",
							Type: sdk.NodeTypePipeline,
							Context: &sdk.NodeContext{
								PipelineID: pip.ID,
							},
						},
					},
				},
			},
		},
		HistoryLength: 2,
		PurgeTags:     []string{"git.branch"},
	}

	(&w).RetroMigrate()

	test.NoError(t, workflow.Insert(db, cache, &w, proj, u))

	w1, err := workflow.Load(context.TODO(), db, cache, proj, "test_purge_1", u, workflow.LoadOptions{
		DeepPipeline: true,
	})
	test.NoError(t, err)

	for i := 0; i < 5; i++ {
		wfr, errWR := workflow.CreateRun(db, w1, nil, u)
		assert.NoError(t, errWR)
		wfr.Workflow = *w1
		_, errWr := workflow.StartWorkflowRun(context.TODO(), db, cache, proj, wfr, &sdk.WorkflowRunPostHandlerOption{
			Manual: &sdk.WorkflowNodeRunManual{
				User: *u,
				Payload: map[string]string{
					"git.branch": "master",
					"git.author": "test",
				},
			},
		}, nil, nil)
		test.NoError(t, errWr)

		wfr.Status = sdk.StatusFail.String()
		test.NoError(t, workflow.UpdateWorkflowRunStatus(db, wfr))
	}

	errP := workflow.PurgeWorkflowRun(context.Background(), db, *w1, nil)
	test.NoError(t, errP)

	wruns, _, _, count, errRuns := workflow.LoadRuns(db, proj.Key, w1.Name, 0, 10, nil)
	test.NoError(t, errRuns)
	test.Equal(t, 5, count, "Number of workflow runs isn't correct")
	fmt.Printf("%+v\n", wruns)
	toDeleteNb := 0
	for _, wfRun := range wruns {
		if wfRun.ToDelete {
			toDeleteNb++
		}
	}

	test.Equal(t, 3, toDeleteNb, "Number of workflow runs to be purged isn't correct")
}

func TestPurgeWorkflowRunWithoutTags(t *testing.T) {
	db, cache, end := test.SetupPG(t, bootstrap.InitiliazeDB)
	defer end()
	event.Initialize(event.KafkaConfig{}, cache)

	u, _ := assets.InsertAdminUser(db)
	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, cache, key, key, u)

	//First pipeline
	pip := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip1",
	}
	test.NoError(t, pipeline.InsertPipeline(db, cache, proj, &pip, u))

	s := sdk.NewStage("stage 1")
	s.Enabled = true
	s.PipelineID = pip.ID
	test.NoError(t, pipeline.InsertStage(db, s))

	proj, _ = project.LoadByID(db, cache, proj.ID, u, project.LoadOptions.WithApplications, project.LoadOptions.WithPipelines, project.LoadOptions.WithEnvironments, project.LoadOptions.WithGroups)

	w := sdk.Workflow{
		Name:       "test_purge_1",
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		WorkflowData: &sdk.WorkflowData{
			Node: sdk.Node{
				Name: "node1",
				Ref:  "node1",
				Type: sdk.NodeTypePipeline,
				Context: &sdk.NodeContext{
					PipelineID: pip.ID,
				},
				Triggers: []sdk.NodeTrigger{
					{
						ChildNode: sdk.Node{
							Name: "node2",
							Ref:  "node2",
							Type: sdk.NodeTypePipeline,
							Context: &sdk.NodeContext{
								PipelineID: pip.ID,
							},
						},
					},
				},
			},
		},
		HistoryLength: 2,
	}
	(&w).RetroMigrate()
	test.NoError(t, workflow.Insert(db, cache, &w, proj, u))

	w1, err := workflow.Load(context.TODO(), db, cache, proj, "test_purge_1", u, workflow.LoadOptions{
		DeepPipeline: true,
	})
	test.NoError(t, err)

	branches := []string{"master", "master", "master", "develop", "develop", "testBr", "testBr", "testBr", "testBr", "test4"}
	for i := 0; i < 10; i++ {
		wr, errWR := workflow.CreateRun(db, w1, nil, u)
		assert.NoError(t, errWR)
		wr.Workflow = *w1
		_, errWr := workflow.StartWorkflowRun(context.TODO(), db, cache, proj, wr, &sdk.WorkflowRunPostHandlerOption{
			Manual: &sdk.WorkflowNodeRunManual{
				User: *u,
				Payload: map[string]string{
					"git.branch": branches[i],
					"git.author": "test",
				},
			},
		}, nil, nil)
		test.NoError(t, errWr)
	}

	errP := workflow.PurgeWorkflowRun(context.Background(), db, *w1, nil)
	test.NoError(t, errP)

	wruns, _, _, count, errRuns := workflow.LoadRuns(db, proj.Key, w1.Name, 0, 10, nil)
	test.NoError(t, errRuns)
	test.Equal(t, 10, count, "Number of workflow runs isn't correct")

	toDeleteNb := 0
	for _, wfRun := range wruns {
		if wfRun.ToDelete {
			toDeleteNb++
		}
	}

	test.Equal(t, 7, toDeleteNb, "Number of workflow runs to be purged isn't correct (because it should keep at least one in success)")
}

func TestPurgeWorkflowRunWithoutTagsBiggerHistoryLength(t *testing.T) {
	db, cache, end := test.SetupPG(t, bootstrap.InitiliazeDB)
	defer end()
	event.Initialize(event.KafkaConfig{}, cache)

	u, _ := assets.InsertAdminUser(db)
	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, cache, key, key, u)

	//First pipeline
	pip := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip1",
	}
	test.NoError(t, pipeline.InsertPipeline(db, cache, proj, &pip, u))

	s := sdk.NewStage("stage 1")
	s.Enabled = true
	s.PipelineID = pip.ID
	test.NoError(t, pipeline.InsertStage(db, s))

	proj, _ = project.LoadByID(db, cache, proj.ID, u, project.LoadOptions.WithApplications, project.LoadOptions.WithPipelines, project.LoadOptions.WithEnvironments, project.LoadOptions.WithGroups)

	w := sdk.Workflow{
		Name:       "test_purge_1",
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		WorkflowData: &sdk.WorkflowData{
			Node: sdk.Node{
				Name: "node1",
				Ref:  "node1",
				Type: sdk.NodeTypePipeline,
				Context: &sdk.NodeContext{
					PipelineID: pip.ID,
				},
				Triggers: []sdk.NodeTrigger{
					{
						ChildNode: sdk.Node{
							Name: "node2",
							Ref:  "node2",
							Type: sdk.NodeTypePipeline,
							Context: &sdk.NodeContext{
								PipelineID: pip.ID,
							},
						},
					},
				},
			},
		},
		HistoryLength: 20,
	}
	(&w).RetroMigrate()
	test.NoError(t, workflow.Insert(db, cache, &w, proj, u))

	w1, err := workflow.Load(context.TODO(), db, cache, proj, "test_purge_1", u, workflow.LoadOptions{
		DeepPipeline: true,
	})
	test.NoError(t, err)

	branches := []string{"master", "master", "master", "develop", "develop", "testBr", "testBr", "testBr", "testBr", "test4"}
	for i := 0; i < 10; i++ {
		wr, errWR := workflow.CreateRun(db, w1, nil, u)
		assert.NoError(t, errWR)
		wr.Workflow = *w1
		_, errWr := workflow.StartWorkflowRun(context.TODO(), db, cache, proj, wr, &sdk.WorkflowRunPostHandlerOption{
			Manual: &sdk.WorkflowNodeRunManual{
				User: *u,
				Payload: map[string]string{
					"git.branch": branches[i],
					"git.author": "test",
				},
			},
		}, nil, nil)
		test.NoError(t, errWr)
	}

	errP := workflow.PurgeWorkflowRun(context.Background(), db, *w1, nil)
	test.NoError(t, errP)

	wruns, _, _, count, errRuns := workflow.LoadRuns(db, proj.Key, w1.Name, 0, 10, nil)
	test.NoError(t, errRuns)
	test.Equal(t, 10, count, "Number of workflow runs isn't correct")

	toDeleteNb := 0
	for _, wfRun := range wruns {
		if wfRun.ToDelete {
			toDeleteNb++
		}
	}

	test.Equal(t, 0, toDeleteNb, "Number of workflow runs to be purged isn't correct (because it should keep at least one in success)")
}
