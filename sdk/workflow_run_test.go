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
