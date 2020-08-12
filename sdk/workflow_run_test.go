package sdk

import (
	"testing"
	"time"

	"github.com/ovh/venom"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
						Status: StatusSuccess,
					},
					{
						Job: ExecutedJob{
							Job: Job{
								Action: Action{
									Name: "job 2",
								},
							},
						},
						Status: StatusFail,
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
	s, err := wfr.Report()
	assert.NoError(t, err)
	t.Log(s)

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
						Status: StatusSuccess,
					},
					{
						Job: ExecutedJob{
							Job: Job{
								Action: Action{
									Name: "job 2",
								},
							},
						},
						Status: StatusFail,
					},
				},
			},
		},
		Tests:  nil,
		Status: StatusSuccess,
	}

	s, err = wfr.Report()
	assert.NoError(t, err)
	t.Log(s)

	wfr = WorkflowNodeRun{
		Stages: []Stage{
			{
				Name: "stage 1",
			},
		},
		Tests: nil,
	}

	s, err = wfr.Report()
	assert.NoError(t, err)
	t.Log(s)
}

func TestWorkflowQueue_Sort(t *testing.T) {
	now := time.Now()
	t10, _ := time.Parse(time.RFC3339, "2018-09-01T10:00:00+00:00")
	t11, _ := time.Parse(time.RFC3339, "2018-09-01T11:00:00+00:00")
	t12, _ := time.Parse(time.RFC3339, "2018-09-01T12:00:00+00:00")
	t13, _ := time.Parse(time.RFC3339, "2018-09-01T13:00:00+00:00")
	t14, _ := time.Parse(time.RFC3339, "2018-09-01T14:00:00+00:00")
	t15, _ := time.Parse(time.RFC3339, "2018-09-01T15:00:00+00:00")
	t16, _ := time.Parse(time.RFC3339, "2018-09-01T16:00:00+00:00")

	tests := []struct {
		name     string
		q        WorkflowQueue
		expected WorkflowQueue
	}{
		{
			name: "test sort 1",
			q: WorkflowQueue{
				{
					ProjectID:     1,
					ID:            1,
					Queued:        t10,
					QueuedSeconds: now.Unix() - t10.Unix(),
				},
				{
					ProjectID:     1,
					ID:            2,
					Queued:        t11,
					QueuedSeconds: now.Unix() - t11.Unix(),
				},
				{
					ProjectID:     2,
					ID:            3,
					Queued:        t12,
					QueuedSeconds: now.Unix() - t12.Unix(),
				},
				{
					ProjectID:     1,
					ID:            4,
					Queued:        t13,
					QueuedSeconds: now.Unix() - t13.Unix(),
				},
				{
					ProjectID:     1,
					ID:            5,
					Queued:        t14,
					QueuedSeconds: now.Unix() - t14.Unix(),
				},
				{
					ProjectID:     2,
					ID:            6,
					Queued:        t15,
					QueuedSeconds: now.Unix() - t15.Unix(),
				},
				{
					ProjectID:     1,
					ID:            7,
					Queued:        t16,
					QueuedSeconds: now.Unix() - t16.Unix(),
				},
			},
			expected: WorkflowQueue{
				{
					ProjectID:     2,
					ID:            3,
					Queued:        t12,
					QueuedSeconds: now.Unix() - t12.Unix(),
				},
				{
					ProjectID:     2,
					ID:            6,
					Queued:        t15,
					QueuedSeconds: now.Unix() - t15.Unix(),
				},
				{
					ProjectID:     1,
					ID:            1,
					Queued:        t10,
					QueuedSeconds: now.Unix() - t10.Unix(),
				},
				{
					ProjectID:     1,
					ID:            2,
					Queued:        t11,
					QueuedSeconds: now.Unix() - t11.Unix(),
				},
				{
					ProjectID:     1,
					ID:            4,
					Queued:        t13,
					QueuedSeconds: now.Unix() - t13.Unix(),
				},
				{
					ProjectID:     1,
					ID:            5,
					Queued:        t14,
					QueuedSeconds: now.Unix() - t14.Unix(),
				},
				{
					ProjectID:     1,
					ID:            7,
					Queued:        t16,
					QueuedSeconds: now.Unix() - t16.Unix(),
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

func TestWorkflowRunVersion_IsValid(t *testing.T) {
	require.NoError(t, WorkflowRunVersion{Value: "1.2.3"}.IsValid())
	require.NoError(t, WorkflowRunVersion{Value: "1.2.3-snapshot.1"}.IsValid())
	require.Error(t, WorkflowRunVersion{Value: "1.2.3.4"}.IsValid())
}
