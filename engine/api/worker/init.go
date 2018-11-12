package worker

import (
	"context"
	"time"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

//Initialize init the package
func Initialize(c context.Context, DBFunc func() *gorp.DbMap, store cache.Store) error {
	db := DBFunc()
	tickHeart := time.NewTicker(10 * time.Second)

	sdk.GoRoutine(c, "insertFirstPatterns", func(ctx context.Context) {
		insertFirstPatterns(db)
	})

	for {
		select {
		case <-c.Done():
			if c.Err() != nil {
				log.Error("Exiting workflow ticker: %v", c.Err())
				return nil
			}
		case <-tickHeart.C:
			go func() {
				if err := deleteDeadWorkers(c, db, store); err != nil {
					log.Warning("worker.deleteDeadWorkers> Error on deleteDeadWorkers : %v", err)
				}
			}()

			go func() {
				if err := disableDeadWorkers(c, db, store); err != nil {
					log.Warning("workflow.disableDeadWorkers> Error on disableDeadWorkers : %v", err)
				}
			}()
		}
	}
}
