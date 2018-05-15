package warning

import (
	"context"
	"strings"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

// Compute warnings from CDS events
func Compute(c context.Context, store cache.Store, DBFunc func() *gorp.DbMap, ch <-chan sdk.Event) {
	db := DBFunc()

	for {
		select {
		case <-c.Done():
			if c.Err() != nil {
				log.Error("Warning.Compute: %v", c.Err())
			}
			return
		case e := <-ch:
			tx, errT := db.Begin()
			if errT != nil {
				log.Warning("computeWithProjectEvent> Unable to start transaction")
				return
			}
			if strings.HasPrefix(e.EventType, "sdk.EventProject") {
				if err := computeWithProjectEvent(tx, store, e); err != nil {
					log.Warning("warning.Compute: unable to compute project event: %v", err)
					_ = tx.Rollback()
					continue
				}
				commit(tx)
				continue
			}
			if strings.HasPrefix(e.EventType, "sdk.EventApplication") {
				computeWithApplicationEvent(tx, store, e)
				commit(tx)
				continue
			}
			if strings.HasPrefix(e.EventType, "sdk.EventEnvironment") {
				computeWithEnvironmentEvent(tx, store, e)
				commit(tx)
				continue
			}
			if strings.HasPrefix(e.EventType, "sdk.EventPipeline") {
				computeWithPipelineEvent(tx, store, e)
				commit(tx)
				continue
			}
			if strings.HasPrefix(e.EventType, "sdk.EventWorkflow") {
				computeWithWorkflowEvent(tx, store, e)
				commit(tx)
				continue
			}
			_ = tx.Rollback()
		}
	}
}

func commit(tx *gorp.Transaction) {
	if err := tx.Commit(); err != nil {
		log.Warning("ComputeWarning.commit: unable to commit transanction: %v", err)
		_ = tx.Rollback()
	}
	return
}
