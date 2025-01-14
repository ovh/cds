package hooks

import (
	"context"
	"testing"

	"github.com/ovh/cds/sdk"
	"github.com/rockbears/log"
	"github.com/stretchr/testify/require"
)

func TestTriggerWorkflow(t *testing.T) {
	log.Factory = log.NewTestingWrapper(t)
	s, cancel := setupTestHookService(t)
	defer cancel()

	ctx := context.TODO()
	hre := sdk.HookRepositoryEvent{
		Initiator: &sdk.V2WorkflowRunInitiator{
			UserID: "1234567890",
		},
		ExtractData: sdk.HookRepositoryEventExtractData{
			Paths: []string{"src/main/main.test", "src/resources/readme.md"},
		},
		Body: []byte(`{"foo": "bar"}`),
		WorkflowHooks: []sdk.HookRepositoryEventWorkflow{
			{
				Status: sdk.HookEventWorkflowStatusScheduled,
				PathFilters: []string{
					"src/main/**/*.java",
				},
			},
		},
	}

	require.NoError(t, s.triggerWorkflows(ctx, &hre))
	require.Equal(t, sdk.HookEventWorkflowStatusSkipped, hre.WorkflowHooks[0].Status)
}
