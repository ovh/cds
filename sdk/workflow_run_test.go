package sdk

import (
	"testing"

	"github.com/ovh/venom"
	"github.com/stretchr/testify/assert"
)

func TestWorkflowRunTag(t *testing.T) {
	wfr := WorkflowRun{}

	assert.True(t, wfr.Tag("environment", "preproduction"))
	assert.False(t, wfr.Tag("environment", "production"))
	assert.Equal(t, 1, len(wfr.Tags))
	assert.Equal(t, "environment", wfr.Tags[0].Tag)
	assert.Equal(t, "preproduction,production", wfr.Tags[0].Value)
}

func TestWorkflowRunReport(t *testing.T) {
	wfr := WorkflowNodeRun{
		Stages: []Stage{
			{
				Name: "stage 1",
				RunJobs: []WorkflowNodeJobRun{
					{
						Job: ExecutedJob{
							Job: Job{
								Action: Action{
									Name: "job 1",
								},
							},
						},
						Status: StatusSuccess.String(),
					},
					{
						Job: ExecutedJob{
							Job: Job{
								Action: Action{
									Name: "job 2",
								},
							},
						},
						Status: StatusFail.String(),
					},
				},
			},
		},
		Tests: &venom.Tests{
			TotalKO: 1,
			TestSuites: []venom.TestSuite{
				{
					Name: "Test suite1",
					TestCases: []venom.TestCase{
						{
							Name: "test case 1",
							Errors: []venom.Failure{
								{},
							},
							Failures: []venom.Failure{
								{},
							},
						},
						{
							Name: "test case 2",
						},
					},
				},
			},
		},
	}

	wfr = WorkflowNodeRun{
		Stages: []Stage{
			{
				Name: "stage 1",
				RunJobs: []WorkflowNodeJobRun{
					{
						Job: ExecutedJob{
							Job: Job{
								Action: Action{
									Name: "job 1",
								},
							},
						},
						Status: StatusSuccess.String(),
					},
					{
						Job: ExecutedJob{
							Job: Job{
								Action: Action{
									Name: "job 2",
								},
							},
						},
						Status: StatusFail.String(),
					},
				},
			},
		},
		Tests: nil,
	}

	s, err := wfr.Report()
	assert.NoError(t, err)
	t.Log(s)
}

func TestWorkflowQueue_Sort(t *testing.T) {
	tests := []struct {
		name     string
		q        WorkflowQueue
		expected WorkflowQueue
	}{
		{
			name: "test sort 1",
			q: WorkflowQueue{
				{
					ProjectID: 1,
					ID:        1,
				},
				{
					ProjectID: 1,
					ID:        2,
				},
				{
					ProjectID: 2,
					ID:        3,
				},
			},
			expected: WorkflowQueue{
				{
					ProjectID: 2,
					ID:        3,
				},
				{
					ProjectID: 1,
					ID:        1,
				},
				{
					ProjectID: 1,
					ID:        2,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.q.Sort()
			assert.Equal(t, tt.expected, tt.q)
		})
	}
}
