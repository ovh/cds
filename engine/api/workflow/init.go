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
func Initialize(ctx context.Context, DBFunc func() *gorp.DbMap, store cache.Store, uiURL, confDefaultOS, confDefaultArch string) {
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
		case <-ctx.Done():
			if ctx.Err() != nil {
				log.Error(ctx, "Exiting workflow ticker: %v", ctx.Err())
				return
			}
		case <-tickHeart.C:
			if err := manageDeadJob(ctx, DBFunc, store); err != nil {
				log.Warning(ctx, "workflow.manageDeadJob> Error on restartDeadJob : %v", err)
			}
		case <-tickStop.C:
			if err := stopRunsBlocked(ctx, db); err != nil {
				log.Warning(ctx, "workflow.stopRunsBlocked> Error on stopRunsBlocked : %v", err)
			}
		}
	}
}
