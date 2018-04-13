package repositories

import (
	"fmt"
	"time"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/sdk"
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
	d.store.SetAdd(rootKey, o.UUID, o)
	return nil
}

func (d *dao) pushOperation(o *sdk.Operation) error {
	d.store.Enqueue(processorKey, o.UUID)
	return nil
}

func (d *dao) deleteOperation(o *sdk.Operation) error {
	d.store.SetRemove(rootKey, o.UUID, o)
	return nil
}

func (d *dao) loadOperation(uuid string) *sdk.Operation {
	key := cache.Key(rootKey, uuid)
	o := new(sdk.Operation)
	if d.store.Get(key, o) {
		return o
	}
	return nil
}

func (d *dao) loadAllOperations() ([]*sdk.Operation, error) {
	n := d.store.SetCard(rootKey)
	opes := make([]*sdk.Operation, n)
	for i := 0; i < n; i++ {
		opes[i] = &sdk.Operation{}
	}
	if err := d.store.SetScan(rootKey, sdk.InterfaceSlice(opes)...); err != nil {
		return nil, err
	}
	return opes, nil
}

var errLockUnavailable = fmt.Errorf("errLockUnavailable")

func (d *dao) lock(uuid string) error {
	if d.store.Lock(cache.Key(locksKey, uuid), 10*time.Minute) {
		d.store.Lock(cache.Key(lastAccessKey, uuid), 3*24*time.Hour)
		return nil
	}
	return errLockUnavailable
}

func (d *dao) deleteLock(uuid string) {
	d.store.Delete(cache.Key(locksKey, uuid))
}

func (d *dao) unlock(uuid string, retention time.Duration) error {
	d.store.Unlock(cache.Key(locksKey, uuid))
	d.store.Lock(cache.Key(lastAccessKey, uuid), retention)
	return nil
}

func (d *dao) isExpired(uuid string) bool {
	k := cache.Key(lastAccessKey, uuid)
	var b bool
	if d.store.Get(k, &b) {
		return false
	}
	return true
}
