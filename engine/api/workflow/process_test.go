package workflow

import (
	"testing"

	"github.com/ovh/cds/sdk"
	"github.com/stretchr/testify/assert"
)

func TestUpdateNodesRunStatus(t *testing.T) {
	var success, building, fail, stop int

	updateNodesRunStatus(sdk.StatusSuccess.String(), &success, &building, &fail, &stop)

	assert.Equal(t, 1, success)
	assert.Equal(t, 0, building)
	assert.Equal(t, 0, fail)
	assert.Equal(t, 0, stop)

	updateNodesRunStatus(sdk.StatusBuilding.String(), &success, &building, &fail, &stop)

	assert.Equal(t, 1, success)
	assert.Equal(t, 1, building)
	assert.Equal(t, 0, fail)
	assert.Equal(t, 0, stop)

	updateNodesRunStatus(sdk.StatusWaiting.String(), &success, &building, &fail, &stop)

	assert.Equal(t, 1, success)
	assert.Equal(t, 2, building)
	assert.Equal(t, 0, fail)
	assert.Equal(t, 0, stop)
}

func TestGetWorkflowRunStatus(t *testing.T) {
	testCases := []struct {
		success  int
		building int
		fail     int
		stop     int
		status   string
	}{
		{success: 1, building: 0, fail: 0, stop: 0, status: sdk.StatusSuccess.String()},
		{success: 1, building: 1, fail: 0, stop: 0, status: sdk.StatusBuilding.String()},
		{success: 1, building: 1, fail: 1, stop: 0, status: sdk.StatusBuilding.String()},
		{success: 1, building: 0, fail: 1, stop: 1, status: sdk.StatusFail.String()},
		{success: 1, building: 0, fail: 0, stop: 1, status: sdk.StatusStopped.String()},
		{success: 1, building: 1, fail: 1, stop: 1, status: sdk.StatusBuilding.String()},
	}

	for _, tc := range testCases {
		status := getWorkflowRunStatus(tc.success, tc.building, tc.fail, tc.stop)
		assert.Equal(t, tc.status, status)
	}
}
