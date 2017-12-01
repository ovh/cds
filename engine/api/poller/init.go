package poller

import (
	"context"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/cache"
)

//Initialize starts the 3 goroutines for pipeline schedulers
func Initialize(c context.Context, store cache.Store, nbExecToKeep int, DBFunc func() *gorp.DbMap) {
	go Cleaner(c, DBFunc, nbExecToKeep)
	go Executer(c, DBFunc, store)
	go Scheduler(c, DBFunc)
}
