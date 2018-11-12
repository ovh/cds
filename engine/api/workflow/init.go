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
func Initialize(c context.Context, DBFunc func() *gorp.DbMap, store cache.Store, uiURL, confDefaultOS, confDefaultArch string) {
	baseUIURL = uiURL
	defaultOS = confDefaultOS
	defaultArch = confDefaultArch
	tickStop := time.NewTicker(30 * time.Minute)
	tickHeart := time.NewTicker(10 * time.Second)
	defer tickHeart.Stop()
	defer tickStop.Stop()
	db := DBFunc()

	for {
		select {
		case <-c.Done():
			if c.Err() != nil {
				log.Error("Exiting workflow ticker: %v", c.Err())
				return
			}
		case <-tickHeart.C:
			if err := restartDeadJob(c, DBFunc, store); err != nil {
				log.Warning("workflow.restartDeadJob> Error on restartDeadJob : %v", err)
			}
		case <-tickStop.C:
			if err := stopRunsBlocked(db); err != nil {
				log.Warning("workflow.stopRunsBlocked> Error on stopRunsBlocked : %v", err)
			}
		}
	}
}
