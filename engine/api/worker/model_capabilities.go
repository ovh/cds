package worker

import (
	"context"
	"fmt"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/sdk/log"
)

//ModelCapabilititiesCacheLoader set all model Capabilities in the cache
func ModelCapabilititiesCacheLoader(c context.Context, DBFunc func() *gorp.DbMap, store cache.Store) {
	var mayIWork string
	dbmap := DBFunc()
	if dbmap == nil {
		return
	}

	loaderKey := cache.Key("worker", "modelcapabilitites", "loading")
	if store.Get(loaderKey, &mayIWork) {
		store.SetWithTTL(loaderKey, "true", 60)
		wms, err := LoadWorkerModels(dbmap)
		if err != nil {
			log.Warning("ModelCapabilititiesCacheLoader> Unable to load worker models: %s", err)
		} else {
			for _, wm := range wms {
				modelKey := cache.Key("worker", "modelcapabilitites", fmt.Sprintf("%d", wm.ID))
				store.Set(modelKey, wm.RegisteredCapabilities)
			}
			store.Delete(loaderKey)
		}
	}
}
