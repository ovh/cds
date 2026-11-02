package forgejo

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ovh/cds/sdk"
)

func TestGetForgejoStateFromStatus(t *testing.T) {
	require.Equal(t, StatusPending, getForgejoStateFromStatus(sdk.V2WorkflowRunStatusCrafting))
	require.Equal(t, StatusPending, getForgejoStateFromStatus(sdk.V2WorkflowRunStatusBlocked))
	require.Equal(t, StatusPending, getForgejoStateFromStatus(sdk.V2WorkflowRunStatusBuilding))
	require.Equal(t, StatusSuccess, getForgejoStateFromStatus(sdk.V2WorkflowRunStatusSuccess))
	require.Equal(t, StatusFailure, getForgejoStateFromStatus(sdk.V2WorkflowRunStatusFail))
	require.Equal(t, StatusFailure, getForgejoStateFromStatus(sdk.V2WorkflowRunStatusStopped))
	require.Equal(t, StatusFailure, getForgejoStateFromStatus(sdk.V2WorkflowRunStatusCancelled))
	require.Equal(t, StatusWarning, getForgejoStateFromStatus(sdk.V2WorkflowRunStatusSkipped))
}
