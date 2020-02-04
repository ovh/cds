package audit

import (
	"context"
	"time"

	"github.com/go-gorp/gorp"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/sdk/log"
)

// ComputePipelineAudit Compute audit on workflow
func ComputePipelineAudit(ctx context.Context, DBFunc func() *gorp.DbMap) {
	deleteTicker := time.NewTicker(15 * time.Minute)
	defer deleteTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			if ctx.Err() != nil {
				log.Error(ctx, "pipeline.ComputeAudit> Exiting: %v", ctx.Err())
				return
			}
		case <-deleteTicker.C:
			if err := pipeline.PurgeAudits(ctx, DBFunc()); err != nil {
				log.Error(ctx, "pipeline.ComputeAudit> Purge error: %v", err)
			}
		}
	}
}
