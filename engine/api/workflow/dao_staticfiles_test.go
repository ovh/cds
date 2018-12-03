package workflow_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ovh/cds/engine/api/bootstrap"
	"github.com/ovh/cds/engine/api/event"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/sdk"
)

func TestInsertStaticFiles(t *testing.T) {
	db, cache, end := test.SetupPG(t, bootstrap.InitiliazeDB)
	defer end()
	_ = event.Initialize(event.KafkaConfig{}, cache)

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

	wfr, _, errWr := workflow.ManualRun(context.TODO(), db, cache, proj, w1, &sdk.WorkflowNodeRunManual{User: *u}, nil)
	test.NoError(t, errWr)

	stFile := sdk.StaticFiles{
		Name:       "mywebsite",
		NodeRunID:  wfr.ID,
		PublicURL:  "http://mypublicurl.com",
		EntryPoint: "index.html",
	}
	test.NoError(t, workflow.InsertStaticFiles(db, &stFile))
	assert.NotZero(t, stFile.ID)
}
