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
			if strings.HasPrefix(e.EventType, "EventProject") {
				computeWithProjectEvent(db, store, e)
				return
			}
			if strings.HasPrefix(e.EventType, "EventApplication") {
				computeWithApplicationEvent(db, store, e)
				return
			}
			if strings.HasPrefix(e.EventType, "EventEnvironment") {
				computeWithEnvironmentEvent(db, store, e)
				return
			}
			if strings.HasPrefix(e.EventType, "EventPipeline") {
				computeWithPipelineEvent(db, store, e)
				return
			}
			if strings.HasPrefix(e.EventType, "EventWorkflow") {
				computeWithWorkflowEvent(db, store, e)
				return
			}
		}
	}
}
