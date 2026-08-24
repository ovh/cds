package repositories

import (
	"context"
	"fmt"
	"time"

	"github.com/rockbears/log"

	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/sdk"
)

var (
	rootKey           = cache.Key("repositories", "operations")
	processorKey      = cache.Key("repositories", "processor")
	locksKey          = cache.Key("repositories", "locks")
	lastAccessRootKey = cache.Key("repositories", "lastAccess")
)

type dao struct {
	store cache.Store
	// hostname scopes lastAccess keys to this instance: clones live on a local
	// filesystem, so retention must reflect each instance's own usage.
	hostname string
}

func (d *dao) lastAccessKey(repoUUID string) string {
	return cache.Key(lastAccessRootKey, d.hostname, repoUUID)
}

func (d *dao) saveLastAccess(repoUUID string, expiration time.Time, ttlSeconds int) {
	d.store.SetWithTTL(d.lastAccessKey(repoUUID), expiration, ttlSeconds)
}

func (d *dao) saveOperation(o *sdk.Operation) error {
	return d.store.SetAdd(rootKey, o.UUID, o)
}

func (d *dao) pushOperation(o *sdk.Operation) error {
	return d.store.Enqueue(processorKey, o.UUID)
}

func (d *dao) deleteOperation(o *sdk.Operation) error {
	return d.store.SetRemove(rootKey, o.UUID, o)
}

func (d *dao) loadOperation(ctx context.Context, uuid string) *sdk.Operation {
	key := cache.Key(rootKey, uuid)
	o := new(sdk.Operation)
	find, err := d.store.Get(key, o)
	if err != nil {
		log.Error(ctx, "cannot get from cache %s: %v", key, err)
	}
	if find {
		return o
	}
	return nil
}

func (d *dao) loadAllOperations(ctx context.Context) ([]*sdk.Operation, error) {
	n, err := d.store.SetCard(rootKey)
	if err != nil {
		return nil, sdk.WrapError(err, "unable to setCard %v", rootKey)
	}
	opes := make([]*sdk.Operation, n)
	for i := 0; i < n; i++ {
		opes[i] = &sdk.Operation{}
	}
	if err := d.store.SetScan(ctx, rootKey, sdk.InterfaceSlice(opes)...); err != nil {
		return nil, err
	}
	return opes, nil
}

var errLockUnavailable = fmt.Errorf("errLockUnavailable")

func (d *dao) lock(uuid string) error {
	ok, err := d.store.Lock(cache.Key(locksKey, uuid), 10*time.Minute, -1, -1)
	if err != nil || !ok {
		return errLockUnavailable
	}
	return nil
}

func (d *dao) deleteLock(ctx context.Context, uuid string) error {
	k := cache.Key(locksKey, uuid)
	if err := d.store.Delete(k); err != nil {
		log.Error(ctx, "unable to cache delete %s: %v", k, err)
	}
	return nil
}

func (d *dao) unlock(ctx context.Context, uuid string) error {
	if err := d.store.Unlock(cache.Key(locksKey, uuid)); err != nil {
		log.Error(ctx, "error on unlock uuid %s: %v", uuid, err)
	}
	return nil
}

func (d *dao) isExpired(ctx context.Context, uuid string) (time.Time, bool) {
	k := d.lastAccessKey(uuid)
	var v time.Time
	find, err := d.store.Get(k, &v)
	if err != nil {
		log.Error(ctx, "cannot get from cache %s: %v", k, err)
	}
	if find {
		return v, false
	}
	return v, true
}
