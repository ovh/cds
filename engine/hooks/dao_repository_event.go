package hooks

import (
	"context"
	"strings"
	"sync/atomic"
	"time"

	"github.com/rockbears/log"

	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/sdk"
)

func (d *dao) RepositoryEventBalance() (int64, int64) {
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
	k := strings.ToLower(cache.Key(repositoryEventRootKey, d.GetRepositoryMemberKey(e.VCSServerName, e.RepositoryName)))
	return d.store.SetAdd(k, e.UUID, e)
}

func (d *dao) RemoveRepositoryEventFromInProgressList(ctx context.Context, e sdk.HookRepositoryEvent) error {
	return d.store.SetRemove(repositoryEventInProgressKey, e.UUID, e)
}

func (d *dao) EnqueueRepositoryEvent(ctx context.Context, e *sdk.HookRepositoryEvent) error {
	// Use to identify event in progress:
	k := strings.ToLower(cache.Key(repositoryEventRootKey, d.GetRepositoryMemberKey(e.VCSServerName, e.RepositoryName), e.UUID))
	log.Debug(ctx, "enqueue event: %s", k)

	if err := d.RemoveRepositoryEventFromInProgressList(ctx, *e); err != nil {
		return err
	}
	if err := d.store.SetRemove(repositoryEventInProgressKey, e.UUID, k); err != nil {
		return err
	}
	if err := d.store.SetAdd(repositoryEventInProgressKey, e.UUID, k); err != nil {
		return err
	}
	return d.store.Enqueue(repositoryEventQueue, k)
}

func (d *dao) getRepositoryEventLockKey(vcsName, repoName, hookEventUUID string) string {
	return strings.ToLower(cache.Key(repositoryEventLockRootKey, d.GetRepositoryMemberKey(vcsName, repoName), hookEventUUID))
}

func (d *dao) LockRepositoryEvent(vcsName, repoName, hookEventUUID string) (bool, error) {
	lockKey := d.getRepositoryEventLockKey(vcsName, repoName, hookEventUUID)
	return d.store.Lock(lockKey, 30*time.Second, 200, 60)
}

func (d *dao) UnlockRepositoryEvent(vcsName, repoName, hookEventUUID string) error {
	lockKey := d.getRepositoryEventLockKey(vcsName, repoName, hookEventUUID)
	return d.store.Unlock(lockKey)
}

func (d *dao) ListInProgressRepositoryEvent(ctx context.Context) ([]string, error) {
	nbHookEventInProgress, err := d.store.SetCard(repositoryEventInProgressKey)
	if err != nil {
		return nil, sdk.WrapError(err, "unable to setCard %v", repositoryEventInProgressKey)
	}
	inProgressEvents := make([]*string, 0, nbHookEventInProgress)
	for i := 0; i < nbHookEventInProgress; i++ {
		content := ""
		inProgressEvents = append(inProgressEvents, &content)
	}
	if err := d.store.SetScan(ctx, repositoryEventInProgressKey, sdk.InterfaceSlice(inProgressEvents)...); err != nil {
		return nil, sdk.WrapError(err, "Unable to scan %s", repositoryEventInProgressKey)
	}

	eventKeys := make([]string, 0, len(inProgressEvents))
	for _, k := range inProgressEvents {
		eventKeys = append(eventKeys, *k)
	}

	return eventKeys, nil
}

func (d *dao) DeleteRepositoryEvent(ctx context.Context, vcsServer, repository, uuid string) error {
	k := strings.ToLower(cache.Key(repositoryEventRootKey, d.GetRepositoryMemberKey(vcsServer, repository)))
	if err := d.store.SetRemove(k, uuid, nil); err != nil {
		return err
	}
	return nil
}

func (d *dao) GetRepositoryEvent(ctx context.Context, vcsServer, repository, uuid string) (*sdk.HookRepositoryEvent, error) {
	k := cache.Key(repositoryEventRootKey, vcsServer+"-"+repository, uuid)
	var e sdk.HookRepositoryEvent
	found, err := d.store.Get(k, &e)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, sdk.WithStack(sdk.ErrNotFound)
	}
	return &e, nil
}

func (d *dao) DeleteAllRepositoryEvent(ctx context.Context, vcsServer, repoName string) error {
  repoEvents, err := d.ListRepositoryEvents(ctx, vcsServer, repoName)
  if err != nil {
    return err
  }
  for _, re := range repoEvents {
    if err := d.DeleteRepositoryEvent(ctx, re.VCSServerName, re.RepositoryName, re.UUID); err != nil {
      return err
    }
  }
  k := strings.ToLower(cache.Key(repositoryEventRootKey, d.GetRepositoryMemberKey(vcsServer, repoName)))
  if err := d.store.Delete(k); err != nil {
    return err
  }
  return nil
}

func (d *dao) ListRepositoryEvents(ctx context.Context, vcsServer, repository string) ([]sdk.HookRepositoryEvent, error) {
	k := cache.Key(repositoryEventRootKey, vcsServer+"-"+repository)

	nbEvents, err := d.store.SetCard(k)
	if err != nil {
		return nil, err
	}
	log.Info(ctx, "%s", k)
	events := make([]*sdk.HookRepositoryEvent, nbEvents)
	for i := 0; i < nbEvents; i++ {
		events[i] = &sdk.HookRepositoryEvent{}
	}
	if err := d.store.SetScan(ctx, k, sdk.InterfaceSlice(events)...); err != nil {
		return nil, err
	}
	finalEvents := make([]sdk.HookRepositoryEvent, 0, len(events))
	for _, e := range events {
		finalEvents = append(finalEvents, *e)
	}
	return finalEvents, nil
}
