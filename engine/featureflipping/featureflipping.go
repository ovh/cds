package featureflipping

import (
	"context"
	"time"

	"github.com/go-gorp/gorp"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
	"github.com/ovh/cds/sdk/luascript"
	gocache "github.com/patrickmn/go-cache"

	"github.com/ovh/cds/engine/gorpmapper"
)

var (
	cacheFeature = gocache.New(time.Minute, time.Minute)
)

func Exists(ctx context.Context, m *gorpmapper.Mapper, db gorp.SqlExecutor, name string) bool {
	f, _ := LoadByName(ctx, m, db, name)
	return f.ID != 0
}

func IsEnabled(ctx context.Context, m *gorpmapper.Mapper, db gorp.SqlExecutor, name string, vars map[string]string) bool {
	cachedFeatureI, has := cacheFeature.Get(name)
	if !has {
		f, err := LoadByName(ctx, m, db, name)
		if err != nil {
			log.Info(ctx, "featureflipping.IsEnabled> error: unable to load Feature '%s' from database: %v", name, err)
			return false
		}
		cacheFeature.SetDefault(name, f)
		cachedFeatureI = f
	} else {
		log.Debug("featureflipping.IsEnabled> feature_flipping '%s' loaded from cache", name)
	}

	cachedFeature := cachedFeatureI.(sdk.Feature)

	luaRule, err := luascript.NewCheck()
	if err != nil {
		log.Error(ctx, "featureflipping.IsEnabled> error: unable to create new lua check: %v", err)
		return false
	}
	luaRule.SetVariables(vars)
	if err := luaRule.Perform(cachedFeature.Rule); err != nil {
		log.Error(ctx, "featureflipping.IsEnabled> error: unable to perform lua check '%s': %v", name, err)
		return false
	}

	return luaRule.Result
}
