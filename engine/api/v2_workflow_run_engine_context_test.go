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
