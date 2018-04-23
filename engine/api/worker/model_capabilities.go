package worker

import (
	"context"
	"fmt"
	"time"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/sdk/log"
)

//ModelCapabilititiesCacheLoader set all model Capabilities in the cache
func ModelCapabilititiesCacheLoader(c context.Context, delay time.Duration, DBFunc func(context.Context) *gorp.DbMap, store cache.Store) {
	tick := time.NewTicker(delay).C
	for {
		select {
		case <-c.Done():
			if c.Err() != nil {
				log.Error("Exiting worker.ModelCapabilititiesCacheLoader: %v", c.Err())
				return
			}
		case <-tick:
			dbmap := DBFunc(c)
			if dbmap != nil {
				var mayIWork string
				loaderKey := cache.Key("worker", "modelcapabilitites", "loading")
				if store.Get(loaderKey, &mayIWork) {
					store.SetWithTTL(loaderKey, "true", 60)
					wms, err := LoadWorkerModels(dbmap)
					if err != nil {
						log.Warning("ModelCapabilititiesCacheLoader> Unable to load worker models: %s", err)
						continue
					}
					for _, wm := range wms {
						modelKey := cache.Key("worker", "modelcapabilitites", fmt.Sprintf("%d", wm.ID))
						store.Set(modelKey, wm.Capabilities)
					}
					store.Delete(loaderKey)
				}
			}
		}
	}
}
