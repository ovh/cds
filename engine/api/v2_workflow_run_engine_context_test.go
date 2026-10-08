package api

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ovh/cds/sdk"
)

// The status of a matrix job is the most severe of its permutations, whatever the order they are loaded in.
func TestComputeExistingRunJobContexts_MatrixStatusIsOrderIndependent(t *testing.T) {
	jobDef := sdk.V2Job{Strategy: &sdk.V2JobStrategy{Matrix: map[string]interface{}{"os": []string{"linux", "windows"}}}}
	permutation := func(os string, status sdk.V2WorkflowRunJobStatus) sdk.V2WorkflowRunJob {
		return sdk.V2WorkflowRunJob{ID: sdk.UUID(), JobID: "build", Job: jobDef, Matrix: sdk.JobMatrix{"os": os}, Status: status}
	}

	cases := []struct {
		name     string
		statuses [2]sdk.V2WorkflowRunJobStatus
		expected sdk.V2WorkflowRunJobStatus
	}{
		{"fail and cancelled", [2]sdk.V2WorkflowRunJobStatus{sdk.V2WorkflowRunJobStatusFail, sdk.V2WorkflowRunJobStatusCancelled}, sdk.V2WorkflowRunJobStatusFail},
		{"cancelled and success", [2]sdk.V2WorkflowRunJobStatus{sdk.V2WorkflowRunJobStatusCancelled, sdk.V2WorkflowRunJobStatusSuccess}, sdk.V2WorkflowRunJobStatusCancelled},
		{"stopped and fail", [2]sdk.V2WorkflowRunJobStatus{sdk.V2WorkflowRunJobStatusStopped, sdk.V2WorkflowRunJobStatusFail}, sdk.V2WorkflowRunJobStatusStopped},
		{"success and skipped", [2]sdk.V2WorkflowRunJobStatus{sdk.V2WorkflowRunJobStatusSuccess, sdk.V2WorkflowRunJobStatusSkipped}, sdk.V2WorkflowRunJobStatusSuccess},
		{"both success", [2]sdk.V2WorkflowRunJobStatus{sdk.V2WorkflowRunJobStatusSuccess, sdk.V2WorkflowRunJobStatusSuccess}, sdk.V2WorkflowRunJobStatusSuccess},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			first := permutation("linux", c.statuses[0])
			second := permutation("windows", c.statuses[1])
			for _, runJobs := range [][]sdk.V2WorkflowRunJob{{first, second}, {second, first}} {
				jobsContext, _ := computeExistingRunJobContexts(context.TODO(), runJobs, nil)
				require.Equal(t, c.expected, jobsContext["build"].Result)
			}
		})
	}
}

// The outgoing hook event carries one conclusion per job, the most severe one for a matrix job.
func TestHookJobConclusions_MatrixUsesMostSevereStatus(t *testing.T) {
	runJobs := []sdk.V2WorkflowRunJob{
		{JobID: "build", Matrix: sdk.JobMatrix{"os": "linux"}, Status: sdk.V2WorkflowRunJobStatusSuccess},
		{JobID: "build", Matrix: sdk.JobMatrix{"os": "windows"}, Status: sdk.V2WorkflowRunJobStatusFail},
		{JobID: "lint", Status: sdk.V2WorkflowRunJobStatusSuccess},
	}
	jobs := hookJobConclusions(runJobs)
	require.Len(t, jobs, 2)
	require.Equal(t, string(sdk.V2WorkflowRunJobStatusFail), jobs["build"].Conclusion)
	require.Equal(t, string(sdk.V2WorkflowRunJobStatusSuccess), jobs["lint"].Conclusion)
}
