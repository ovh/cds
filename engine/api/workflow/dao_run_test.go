package workflow_test

import (
	"testing"

	"github.com/ovh/cds/engine/api/bootstrap"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/sdk"
)

func TestCanBeRun(t *testing.T) {
	wnrs := map[int64][]sdk.WorkflowNodeRun{}
	node1 := sdk.WorkflowNode{ID: 25}
	nodeRoot := &sdk.WorkflowNode{
		ID: 10,
		Triggers: []sdk.WorkflowNodeTrigger{
			{
				WorkflowDestNode: node1,
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
			Root:       nodeRoot,
			RootID:     10,
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
	db, cache := test.SetupPG(t, bootstrap.InitiliazeDB)
	u, _ := assets.InsertAdminUser(db)
	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, cache, key, key, u)

	//First pipeline
	pip := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip1",
		Type:       sdk.BuildPipeline,
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
		HistoryLength: 2,
		PurgeTags:     []string{"git.branch"},
	}

	test.NoError(t, workflow.Insert(db, cache, &w, proj, u))

	w1, err := workflow.Load(db, cache, key, "test_purge_1", u, workflow.LoadOptions{
		DeepPipeline: true,
	})
	test.NoError(t, err)

	for i := 0; i < 5; i++ {
		_, errWr := workflow.ManualRun(db, db, cache, proj, w1, &sdk.WorkflowNodeRunManual{
			User: *u,
			Payload: map[string]string{
				"git.branch": "master",
				"git.author": "test",
			},
		}, nil)
		test.NoError(t, errWr)
	}

	errP := workflow.PurgeWorkflowRun(db, *w1)
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

func TestPurgeWorkflowRunWithoutTags(t *testing.T) {
	db, cache := test.SetupPG(t, bootstrap.InitiliazeDB)
	u, _ := assets.InsertAdminUser(db)
	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, cache, key, key, u)

	//First pipeline
	pip := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip1",
		Type:       sdk.BuildPipeline,
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
		HistoryLength: 2,
	}

	test.NoError(t, workflow.Insert(db, cache, &w, proj, u))

	w1, err := workflow.Load(db, cache, key, "test_purge_1", u, workflow.LoadOptions{
		DeepPipeline: true,
	})
	test.NoError(t, err)

	branches := []string{"master", "master", "master", "develop", "develop", "testBr", "testBr", "testBr", "testBr", "test4"}
	for i := 0; i < 10; i++ {
		_, errWr := workflow.ManualRun(db, db, cache, proj, w1, &sdk.WorkflowNodeRunManual{
			User: *u,
			Payload: map[string]string{
				"git.branch": branches[i],
				"git.author": "test",
			},
		}, nil)
		test.NoError(t, errWr)
	}

	errP := workflow.PurgeWorkflowRun(db, *w1)
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

	test.Equal(t, 8, toDeleteNb, "Number of workflow runs to be purged isn't correct")
}
