package repositories

import "github.com/ovh/cds/engine/api/cache"

var (
	rootKey      = cache.Key("repositories", "operations")
	processorKey = cache.Key("repositories", "processor")
	reposKey     = cache.Key("repositories", "repos")
)

type dao struct {
	store cache.Store
}

func (d *dao) saveOperation(o *Operation) error {
	d.store.SetAdd(rootKey, o.UUID, o)
	return nil
}

func (d *dao) pushOperation(o *Operation) error {
	d.store.Enqueue(processorKey, o.UUID)
	return nil
}

func (d *dao) loadOperation(uuid string) *Operation {
	key := cache.Key(rootKey, uuid)
	o := new(Operation)
	if d.store.Get(key, o) {
		return o
	}
	return nil
}
