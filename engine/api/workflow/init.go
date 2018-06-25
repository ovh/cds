package workflow

import (
	"context"
	"time"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/sdk/log"
)

var baseUIURL, defaultOS, defaultArch string

//Initialize starts goroutines for workflows
func Initialize(c context.Context, store cache.Store, uiURL, confDefaultOS, confDefaultArch string, DBFunc func() *gorp.DbMap) {
	baseUIURL = uiURL
	defaultOS = confDefaultOS
	defaultArch = confDefaultArch
	tickPurge := time.NewTicker(30 * time.Minute)
	defer tickPurge.Stop()

	for {
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
