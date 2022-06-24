package hooks

import (
	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/sdk"
)

var (
	EntitiesHookRootKey = cache.Key("hooks", "entities")
)

func (d *dao) SaveRepoWebHook(hookKey string, t *sdk.Task) error {
	// Need this to be able to retrieve a task when comming from /v2/webhook/repository/{vcsType}, route without uuid
	if err := d.store.SetWithTTL(hookKey, t.UUID, 0); err != nil {
		return err
	}
	if err := d.SaveTask(t); err != nil {
		_ = d.store.Delete(hookKey) // nolint
		return err
	}
	return nil
}

func (d *dao) GetAllEntitiesHookByKey(hookKey string) ([]string, error) {
	var uuids []string
	if _, err := d.store.Get(hookKey, &uuids); err != nil {
		return nil, err
	}
	return uuids, nil
}
