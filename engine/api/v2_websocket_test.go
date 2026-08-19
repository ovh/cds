package api

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ovh/cds/sdk"
)

func TestWebsocketV2ComputeEventKeys(t *testing.T) {
	api := &API{}

	runKey := sdk.WebsocketV2Filter{
		Type:          sdk.WebsocketV2FilterTypeProjectRun,
		ProjectKey:    "KEY",
		WorkflowRunID: "run-1",
	}.Key()

	t.Run("run event matches both the run list and the run itself", func(t *testing.T) {
		keys := api.websocketV2ComputeEventKeys(sdk.FullEventV2{
			Type:          sdk.EventRunBuilding,
			ProjectKey:    "KEY",
			WorkflowRunID: "run-1",
		})
		require.Contains(t, keys, runKey)
		require.Contains(t, keys, sdk.WebsocketV2Filter{
			Type:       sdk.WebsocketV2FilterTypeProjectRuns,
			ProjectKey: "KEY",
		}.Key())
	})

	t.Run("job event matches the queue and the run itself", func(t *testing.T) {
		keys := api.websocketV2ComputeEventKeys(sdk.FullEventV2{
			Type:          sdk.EventRunJobBuilding,
			ProjectKey:    "KEY",
			WorkflowRunID: "run-1",
			RunJobID:      "job-1",
		})
		require.Contains(t, keys, runKey)
		require.Contains(t, keys, sdk.WebsocketV2Filter{Type: sdk.WebsocketV2FilterTypeQueue}.Key())
	})

	t.Run("step and result events only match the run itself", func(t *testing.T) {
		for _, eventType := range []sdk.EventType{sdk.EventRunJobStepUpdated, sdk.EventRunJobRunResultAdded, sdk.EventRunUpdated, sdk.EventRunDeleted} {
			keys := api.websocketV2ComputeEventKeys(sdk.FullEventV2{
				Type:          eventType,
				ProjectKey:    "KEY",
				WorkflowRunID: "run-1",
			})
			require.Equal(t, []string{runKey}, keys, "unexpected keys for event %s", eventType)
		}
	})

	t.Run("event without run id does not match a run filter", func(t *testing.T) {
		keys := api.websocketV2ComputeEventKeys(sdk.FullEventV2{
			Type:       sdk.EventRepositoryCreated,
			ProjectKey: "KEY",
		})
		require.NotContains(t, keys, runKey)
	})
}

func TestWebsocketV2FiltersHasOneKey(t *testing.T) {
	projectRuns := sdk.WebsocketV2Filter{Type: sdk.WebsocketV2FilterTypeProjectRuns, ProjectKey: "KEY"}
	projectRun := sdk.WebsocketV2Filter{Type: sdk.WebsocketV2FilterTypeProjectRun, ProjectKey: "KEY", WorkflowRunID: "run-1"}
	queue := sdk.WebsocketV2Filter{Type: sdk.WebsocketV2FilterTypeQueue}

	t.Run("a run filter allows the event without post check", func(t *testing.T) {
		found, needPostCheck := webSocketV2Filters{projectRuns, projectRun}.HasOneKey(projectRuns.Key(), projectRun.Key())
		require.True(t, found)
		require.False(t, needPostCheck, "the run filter matches, the search of the run list must not be replayed")
	})

	t.Run("the run list filter alone still needs a post check", func(t *testing.T) {
		found, needPostCheck := webSocketV2Filters{projectRuns}.HasOneKey(projectRuns.Key(), projectRun.Key())
		require.True(t, found)
		require.True(t, needPostCheck)
	})

	t.Run("the queue filter alone still needs a post check", func(t *testing.T) {
		found, needPostCheck := webSocketV2Filters{queue}.HasOneKey(queue.Key(), projectRun.Key())
		require.True(t, found)
		require.True(t, needPostCheck)
	})

	t.Run("no filter matches", func(t *testing.T) {
		found, _ := webSocketV2Filters{queue}.HasOneKey(projectRun.Key())
		require.False(t, found)
	})
}
