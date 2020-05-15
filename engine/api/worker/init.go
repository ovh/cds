package worker

import (
	"context"
	"time"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/sdk/log"
)

//Initialize init the package
func Initialize(c context.Context, DBFunc func() *gorp.DbMap, store cache.Store) error {
	db := DBFunc()
	tickHeart := time.NewTicker(10 * time.Second)

	for {
		select {
		case <-c.Done():
			if c.Err() != nil {
				log.Error(c, "Exiting workflow ticker: %v", c.Err())
				return nil
			}
		case <-tickHeart.C:
			go func() {
				if err := DeleteDeadWorkers(c, db); err != nil {
					log.Warning(c, "worker.deleteDeadWorkers> Error on deleteDeadWorkers : %v", err)
				}
			}()

			go func() {
				if err := DisableDeadWorkers(c, db); err != nil {
					log.Warning(c, "workflow.disableDeadWorkers> Error on disableDeadWorkers : %v", err)
				}
			}()
		}
	}
}
