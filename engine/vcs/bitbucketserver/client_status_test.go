package bitbucketserver

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/sdk"
)

func TestGetBitbucketStateFromStatus(t *testing.T) {
	require.Equal(t, inProgress, getBitbucketStateFromStatus(string(sdk.V2WorkflowRunStatusCrafting)))
	require.Equal(t, inProgress, getBitbucketStateFromStatus(string(sdk.V2WorkflowRunStatusBlocked)))
	require.Equal(t, inProgress, getBitbucketStateFromStatus(string(sdk.V2WorkflowRunStatusBuilding)))
	require.Equal(t, successful, getBitbucketStateFromStatus(string(sdk.V2WorkflowRunStatusSuccess)))
	require.Equal(t, failed, getBitbucketStateFromStatus(string(sdk.V2WorkflowRunStatusFail)))
	require.Equal(t, failed, getBitbucketStateFromStatus(string(sdk.V2WorkflowRunStatusStopped)))
	require.Equal(t, cancelled, getBitbucketStateFromStatus(string(sdk.V2WorkflowRunStatusCancelled)))
	require.Equal(t, unknown, getBitbucketStateFromStatus(string(sdk.V2WorkflowRunStatusSkipped)))
}

func TestListStatuses(t *testing.T) {
	client := getAuthorizedClient(t)
	statuses, err := client.ListStatuses(context.Background(), "CDS/tests", "0b6d50472e9b2c03d72a422ea11bf3faa570d0bd")
	test.NoError(t, err)
	t.Logf("%+v", statuses)
}
