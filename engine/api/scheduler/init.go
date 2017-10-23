package scheduler

import (
	"context"
	"math/rand"
	"time"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/sdk/log"
)

//Initialize starts the 3 goroutines for pipeline schedulers
func Initialize(c context.Context, store cache.Store, nbExecToKeep int, DBFunc func() *gorp.DbMap) {
	rand.Seed(time.Now().Unix())
	tickCleaner := time.NewTicker(10 * time.Minute)
	tickScheduler := time.NewTicker(10 * time.Second)
	tickExecuter := time.NewTicker(10 * time.Second)

	for {
		time.Sleep(time.Duration(rand.Intn(500)) * time.Millisecond)

		select {
		case <-c.Done():
			if c.Err() != nil {
				log.Error("Exiting scheduler.Cleaner: %v", c.Err())
				return
			}
		case <-tickCleaner.C:
			if _, err := CleanerRun(DBFunc(), nbExecToKeep); err != nil {
				log.Warning("scheduler.Cleaner> Error : %s", err)
			}
		case <-tickExecuter.C:
			_, err := ExecuterRun(DBFunc, store)
			if err != nil {
				log.Error("scheduler.Executer> %s", err)
			}
		case <-tickScheduler.C:
			_, status, err := Run(DBFunc())
			if err != nil {
				log.Error("%s: %s", status, err)
			}
			schedulerStatus = status
		}
	}
}
