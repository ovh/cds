package workflow_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ovh/cds/engine/api/authentication"
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
	db, cache := test.SetupPG(t, bootstrap.InitiliazeDB)

	_ = event.Initialize(context.Background(), db.DbMap, cache)

	u, _ := assets.InsertAdminUser(t, db)
	consumer, _ := authentication.LoadConsumerByTypeAndUserID(context.TODO(), db, sdk.ConsumerLocal, u.ID, authentication.LoadConsumerOptions.WithAuthentifiedUser)

	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, cache, key, key)

	//First pipeline
	pip := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip1",
	}
	test.NoError(t, pipeline.InsertPipeline(db, &pip))

	s := sdk.NewStage("stage 1")
	s.Enabled = true
	s.PipelineID = pip.ID
	test.NoError(t, pipeline.InsertStage(db, s))

	proj, _ = project.LoadByID(db, proj.ID, project.LoadOptions.WithApplications, project.LoadOptions.WithPipelines, project.LoadOptions.WithEnvironments, project.LoadOptions.WithGroups)

	w := sdk.Workflow{
		Name:       "test_staticfiles_1",
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		WorkflowData: sdk.WorkflowData{
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

	test.NoError(t, workflow.Insert(context.TODO(), db, cache, *proj, &w))

	w1, err := workflow.Load(context.TODO(), db, cache, *proj, "test_staticfiles_1", workflow.LoadOptions{
		DeepPipeline: true,
	})
	test.NoError(t, err)

	wfr, errWR := workflow.CreateRun(db.DbMap, w1, sdk.WorkflowRunPostHandlerOption{AuthConsumerID: consumer.ID})
	assert.NoError(t, errWR)
	wfr.Workflow = *w1
	_, errWr := workflow.StartWorkflowRun(context.TODO(), db, cache, *proj, wfr, &sdk.WorkflowRunPostHandlerOption{
		Manual: &sdk.WorkflowNodeRunManual{Username: u.Username},
	}, *consumer, nil)
	test.NoError(t, errWr)

	var stFile sdk.StaticFiles

	var nrss []sdk.WorkflowNodeRun

	for _, nrs := range wfr.WorkflowNodeRuns {
		nrss = append(nrss, nrs...)
	}

	for _, nr := range nrss {
		stFile = sdk.StaticFiles{
			Name:       "mywebsite",
			NodeRunID:  nr.ID,
			PublicURL:  "http://mypublicurl.com",
			EntryPoint: "index.html",
		}
		test.NoError(t, workflow.InsertStaticFiles(db, &stFile))
		break
	}

	assert.NotZero(t, stFile.ID)
}
