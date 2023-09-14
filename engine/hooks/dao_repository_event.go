package hooks

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/sdk"
)

func (d *dao) RepsitoryEventBalance() (int64, int64) {
	return d.enqueuedRepositoryEvents, d.dequeuedRepositoryEvents
}

func (d *dao) enqueuedRepositoryEventIncr() {
	atomic.AddInt64(&d.enqueuedRepositoryEvents, 1)
}

func (d *dao) dequeuedRepositoryEventIncr() {
	atomic.AddInt64(&d.dequeuedRepositoryEvents, 1)
}

func (d *dao) RepositoryEventQueueLen() (int, error) {
	return d.store.QueueLen(repositoryEventQueue)
}

func (d *dao) SaveRepositoryEvent(_ context.Context, e *sdk.HookRepositoryEvent) error {
	e.LastUpdate = time.Now().UnixMilli()
	k := cache.Key(repositoryEventRootKey, d.GetRepositoryMemberKey(e.VCSServerType, e.VCSServerName, e.RepositoryName))
	return d.store.SetAdd(k, e.UUID, e)
}

func (d *dao) RemoveRepositoryEventFromInProgressList(_ context.Context, e sdk.HookRepositoryEvent) error {
	k := cache.Key(repositoryEventRootKey, d.GetRepositoryMemberKey(e.VCSServerType, e.VCSServerName, e.RepositoryName), e.UUID)
	return d.store.SetRemove(cache.Key(repositoryEventInProgressKey), k, e)
}

func (d *dao) EnqueueRepositoryEvent(_ context.Context, e *sdk.HookRepositoryEvent) error {
	// Use to identify event in progress:
	k := cache.Key(repositoryEventRootKey, d.GetRepositoryMemberKey(e.VCSServerType, e.VCSServerName, e.RepositoryName), e.UUID)
	if err := d.store.SetRemove(repositoryEventInProgressKey, fmt.Sprintf("%d", e.Timestamp), k); err != nil {
		return err
	}
	if err := d.store.SetAdd(repositoryEventInProgressKey, fmt.Sprintf("%d", e.Timestamp), k); err != nil {
		return err
	}
	return d.store.Enqueue(repositoryEventQueue, k)
}

func (d *dao) getRepositoryEventLockKey(vcsType, vcsName, repoName, hookEventUUID string) string {
	return cache.Key(repositoryEventLockRootKey, d.GetRepositoryMemberKey(vcsType, vcsName, repoName), hookEventUUID)
}

func (d *dao) LockRepositoryEvent(vcsType, vcsName, repoName, hookEventUUID string) (bool, error) {
	lockKey := d.getRepositoryEventLockKey(vcsType, vcsName, repoName, hookEventUUID)
	return d.store.Lock(lockKey, 30*time.Second, 200, 60)
}

func (d *dao) UnlockRepositoryEvent(vcsType, vcsName, repoName, hookEventUUID string) error {
	lockKey := d.getRepositoryEventLockKey(vcsType, vcsName, repoName, hookEventUUID)
	return d.store.Unlock(lockKey)
}

func (d *dao) ListInProgressRepositoryEvent(ctx context.Context) ([]string, error) {
	nbHookEventInProgress, err := d.store.SetCard(repositoryEventInProgressKey)
	if err != nil {
		return nil, sdk.WrapError(err, "unable to setCard %v", repositoryEventInProgressKey)
	}
	inProgressEvents := make([]string, nbHookEventInProgress, nbHookEventInProgress)
	for i := 0; i < nbHookEventInProgress; i++ {
		inProgressEvents[i] = ""
	}
	if err := d.store.SetScan(ctx, repositoryEventInProgressKey, sdk.InterfaceSlice(inProgressEvents)...); err != nil {
		return nil, sdk.WrapError(err, "Unable to scan %s", rootKey)
	}

	return inProgressEvents, nil
}
