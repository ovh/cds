package gitea

import (
	"testing"

	"code.gitea.io/sdk/gitea"
	"github.com/stretchr/testify/require"

	"github.com/ovh/cds/sdk"
)

func TestGetGiteaStateFromStatus(t *testing.T) {
	require.Equal(t, gitea.StatusPending, getGiteaStateFromStatus(sdk.V2WorkflowRunStatusCrafting))
	require.Equal(t, gitea.StatusPending, getGiteaStateFromStatus(sdk.V2WorkflowRunStatusBlocked))
	require.Equal(t, gitea.StatusPending, getGiteaStateFromStatus(sdk.V2WorkflowRunStatusBuilding))
	require.Equal(t, gitea.StatusSuccess, getGiteaStateFromStatus(sdk.V2WorkflowRunStatusSuccess))
	require.Equal(t, gitea.StatusFailure, getGiteaStateFromStatus(sdk.V2WorkflowRunStatusFail))
	require.Equal(t, gitea.StatusFailure, getGiteaStateFromStatus(sdk.V2WorkflowRunStatusStopped))
	require.Equal(t, gitea.StatusFailure, getGiteaStateFromStatus(sdk.V2WorkflowRunStatusCancelled))
	require.Equal(t, gitea.StatusWarning, getGiteaStateFromStatus(sdk.V2WorkflowRunStatusSkipped))
}
