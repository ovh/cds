package action

import (
	"context"
	"fmt"
	"time"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

//RequirementsCacheLoader set all action requirement in the cache
func RequirementsCacheLoader(c context.Context, delay time.Duration, DBFunc func() *gorp.DbMap, store cache.Store) {
	tick := time.NewTicker(delay).C

	for {
		select {
		case <-c.Done():
			if c.Err() != nil {
				log.Error("Exiting RequirementsCacheLoader: %v", c.Err())
				return
			}
		case <-tick:
			db := DBFunc()
			if db != nil {
				var mayIWork string
				loaderKey := cache.Key("action", "requirements", "loading")
				if store.Get(loaderKey, &mayIWork) {
					store.SetWithTTL(loaderKey, "true", 60)
					actions, err := LoadActions(db)
					if err != nil {
						log.Warning("RequirementsCacheLoader> Unable to load worker models: %s", err)
						continue
					}
					for _, a := range actions {
						k := cache.Key("action", "requirements", fmt.Sprintf("%d", a.ID))
						store.Set(k, a.Requirements)
					}
					store.Delete(loaderKey)
				}
			}
		}
	}
}

//GetRequirements load action capabilities from cache
func GetRequirements(db gorp.SqlExecutor, store cache.Store, id int64) ([]sdk.Requirement, error) {
	k := cache.Key("action", "requirements", fmt.Sprintf("%d", id))
	req := []sdk.Requirement{}
	//if we didn't got any data, try to load from DB
	if !store.Get(k, &req) {
		var err error
		req, err = LoadActionRequirements(db, id)
		if err != nil {
			return nil, fmt.Errorf("GetRequirements> cannot LoadActionRequirements: %s", err)
		}
		store.Set(k, req)
	}
	return req, nil
}
