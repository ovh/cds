package workflow

import (
	"testing"

	"github.com/ovh/cds/engine/api/test"
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
		test.Equal(t, canBeRun(wr, wnr), tc.canBeRun)
	}
}
