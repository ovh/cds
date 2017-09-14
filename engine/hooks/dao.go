package hooks

import "github.com/ovh/cds/engine/api/cache"

type dao struct {
	store cache.Store
}

func (d *dao) FindLongRunningTask(uuid string) *LongRunningTask {
	return nil
}
