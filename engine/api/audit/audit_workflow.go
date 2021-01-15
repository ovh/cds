package audit

import (
	"context"
	"strings"
	"time"

	"github.com/go-gorp/gorp"
	"github.com/rockbears/log"

	"github.com/ovh/cds/engine/api/event"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/sdk"
)

// ComputeWorkflowAudit Compute audit on workflow
func ComputeWorkflowAudit(ctx context.Context, DBFunc func() *gorp.DbMap) {
	chanEvent := make(chan sdk.Event)
	event.Subscribe(chanEvent)
	deleteTicker := time.NewTicker(15 * time.Minute)
	defer deleteTicker.Stop()

	db := DBFunc()
	for {
		select {
		case <-ctx.Done():
			if ctx.Err() != nil {
				log.Error(ctx, "ComputeWorkflowAudit> Exiting: %v", ctx.Err())
				return
			}
		case <-deleteTicker.C:
			if err := workflow.PurgeAudits(ctx, DBFunc()); err != nil {
				log.Error(ctx, "ComputeWorkflowAudit> Purge error: %v", err)
			}
		case e := <-chanEvent:
			if !strings.HasPrefix(e.EventType, "sdk.EventWorkflow") {
				continue
			}

			if audit, ok := workflow.Audits[e.EventType]; ok {
				if err := audit.Compute(ctx, db, e); err != nil {
					log.Warn(ctx, "ComputeAudit> Unable to compute audit on event %s: %v", e.EventType, err)
				}
			}
		}
	}
}
