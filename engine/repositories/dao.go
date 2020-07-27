package repositories

import (
	"context"
	"fmt"
	"time"

	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

var (
	rootKey       = cache.Key("repositories", "operations")
	processorKey  = cache.Key("repositories", "processor")
	locksKey      = cache.Key("repositories", "locks")
	lastAccessKey = cache.Key("repositories", "access")
)

type dao struct {
	store cache.Store
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

	_, err2 := d.store.Lock(cache.Key(lastAccessKey, uuid), 3*24*time.Hour, -1, -1)
	if err2 != nil {
		return sdk.WrapError(err2, "error on lock uuid: %s", uuid)
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

func (d *dao) unlock(ctx context.Context, uuid string, retention time.Duration) error {
	if err := d.store.Unlock(cache.Key(locksKey, uuid)); err != nil {
		log.Error(ctx, "error on unlock uuid %s: %v", uuid, err)
	}
	if _, err := d.store.Lock(cache.Key(lastAccessKey, uuid), retention, -1, -1); err != nil {
		return sdk.WrapError(err, "error on cache.lock uuid:%s", uuid)
	}
	return nil
}

func (d *dao) isExpired(ctx context.Context, uuid string) bool {
	k := cache.Key(lastAccessKey, uuid)
	var b bool
	find, err := d.store.Get(k, &b)
	if err != nil {
		log.Error(ctx, "cannot get from cache %s: %v", k, err)
	}
	if find {
		return false
	}
	return true
}
