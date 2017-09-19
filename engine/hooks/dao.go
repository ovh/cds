package hooks

import (
	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/sdk"
)

type dao struct {
	store cache.Store
}

func (d *dao) FindAllLongRunningTasks() ([]LongRunningTask, error) {
	nbTasks := d.store.SetCard(longRunningRootKey)
	tasks := make([]*LongRunningTask, nbTasks, nbTasks)
	for i := 0; i < nbTasks; i++ {
		tasks[i] = &LongRunningTask{}
	}
	if err := d.store.SetScan(longRunningRootKey, interfaceSlice(tasks)...); err != nil {
		return nil, sdk.WrapError(err, "hooks>FindAllLongRunningTasks> Unable to scan %s", longRunningRootKey)
	}

	alltasks := make([]LongRunningTask, nbTasks)
	for i := 0; i < nbTasks; i++ {
		alltasks[i] = *tasks[i]
	}

	return alltasks, nil
}

func (d *dao) FindLongRunningTask(uuid string) *LongRunningTask {
	key := cache.Key("hooks", "tasks", "long_running", uuid)
	t := &LongRunningTask{}
	if d.store.Get(key, t) {
		return t
	}
	return nil
}

func (d *dao) SaveLongRunningTask(r *LongRunningTask) {
	d.store.SetAdd(longRunningRootKey, r.UUID, r)
}
