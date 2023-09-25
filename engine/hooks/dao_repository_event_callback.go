package hooks

import (
	"context"
	"sync/atomic"

	"github.com/ovh/cds/sdk"
)

func (d *dao) RepositoryEventCallbackBalance() (int64, int64) {
	return d.enqueuedRepositoryEventCallbacks, d.dequeuedRepositoryEventCallbacks
}

func (d *dao) enqueuedRepositoryEventCallbackIncr() {
	atomic.AddInt64(&d.enqueuedRepositoryEventCallbacks, 1)
}

func (d *dao) dequeuedRepositoryEventCallbackIncr() {
	atomic.AddInt64(&d.dequeuedRepositoryEventCallbacks, 1)
}

func (d *dao) RepositoryEventCallbackQueueLen() (int, error) {
	return d.store.QueueLen(repositoryEventCallbackQueue)
}

func (d *dao) EnqueueRepositoryEventCallback(_ context.Context, e sdk.HookAnalysisCallback) error {
	return d.store.Enqueue(repositoryEventCallbackQueue, e)
}
