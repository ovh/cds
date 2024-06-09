package workflow

import (
	"testing"

	"github.com/ovh/cds/sdk"
	"github.com/stretchr/testify/assert"
)

func TestComputeRunStatus(t *testing.T) {
	runStatus := &statusCounter{}
	computeRunStatus(sdk.StatusSuccess, runStatus)

	assert.Equal(t, 1, runStatus.success)
	assert.Equal(t, 0, runStatus.building)
	assert.Equal(t, 0, runStatus.failed)
	assert.Equal(t, 0, runStatus.stopped)

	computeRunStatus(sdk.StatusBuilding, runStatus)

	assert.Equal(t, 1, runStatus.success)
	assert.Equal(t, 1, runStatus.building)
	assert.Equal(t, 0, runStatus.failed)
	assert.Equal(t, 0, runStatus.stopped)

	computeRunStatus(sdk.StatusWaiting, runStatus)

	assert.Equal(t, 1, runStatus.success)
	assert.Equal(t, 2, runStatus.building)
	assert.Equal(t, 0, runStatus.failed)
	assert.Equal(t, 0, runStatus.stopped)
}

func TestGetWorkflowRunStatus(t *testing.T) {
	testCases := []struct {
		runStatus statusCounter
		status    string
	}{
		{runStatus: statusCounter{success: 1, building: 0, failed: 0, stopped: 0}, status: sdk.StatusSuccess},
		{runStatus: statusCounter{success: 1, building: 1, failed: 0, stopped: 0}, status: sdk.StatusBuilding},
		{runStatus: statusCounter{success: 1, building: 1, failed: 1, stopped: 0}, status: sdk.StatusBuilding},
		{runStatus: statusCounter{success: 1, building: 0, failed: 1, stopped: 1}, status: sdk.StatusFail},
		{runStatus: statusCounter{success: 1, building: 0, failed: 0, stopped: 1}, status: sdk.StatusStopped},
		{runStatus: statusCounter{success: 1, building: 1, failed: 1, stopped: 1}, status: sdk.StatusBuilding},
		{runStatus: statusCounter{success: 1, building: 1, failed: 1, stopped: 1, skipped: 1}, status: sdk.StatusBuilding},
		{runStatus: statusCounter{success: 0, building: 0, failed: 1, stopped: 0, skipped: 1}, status: sdk.StatusFail},
		{runStatus: statusCounter{success: 0, building: 0, failed: 0, stopped: 0, skipped: 1}, status: sdk.StatusSkipped},
		{runStatus: statusCounter{success: 0, building: 0, failed: 0, stopped: 0, skipped: 1, disabled: 1}, status: sdk.StatusSkipped},
		{runStatus: statusCounter{success: 0, building: 0, failed: 0, stopped: 0, skipped: 0, disabled: 1}, status: sdk.StatusDisabled},
		{status: sdk.StatusNeverBuilt},
	}

	for _, tc := range testCases {
		status := getRunStatus(tc.runStatus)
		assert.Equal(t, tc.status, status)
	}
}
