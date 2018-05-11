package warning

import (
	"context"
	"strings"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

const (
	MISSING_PROJECT_VARIABLE  = "MISSING_PROJECT_VARIABLE"
	UNUSED_PROJECT_VARIABLE   = "UNUSED_PROJECT_VARIABLE"
	MISSING_PROJECT_KEY       = "MISSING_PROJECT_KEY"
	UNUSED_PROJECT_KEY        = "UNUSED_PROJECT_KEY"
	MISSING_VCS_CONFIGURATION = "MISSING_VCS_CONFIGURATION"

	MISSING_APPLICATION_VARIABLE = "MISSING_APPLICATION_VARIABLE"
	UNUSED_APPLICATION_VARIABLE  = "UNUSED_APPLICATION_VARIABLE"
	MISSING_APPLICATION_KEY      = "MISSING_APPLICATION_KEY"
	UNUSED_APPLICATION_KEY       = "UNUSED_APPLICATION_KEY"

	MISSING_ENVIRONMENT_VARIABLE = "MISSING_ENVIRONMENT_VARIABLE"
	UNUSED_ENVIRONMENT_VARIABLE  = "UNUSED_ENVIRONMENT_VARIABLE"
	MISSING_ENVIRONMENT_KEY      = "MISSING_ENVIRONMENT_KEY"
	UNUSED_ENVIRONMENT_KEY       = "UNUSED_ENVIRONMENT_KEY"

	MISSING_PIPELINE_PARAMETER = "MISSING_PIPELINE_PARAMETER"
	UNUSED_PIPELINE_PARAMETER  = "UNUSED_PIPELINE_PARAMETER"
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
