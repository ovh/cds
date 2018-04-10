package sdk

import (
	"testing"

	"fmt"

	"github.com/stretchr/testify/assert"
)

func TestWorkflowNode_AncestorsWithTriggers(t *testing.T) {
	w := Workflow{
		ID: 1,
	}
	w.Root = &WorkflowNode{
		ID: 1,
	}

	node4 := WorkflowNode{
		ID: 4,
	}

	w.Root.Triggers = []WorkflowNodeTrigger{
		{
			WorkflowDestNode: WorkflowNode{
				ID: 2,
				Triggers: []WorkflowNodeTrigger{
					{
						WorkflowDestNode: WorkflowNode{
							ID: 3,
							Triggers: []WorkflowNodeTrigger{
								{
									WorkflowDestNode: node4,
								},
							},
						},
					},
				},
			},
		},
	}

	ids := node4.Ancestors(&w, true)
	t.Logf("Deep node4.Ancestors: %v\n", ids)
	assert.Equal(t, 3, len(ids))

	ids = node4.Ancestors(&w, false)
	t.Logf("Not deep node4.Ancestors: %v\n", ids)
	assert.Equal(t, 1, len(ids))
	assert.Equal(t, 3, ids[0])
}

func TestWorkflowNode_AncestorsDirectAfterJoin(t *testing.T) {
	w := Workflow{
		ID: 1,
	}
	w.Root = &WorkflowNode{
		ID: 1,
		Triggers: []WorkflowNodeTrigger{
			{
				WorkflowDestNode: WorkflowNode{
					ID: 2,
				},
			},
			{
				WorkflowDestNode: WorkflowNode{
					ID: 3,
				},
			},
		},
	}

	node4 := WorkflowNode{
		ID: 4,
	}

	w.Joins = []WorkflowNodeJoin{
		{
			SourceNodeIDs: []int64{2, 3},
			Triggers: []WorkflowNodeJoinTrigger{
				{
					WorkflowDestNode: node4,
				},
			},
		},
	}

	ids := node4.Ancestors(&w, true)
	t.Logf("Deep node4.Ancestors: %v\n", ids)
	assert.Equal(t, 3, len(ids))

	ids = node4.Ancestors(&w, false)
	t.Logf("Not deep node4.Ancestors: %v\n", ids)
	assert.Equal(t, 2, len(ids))

	if !((ids[0] == 2 && ids[1] == 3) || (ids[0] == 3 && ids[1] == 2)) {
		assert.Error(t, fmt.Errorf("Wrong parent ID"))
	}
}

func TestWorkflowNode_AncestorsAfterJoin(t *testing.T) {
	w := Workflow{
		ID: 1,
	}
	w.Root = &WorkflowNode{
		ID: 1,
		Triggers: []WorkflowNodeTrigger{
			{
				WorkflowDestNode: WorkflowNode{
					ID: 2,
				},
			},
			{
				WorkflowDestNode: WorkflowNode{
					ID: 3,
				},
			},
		},
	}

	node5 := WorkflowNode{
		ID: 5,
	}

	w.Joins = []WorkflowNodeJoin{
		{
			SourceNodeIDs: []int64{2, 3},
			Triggers: []WorkflowNodeJoinTrigger{
				{
					WorkflowDestNode: WorkflowNode{
						ID: 4,
						Triggers: []WorkflowNodeTrigger{
							{
								WorkflowDestNode: node5,
							},
						},
					},
				},
			},
		},
	}

	ids := node5.Ancestors(&w, true)
	t.Logf("Deep node5.Ancestors: %v\n", ids)
	assert.Equal(t, 4, len(ids))

	ids = node5.Ancestors(&w, false)
	t.Logf("Not deep node5.Ancestors: %v\n", ids)
	assert.Equal(t, 1, len(ids))
	assert.Equal(t, int64(4), ids[0])
}
