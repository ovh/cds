package hooks

import (
	"context"
	"fmt"
	"testing"

	"github.com/rockbears/log"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient/mock_cdsclient"
)

func newTestHook(id, vcs, repo, workflow, cron, tz string) sdk.V2WorkflowHook {
	return sdk.V2WorkflowHook{
		ID:             id,
		VCSName:        vcs,
		RepositoryName: repo,
		WorkflowName:   workflow,
		ProjectKey:     "PROJ",
		Ref:            "refs/heads/main",
		Commit:         "HEAD",
		Type:           sdk.WorkflowHookTypeScheduler,
		Data: sdk.V2WorkflowHookData{
			Cron:         cron,
			CronTimeZone: tz,
		},
	}
}

// cleanRedisSchedulerKeys removes all scheduler-related keys from Redis to ensure test isolation.
func cleanRedisSchedulerKeys(t *testing.T, store cache.Store) {
	t.Helper()
	require.NoError(t, store.DeleteAll(cache.Key(scheduleDefinitionRootKey, "*")))
	require.NoError(t, store.DeleteAll(cache.Key(schedulerNextExecutionRootKey, "*")))
	require.NoError(t, store.Delete(schedulerNextExecutionRootKey))
	require.NoError(t, store.Delete(schedulerResyncLockKey))
}

// TestResyncAddMissingSchedulers verifies that schedulers present in DB but missing from Redis
// are created with their definition and next execution.
func TestResyncAddMissingSchedulers(t *testing.T) {
	log.Factory = log.NewTestingWrapper(t)
	s, cancel := setupTestHookService(t)
	defer cancel()
	cleanRedisSchedulerKeys(t, s.Dao.store)
	ctx := context.TODO()

	hook1 := newTestHook("hook-add-1", "github", "myorg/myrepo", "my-workflow", "0 */5 * * *", "UTC")
	hook2 := newTestHook("hook-add-2", "github", "myorg/myrepo", "my-workflow-2", "0 0 * * *", "Europe/Paris")

	// Mock: API returns 2 hooks, Redis is empty
	mockClient := s.Client.(*mock_cdsclient.MockInterface)
	mockClient.EXPECT().HookListAllSchedulerHooks(gomock.Any()).Return([]sdk.V2WorkflowHook{hook1, hook2}, nil)

	// Step 3 will not call HookGetWorkflowHook since hooks are missing from Redis (added, not updated)

	err := s.resyncSchedulers(ctx)
	require.NoError(t, err)

	// Verify definitions were created in Redis
	def1, err := s.Dao.GetSchedulerDefinition(ctx, hook1.VCSName, hook1.RepositoryName, hook1.WorkflowName, hook1.ID)
	require.NoError(t, err)
	require.NotNil(t, def1)
	require.Equal(t, "0 */5 * * *", def1.Data.Cron)

	def2, err := s.Dao.GetSchedulerDefinition(ctx, hook2.VCSName, hook2.RepositoryName, hook2.WorkflowName, hook2.ID)
	require.NoError(t, err)
	require.NotNil(t, def2)
	require.Equal(t, "0 0 * * *", def2.Data.Cron)

	// Verify executions were created
	exec1, err := s.Dao.GetSchedulerExecution(ctx, hook1.ID)
	require.NoError(t, err)
	require.NotNil(t, exec1)

	exec2, err := s.Dao.GetSchedulerExecution(ctx, hook2.ID)
	require.NoError(t, err)
	require.NotNil(t, exec2)

	// Cleanup
	require.NoError(t, s.Dao.RemoveScheduler(ctx, hook1.VCSName, hook1.RepositoryName, hook1.WorkflowName, hook1.ID))
	require.NoError(t, s.Dao.RemoveScheduler(ctx, hook2.VCSName, hook2.RepositoryName, hook2.WorkflowName, hook2.ID))
}

// TestResyncRemoveOrphanSchedulers verifies that schedulers present in Redis but not in DB
// are removed after double-check with the API.
func TestResyncRemoveOrphanSchedulers(t *testing.T) {
	log.Factory = log.NewTestingWrapper(t)
	s, cancel := setupTestHookService(t)
	defer cancel()
	cleanRedisSchedulerKeys(t, s.Dao.store)
	ctx := context.TODO()

	orphan := newTestHook("hook-orphan-1", "github", "myorg/myrepo", "old-workflow", "0 0 * * *", "UTC")

	// Seed Redis with an orphan scheduler
	require.NoError(t, s.Dao.CreateSchedulerDefinition(ctx, orphan))
	require.NoError(t, s.createSchedulerNextExecution(ctx, orphan))

	// Mock: API returns empty list (no schedulers in DB)
	mockClient := s.Client.(*mock_cdsclient.MockInterface)
	mockClient.EXPECT().HookListAllSchedulerHooks(gomock.Any()).Return([]sdk.V2WorkflowHook{}, nil)
	// Double-check returns not found (step 4 removal)
	mockClient.EXPECT().HookGetWorkflowHook(gomock.Any(), orphan.ID).Return(nil, fmt.Errorf("not found")).AnyTimes()

	err := s.resyncSchedulers(ctx)
	require.NoError(t, err)

	// Verify definition was removed
	def, err := s.Dao.GetSchedulerDefinition(ctx, orphan.VCSName, orphan.RepositoryName, orphan.WorkflowName, orphan.ID)
	require.NoError(t, err)
	require.Nil(t, def)

	// Verify execution data key was removed
	exec, err := s.Dao.GetSchedulerExecution(ctx, orphan.ID)
	require.NoError(t, err)
	require.Nil(t, exec)
}

// TestResyncUpdateSchedulerConfig verifies that when a scheduler's cron or timezone has changed in the DB,
// the Redis definition and execution are updated.
func TestResyncUpdateSchedulerConfig(t *testing.T) {
	log.Factory = log.NewTestingWrapper(t)
	s, cancel := setupTestHookService(t)
	defer cancel()
	cleanRedisSchedulerKeys(t, s.Dao.store)
	ctx := context.TODO()

	hookID := "hook-update-1"
	oldHook := newTestHook(hookID, "github", "myorg/myrepo", "my-workflow", "0 0 * * *", "UTC")
	newHook := newTestHook(hookID, "github", "myorg/myrepo", "my-workflow", "0 */10 * * *", "Europe/Paris")

	// Seed Redis with old config
	require.NoError(t, s.Dao.CreateSchedulerDefinition(ctx, oldHook))
	require.NoError(t, s.createSchedulerNextExecution(ctx, oldHook))

	oldExec, err := s.Dao.GetSchedulerExecution(ctx, hookID)
	require.NoError(t, err)
	require.NotNil(t, oldExec)

	// Mock: API returns new config in list
	mockClient := s.Client.(*mock_cdsclient.MockInterface)
	mockClient.EXPECT().HookListAllSchedulerHooks(gomock.Any()).Return([]sdk.V2WorkflowHook{newHook}, nil)
	// Step 3: reload fresh hook for comparison
	mockClient.EXPECT().HookGetWorkflowHook(gomock.Any(), hookID).Return(&newHook, nil)

	err = s.resyncSchedulers(ctx)
	require.NoError(t, err)

	// Verify definition was updated
	def, err := s.Dao.GetSchedulerDefinition(ctx, newHook.VCSName, newHook.RepositoryName, newHook.WorkflowName, hookID)
	require.NoError(t, err)
	require.NotNil(t, def)
	require.Equal(t, "0 */10 * * *", def.Data.Cron)
	require.Equal(t, "Europe/Paris", def.Data.CronTimeZone)

	// Verify execution was recreated (different next execution time due to new cron)
	newExec, err := s.Dao.GetSchedulerExecution(ctx, hookID)
	require.NoError(t, err)
	require.NotNil(t, newExec)

	// Cleanup
	require.NoError(t, s.Dao.RemoveScheduler(ctx, newHook.VCSName, newHook.RepositoryName, newHook.WorkflowName, hookID))
}

// TestResyncIdempotent verifies that when DB and Redis are in sync, no changes are made.
func TestResyncIdempotent(t *testing.T) {
	log.Factory = log.NewTestingWrapper(t)
	s, cancel := setupTestHookService(t)
	defer cancel()
	cleanRedisSchedulerKeys(t, s.Dao.store)
	ctx := context.TODO()

	hook := newTestHook("hook-idem-1", "github", "myorg/myrepo", "my-workflow", "0 */5 * * *", "UTC")

	// Seed Redis with the same data as in the DB
	require.NoError(t, s.Dao.CreateSchedulerDefinition(ctx, hook))
	require.NoError(t, s.createSchedulerNextExecution(ctx, hook))

	execBefore, err := s.Dao.GetSchedulerExecution(ctx, hook.ID)
	require.NoError(t, err)
	require.NotNil(t, execBefore)

	// Mock: API returns the same hook
	mockClient := s.Client.(*mock_cdsclient.MockInterface)
	mockClient.EXPECT().HookListAllSchedulerHooks(gomock.Any()).Return([]sdk.V2WorkflowHook{hook}, nil)
	// Step 3: fresh reload returns same config → no update
	mockClient.EXPECT().HookGetWorkflowHook(gomock.Any(), hook.ID).Return(&hook, nil)

	err = s.resyncSchedulers(ctx)
	require.NoError(t, err)

	// Verify definition is still there
	def, err := s.Dao.GetSchedulerDefinition(ctx, hook.VCSName, hook.RepositoryName, hook.WorkflowName, hook.ID)
	require.NoError(t, err)
	require.NotNil(t, def)
	require.Equal(t, "0 */5 * * *", def.Data.Cron)

	// Verify execution is still there and unchanged
	execAfter, err := s.Dao.GetSchedulerExecution(ctx, hook.ID)
	require.NoError(t, err)
	require.NotNil(t, execAfter)
	require.Equal(t, execBefore.NextExecutionTime, execAfter.NextExecutionTime)

	// Cleanup
	require.NoError(t, s.Dao.RemoveScheduler(ctx, hook.VCSName, hook.RepositoryName, hook.WorkflowName, hook.ID))
}

// TestResyncOrphanNotRemovedIfAPIStillHasIt verifies that a scheduler present in Redis but not
// in the initial DB snapshot is NOT removed if the API double-check confirms it exists.
func TestResyncOrphanNotRemovedIfAPIStillHasIt(t *testing.T) {
	log.Factory = log.NewTestingWrapper(t)
	s, cancel := setupTestHookService(t)
	defer cancel()
	cleanRedisSchedulerKeys(t, s.Dao.store)
	ctx := context.TODO()

	recentHook := newTestHook("hook-recent-1", "github", "myorg/myrepo", "new-workflow", "0 0 * * *", "UTC")

	// Seed Redis (simulates a hook created after the DB snapshot was taken)
	require.NoError(t, s.Dao.CreateSchedulerDefinition(ctx, recentHook))
	require.NoError(t, s.createSchedulerNextExecution(ctx, recentHook))

	// Mock: API snapshot doesn't include this hook (created after snapshot)
	mockClient := s.Client.(*mock_cdsclient.MockInterface)
	mockClient.EXPECT().HookListAllSchedulerHooks(gomock.Any()).Return([]sdk.V2WorkflowHook{}, nil)
	// Double-check: API confirms the hook still exists (called from step 4 for definition and step 5 for execution)
	mockClient.EXPECT().HookGetWorkflowHook(gomock.Any(), recentHook.ID).Return(&recentHook, nil).AnyTimes()

	err := s.resyncSchedulers(ctx)
	require.NoError(t, err)

	// Verify the scheduler was NOT removed
	def, err := s.Dao.GetSchedulerDefinition(ctx, recentHook.VCSName, recentHook.RepositoryName, recentHook.WorkflowName, recentHook.ID)
	require.NoError(t, err)
	require.NotNil(t, def)

	exec, err := s.Dao.GetSchedulerExecution(ctx, recentHook.ID)
	require.NoError(t, err)
	require.NotNil(t, exec)

	// Cleanup
	require.NoError(t, s.Dao.RemoveScheduler(ctx, recentHook.VCSName, recentHook.RepositoryName, recentHook.WorkflowName, recentHook.ID))
}

// TestResyncEnsurePendingExecutions verifies that if a scheduler definition exists in Redis
// but its execution is missing, the execution is recreated.
func TestResyncEnsurePendingExecutions(t *testing.T) {
	log.Factory = log.NewTestingWrapper(t)
	s, cancel := setupTestHookService(t)
	defer cancel()
	cleanRedisSchedulerKeys(t, s.Dao.store)
	ctx := context.TODO()

	hook := newTestHook("hook-noexec-1", "github", "myorg/myrepo", "my-workflow", "0 */5 * * *", "UTC")

	// Seed Redis with definition only, no execution
	require.NoError(t, s.Dao.CreateSchedulerDefinition(ctx, hook))

	// Mock: API returns this hook
	mockClient := s.Client.(*mock_cdsclient.MockInterface)
	mockClient.EXPECT().HookListAllSchedulerHooks(gomock.Any()).Return([]sdk.V2WorkflowHook{hook}, nil)
	// Step 3: fresh reload returns same config → no update needed
	mockClient.EXPECT().HookGetWorkflowHook(gomock.Any(), hook.ID).Return(&hook, nil)

	err := s.resyncSchedulers(ctx)
	require.NoError(t, err)

	// Verify execution was created by step 6
	exec, err := s.Dao.GetSchedulerExecution(ctx, hook.ID)
	require.NoError(t, err)
	require.NotNil(t, exec)
	require.True(t, exec.NextExecutionTime > 0)

	// Cleanup
	require.NoError(t, s.Dao.RemoveScheduler(ctx, hook.VCSName, hook.RepositoryName, hook.WorkflowName, hook.ID))
}

// TestResyncCleanOrphanExecutions verifies that executions for hooks that no longer exist
// in the DB are cleaned up.
func TestResyncCleanOrphanExecutions(t *testing.T) {
	log.Factory = log.NewTestingWrapper(t)
	s, cancel := setupTestHookService(t)
	defer cancel()
	cleanRedisSchedulerKeys(t, s.Dao.store)
	ctx := context.TODO()

	orphanHook := newTestHook("hook-orphan-exec-1", "github", "myorg/myrepo", "deleted-workflow", "0 0 * * *", "UTC")

	// Seed Redis with an execution only (definition already cleaned up by something else)
	require.NoError(t, s.createSchedulerNextExecution(ctx, orphanHook))

	exec, err := s.Dao.GetSchedulerExecution(ctx, orphanHook.ID)
	require.NoError(t, err)
	require.NotNil(t, exec)

	// Mock: API returns no hooks
	mockClient := s.Client.(*mock_cdsclient.MockInterface)
	mockClient.EXPECT().HookListAllSchedulerHooks(gomock.Any()).Return([]sdk.V2WorkflowHook{}, nil)
	// Step 5: double-check returns not found
	mockClient.EXPECT().HookGetWorkflowHook(gomock.Any(), orphanHook.ID).Return(nil, fmt.Errorf("not found"))

	err = s.resyncSchedulers(ctx)
	require.NoError(t, err)

	// Verify orphan execution was removed
	exec, err = s.Dao.GetSchedulerExecution(ctx, orphanHook.ID)
	require.NoError(t, err)
	require.Nil(t, exec)
}
