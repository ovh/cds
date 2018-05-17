package workflow

import (
	"context"
	"math/rand"
	"time"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/sdk/log"
)

var baseUIURL string

//Initialize starts goroutines for workflows
func Initialize(c context.Context, store cache.Store, uiURL string, DBFunc func() *gorp.DbMap) {
	baseUIURL = uiURL
	rand.Seed(time.Now().Unix())
	tickPurge := time.NewTicker(30 * time.Minute)

	for {
		time.Sleep(time.Duration(rand.Intn(500)) * time.Millisecond)

		select {
		case <-c.Done():
			if c.Err() != nil {
				log.Error("Exiting scheduler.Cleaner: %v", c.Err())
				return
			}
		case <-tickPurge.C:
			log.Debug("PurgeRun> Deleting all workflow run marked to delete...")
			if err := deleteWorkflowRunsHistory(DBFunc()); err != nil {
				log.Warning("scheduler.Purge> Error : %s", err)
			}
		}
	}
}
