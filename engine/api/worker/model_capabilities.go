package worker

import (
	"context"
	"fmt"
	"time"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

//ModelCapabilititiesCacheLoader set all model Capabilities in the cache
func ModelCapabilititiesCacheLoader(c context.Context, delay time.Duration, DBFunc func() *gorp.DbMap) {
	tick := time.NewTicker(delay).C
	for {
		select {
		case <-c.Done():
			if c.Err() != nil {
				log.Error("Exiting worker.ModelCapabilititiesCacheLoader: %v", c.Err())
				return
			}
		case <-tick:
			dbmap := DBFunc()
			if dbmap != nil {
				var mayIWork string
				loaderKey := cache.Key("worker", "modelcapabilitites", "loading")
				if cache.Get(loaderKey, &mayIWork) {
					cache.SetWithTTL(loaderKey, "true", 60)
					wms, err := LoadWorkerModels(dbmap)
					if err != nil {
						log.Warning("ModelCapabilititiesCacheLoader> Unable to load worker models: %s", err)
						continue
					}
					for _, wm := range wms {
						modelKey := cache.Key("worker", "modelcapabilitites", fmt.Sprintf("%d", wm.ID))
						cache.Set(modelKey, wm.Capabilities)
					}
					cache.Delete(loaderKey)
				}
			}
		}
	}
}

//GetModelCapabilities load model capabilities from cache
func GetModelCapabilities(db gorp.SqlExecutor, modelID int64) ([]sdk.Requirement, error) {
	modelKey := cache.Key("worker", "modelcapabilitites", fmt.Sprintf("%d", modelID))
	req := []sdk.Requirement{}
	//if we didn't got any data, try to load from DB
	if !cache.Get(modelKey, &req) {
		var err error
		req, err = LoadWorkerModelCapabilities(db, modelID)
		if err != nil {
			return nil, fmt.Errorf("GetModelCapabilities> cannot loadWorkerModelCapabilities: %s", err)
		}
		cache.Set(modelKey, req)
	}
	return req, nil
}
