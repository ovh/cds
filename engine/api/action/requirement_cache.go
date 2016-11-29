package action

import (
	"fmt"
	"time"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/database"
	"github.com/ovh/cds/engine/log"
	"github.com/ovh/cds/sdk"
)

//RequirementsCacheLoader set all action requirement in the cache
func RequirementsCacheLoader(delay time.Duration) {
	for {
		time.Sleep(delay * time.Second)
		db := database.DB()
		if db != nil {
			var mayIWork string
			loaderKey := cache.Key("action", "requirements", "loading")
			cache.Get(loaderKey, &mayIWork)
			if mayIWork == "" {
				cache.SetWithTTL(loaderKey, "true", 60)
				actions, err := LoadActions(db)
				if err != nil {
					log.Warning("ModelCapabilititiesCacheLoader> Unable to load worker models: %s", err)
					continue
				}
				for _, a := range actions {
					k := cache.Key("action", "requirements", fmt.Sprintf("%d", a.ID))
					cache.Set(k, a.Requirements)
				}
				cache.Delete(loaderKey)
			}
		}
	}
}

//GetRequirements load action capabilities from cache
func GetRequirements(db database.Querier, id int64) ([]sdk.Requirement, error) {
	k := cache.Key("action", "requirements", fmt.Sprintf("%d", id))
	req := []sdk.Requirement{}
	//if we didn't got any data, try to load from DB
	if !cache.Get(k, &req) {
		var err error
		req, err = LoadActionRequirements(db, id)
		if err != nil {
			return nil, fmt.Errorf("Requirements> cannot LoadActionRequirements: %s\n", err)
		}
		cache.Set(k, req)
	}
	return req, nil
}
