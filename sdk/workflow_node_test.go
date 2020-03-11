package sdk

import (
	"fmt"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAncestorsSimple(t *testing.T) {
	w := Workflow{
		WorkflowData: WorkflowData{
			Node: Node{
				ID:   1,
				Type: NodeTypePipeline,
				Triggers: []NodeTrigger{
					{
						ChildNode: Node{
							ID:   2,
							Type: NodeTypePipeline,
						},
					},
					{
						ChildNode: Node{
							ID:   3,
							Type: NodeTypePipeline,
						},
					},
					{
						ChildNode: Node{
							ID:   4,
							Type: NodeTypePipeline,
							Triggers: []NodeTrigger{
								{
									ChildNode: Node{
										ID:   5,
										Type: NodeTypeFork,
										Triggers: []NodeTrigger{
											{
												ChildNode: Node{
													ID:   6,
													Type: NodeTypePipeline,
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			Joins: []Node{
				{
					ID:   7,
					Type: NodeTypeJoin,
					JoinContext: []NodeJoin{
						{
							ParentID: 2,
						},
						{
							ParentID: 3,
						},
					},
					Triggers: []NodeTrigger{
						{
							ChildNode: Node{
								ID:   8,
								Type: NodeTypePipeline,
								Triggers: []NodeTrigger{
									{
										ChildNode: Node{
											ID:   9,
											Type: NodeTypePipeline,
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	tests := []struct {
		childID int64
		want    []int64
	}{
		{
			childID: 1,
			want:    []int64{},
		},
		{
			childID: 2,
			want:    []int64{1},
		},
		{
			childID: 5,
			want:    []int64{4},
		},
		{
			childID: 6,
			want:    []int64{5},
		},
		{
			childID: 7,
			want:    []int64{2, 3},
		},
		{
			childID: 8,
			want:    []int64{7},
		},
		{
			childID: 9,
			want:    []int64{8},
		},
	}

	for i, tt := range tests {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			n := w.WorkflowData.NodeByID(tt.childID)
			ids := n.Ancestors(w.WorkflowData)

			int64AsIntValues := make([]int, len(ids))
			for i, val := range ids {
				int64AsIntValues[i] = int(val)
			}
			sort.Ints(int64AsIntValues)

			for i, val := range int64AsIntValues {
				ids[i] = int64(val)
			}

			assert.Equal(t, ids, tt.want)
		})

	}
}
