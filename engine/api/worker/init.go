package worker

import (
	"context"
	"time"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/cache"
)

//Initialize init the package
func Initialize(c context.Context, DBFunc func() *gorp.DbMap, store cache.Store) error {
	go CheckHeartbeat(c, DBFunc)
	go ModelCapabilititiesCacheLoader(c, 10*time.Second, DBFunc, store)
	go insertFirstPatterns(DBFunc())
	return nil
}
