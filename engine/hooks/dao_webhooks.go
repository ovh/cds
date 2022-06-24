package hooks

import (
	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/sdk"
)

var (
	EntitiesHookRootKey = cache.Key("hooks", "entities")
)

func (d *dao) SaveRepoWebHook(t *sdk.Task) error {
	entitiesHookKey := cache.Key(EntitiesHookRootKey,
		t.Configuration[sdk.HookConfigVCSType].Value,
		t.Configuration[sdk.HookConfigVCSServer].Value,
		t.Configuration[sdk.HookConfigRepoFullName].Value,
		t.Configuration[sdk.HookConfigTypeProject].Value)
	// Need this to be able to retrieve a task when comming from /v2/webhook/repository/{vcsType}, route without uuid
	if err := d.store.SetWithTTL(entitiesHookKey, t.UUID, 0); err != nil {
		return err
	}
	if err := d.SaveTask(t); err != nil {
		_ = d.store.Delete(entitiesHookKey) // nolint
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
