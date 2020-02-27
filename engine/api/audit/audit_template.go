package audit

import (
	"context"
	"strings"

	"github.com/go-gorp/gorp"
	"github.com/ovh/cds/engine/api/event"
	"github.com/ovh/cds/engine/api/workflowtemplate"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

// ComputeTemplateAudit compute audit on workflow template.
func ComputeTemplateAudit(ctx context.Context, DBFunc func() *gorp.DbMap) {
	chanEvent := make(chan sdk.Event)
	event.Subscribe(chanEvent)

	db := DBFunc()
	for {
		select {
		case <-ctx.Done():
			if ctx.Err() != nil {
				log.Error(ctx, "%v", sdk.WithStack(ctx.Err()))
				return
			}
		case e := <-chanEvent:
			if !strings.HasPrefix(e.EventType, "sdk.EventWorkflowTemplate") {
				continue
			}

			if audit, ok := workflowtemplate.Audits[e.EventType]; ok {
				if err := audit.Compute(ctx, db, e); err != nil {
					log.Warning(ctx, "%v", sdk.WrapError(err, "Unable to compute audit on event %s", e.EventType))
				}
			}
		}
	}
}
