package workflow_test

import (
	"context"
	"testing"

	"github.com/ovh/cds/engine/api/authentication"
	"github.com/ovh/cds/engine/api/bootstrap"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/sdk"
	"github.com/stretchr/testify/require"
)

func TestProcessJoinDefaultCondition(t *testing.T) {
	db, cache := test.SetupPG(t, bootstrap.InitiliazeDB)

	u, _ := assets.InsertAdminUser(t, db)
	consumer, _ := authentication.LoadConsumerByTypeAndUserID(context.TODO(), db, sdk.ConsumerLocal, u.ID, authentication.LoadConsumerOptions.WithAuthentifiedUser)

	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, cache, key, key)

	// Test data
	wr := &sdk.WorkflowRun{
		ProjectID:        proj.ID,
		WorkflowNodeRuns: map[int64][]sdk.WorkflowNodeRun{},
		Workflow: sdk.Workflow{
			Name:       "myworkflow",
			ProjectKey: key,
			ProjectID:  proj.ID,
			WorkflowData: sdk.WorkflowData{
				Node: sdk.Node{
					ID:   1,
					Name: "myfork",
				},
				Joins: []sdk.Node{
					{
						Name: "myjoin",
						ID:   666,
						JoinContext: []sdk.NodeJoin{
							{
								ParentID: 1,
							},
						},
					},
				},
			},
		},
	}

	// Insert workflow
	require.NoError(t, workflow.Insert(context.TODO(), db, cache, *proj, &wr.Workflow))

	// Create run
	wrr, err := workflow.CreateRun(db.DbMap, &wr.Workflow, sdk.WorkflowRunPostHandlerOption{AuthConsumerID: consumer.ID})
	require.NoError(t, err)
	wr.ID = wrr.ID
	wr.WorkflowID = wr.Workflow.ID
	require.NoError(t, workflow.UpdateWorkflowRun(context.TODO(), db, wr))

	// Start workflow
	_, err = workflow.StartWorkflowRun(context.TODO(), db, cache, *proj, wr, &sdk.WorkflowRunPostHandlerOption{Manual: &sdk.WorkflowNodeRunManual{}}, *consumer, nil)
	require.NoError(t, err)

	wrUpdated, err := workflow.LoadRun(context.TODO(), db, proj.Key, wr.Workflow.Name, wr.Number, workflow.LoadRunOptions{})
	require.NoError(t, err)

	// Fork and Join has been run
	require.Equal(t, 2, len(wrUpdated.WorkflowNodeRuns))
	require.Equal(t, sdk.StatusSuccess, wrUpdated.WorkflowNodeRuns[wr.Workflow.WorkflowData.Joins[0].ID][0].Status)
}

func TestProcessJoinCustomCondition(t *testing.T) {
	db, cache := test.SetupPG(t, bootstrap.InitiliazeDB)

	u, _ := assets.InsertAdminUser(t, db)
	consumer, _ := authentication.LoadConsumerByTypeAndUserID(context.TODO(), db, sdk.ConsumerLocal, u.ID, authentication.LoadConsumerOptions.WithAuthentifiedUser)

	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, cache, key, key)

	// Test data
	wr := &sdk.WorkflowRun{
		ProjectID:        proj.ID,
		WorkflowNodeRuns: map[int64][]sdk.WorkflowNodeRun{},
		Workflow: sdk.Workflow{
			Name:       "myworkflow",
			ProjectKey: key,
			ProjectID:  proj.ID,
			WorkflowData: sdk.WorkflowData{
				Node: sdk.Node{
					ID:   1,
					Name: "myfork",
				},
				Joins: []sdk.Node{
					{
						Name: "myjoin",
						ID:   666,
						JoinContext: []sdk.NodeJoin{
							{
								ParentID: 1,
							},
						},
						Context: &sdk.NodeContext{
							Conditions: sdk.WorkflowNodeConditions{
								PlainConditions: []sdk.WorkflowNodeCondition{
									{
										Variable: "cds.status",
										Operator: "eq",
										Value:    sdk.StatusFail,
									},
								},
							},
						},
					},
				},
			},
		},
	}

	// Insert workflow
	require.NoError(t, workflow.Insert(context.TODO(), db, cache, *proj, &wr.Workflow))

	// Create run
	wrr, err := workflow.CreateRun(db.DbMap, &wr.Workflow, sdk.WorkflowRunPostHandlerOption{AuthConsumerID: consumer.ID})
	require.NoError(t, err)
	wr.ID = wrr.ID
	wr.WorkflowID = wr.Workflow.ID

	require.NoError(t, workflow.UpdateWorkflowRun(context.TODO(), db, wr))

	// Start run
	_, err = workflow.StartWorkflowRun(context.TODO(), db, cache, *proj, wr, &sdk.WorkflowRunPostHandlerOption{Manual: &sdk.WorkflowNodeRunManual{}}, *consumer, nil)
	require.NoError(t, err)

	wrUpdated, err := workflow.LoadRun(context.TODO(), db, proj.Key, wr.Workflow.Name, wr.Number, workflow.LoadRunOptions{})
	require.NoError(t, err)

	// Only fork has run
	require.Equal(t, 1, len(wrUpdated.WorkflowNodeRuns))
	require.Equal(t, sdk.StatusSuccess, wrUpdated.WorkflowNodeRuns[wr.Workflow.WorkflowData.Node.ID][0].Status)
}
