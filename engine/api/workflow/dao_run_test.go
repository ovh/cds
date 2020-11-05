package workflow_test

import (
	"context"
	"github.com/ovh/cds/engine/api/bootstrap"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/sdk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
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
		{ID: 3, WorkflowNodeID: nodeRoot.ID, Status: sdk.StatusBuilding},
	}
	wnrs[node1.ID] = []sdk.WorkflowNodeRun{
		{ID: 3, WorkflowNodeID: node1.ID, Status: sdk.StatusFail},
	}
	wr := &sdk.WorkflowRun{
		Workflow: sdk.Workflow{
			Name:       "test_1",
			ProjectID:  1,
			ProjectKey: "key",
			WorkflowData: sdk.WorkflowData{
				Node: nodeRoot,
			},
		},
		WorkflowID:       2,
		WorkflowNodeRuns: wnrs,
	}

	wnr := &sdk.WorkflowNodeRun{
		WorkflowNodeID: node1.ID,
		Status:         sdk.StatusSuccess, // a node node always have a status
	}

	ts := []struct {
		status   string
		canBeRun bool
	}{
		{status: sdk.StatusBuilding, canBeRun: false},
		{status: sdk.StatusSuccess, canBeRun: true},
	}

	for _, tc := range ts {
		wnrs[nodeRoot.ID][0].Status = tc.status
		test.Equal(t, tc.canBeRun, workflow.CanBeRun(wr, wnr))
	}
}

func TestLoadRunsIDsToDelete(t *testing.T) {
	db, cache := test.SetupPG(t, bootstrap.InitiliazeDB)

	_, _ = db.Exec("update workflow_run set to_delete=false ")

	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, cache, key, key)

	w := assets.InsertTestWorkflow(t, db, cache, proj, sdk.RandomString(10))

	wr1, err := workflow.CreateRun(db.DbMap, w, sdk.WorkflowRunPostHandlerOption{Hook: &sdk.WorkflowNodeRunHookEvent{}})
	assert.NoError(t, err)

	wr2, err := workflow.CreateRun(db.DbMap, w, sdk.WorkflowRunPostHandlerOption{Hook: &sdk.WorkflowNodeRunHookEvent{}})
	assert.NoError(t, err)

	wr1.ToDelete = true
	wr2.ToDelete = true

	require.NoError(t, workflow.UpdateWorkflowRun(context.TODO(), db, wr1))
	require.NoError(t, workflow.UpdateWorkflowRun(context.TODO(), db, wr2))

	ids, offset, limit, count, err := workflow.LoadRunsIDsToDelete(db, 0, 1)
	require.NoError(t, err)
	require.Len(t, ids, 1)
	require.Equal(t, ids[0], wr1.ID)
	require.Equal(t, int64(0), offset)
	require.Equal(t, int64(1), limit)
	require.Equal(t, int64(2), count)

	ids, offset, limit, count, err = workflow.LoadRunsIDsToDelete(db, 1, 1)
	require.NoError(t, err)
	require.Len(t, ids, 1)
	require.Equal(t, ids[0], wr2.ID)
	require.Equal(t, int64(1), offset)
	require.Equal(t, int64(1), limit)
	require.Equal(t, int64(2), count)

	ids, offset, limit, count, err = workflow.LoadRunsIDsToDelete(db, 0, 50)
	require.NoError(t, err)
	require.Len(t, ids, 2)

}
