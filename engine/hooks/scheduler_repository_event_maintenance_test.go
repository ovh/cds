package hooks

import (
	"context"
	"testing"
	"time"

	"github.com/ovh/cds/sdk"
	"github.com/rockbears/log"
	"github.com/stretchr/testify/require"
)

func TestMaintenanceQueue_ManualEventIsRoutedToMaintenanceQueue(t *testing.T) {
	log.Factory = log.NewTestingWrapper(t)
	s, cancel := setupTestHookService(t)
	defer cancel()

	ctx := context.TODO()

	// Enable maintenance
	s.Maintenance = true

	// Trigger a manual workflow event while in maintenance
	hre, err := s.handleManualWorkflowEvent(ctx, sdk.HookManualWorkflowRun{
		Project:        "MYPROJECT",
		Workflow:       "my-workflow",
		VCSServer:      "github",
		Repository:     "ovh/cds",
		WorkflowRef:    "refs/heads/main",
		WorkflowCommit: "abc123",
	})
	require.NoError(t, err)
	require.NotNil(t, hre)

	// The event should have IsInMaintenance=true
	require.NotNil(t, hre.ExtractData.Manual)
	require.True(t, hre.ExtractData.Manual.IsInMaintenance)

	// Dequeue from the maintenance queue — we should get this event
	var eventKey string
	err = s.Cache.DequeueWithContext(ctx, repositoryEventMaintenanceQueue, 250*time.Millisecond, &eventKey)
	require.NoError(t, err)
	require.NotEmpty(t, eventKey)
	require.Contains(t, eventKey, hre.UUID)
}

func TestMaintenanceQueue_NonManualEventIsNotRoutedToMaintenanceQueue(t *testing.T) {
	log.Factory = log.NewTestingWrapper(t)
	s, cancel := setupTestHookService(t)
	defer cancel()

	ctx := context.TODO()

	// Enable maintenance
	s.Maintenance = true

	// Enqueue a push event (non-manual) while in maintenance
	pushEvent := &sdk.HookRepositoryEvent{
		UUID:           sdk.UUID(),
		VCSServerName:  "github",
		RepositoryName: "ovh/cds",
		Status:         sdk.HookEventStatusScheduled,
		EventName:      sdk.WorkflowHookEventNamePush,
		Created:        time.Now().UnixNano(),
		ExtractData: sdk.HookRepositoryEventExtractData{
			Ref:    "refs/heads/main",
			Commit: "abc123",
		},
	}
	require.NoError(t, s.Dao.SaveRepositoryEvent(ctx, pushEvent))
	require.NoError(t, s.Dao.EnqueueRepositoryEvent(ctx, pushEvent))

	// Try to dequeue from maintenance queue with a short-lived context — should get nothing
	ctxTimeout, cancelTimeout := context.WithTimeout(ctx, 2*time.Second)
	defer cancelTimeout()
	var eventKey string
	err := s.Cache.DequeueWithContext(ctxTimeout, repositoryEventMaintenanceQueue, 250*time.Millisecond, &eventKey)
	// Expect context deadline exceeded (nothing to dequeue)
	require.Error(t, err)
	require.Empty(t, eventKey)
}
