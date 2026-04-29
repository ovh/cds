package hooks

import (
	"context"
	"time"

	"go.opencensus.io/trace"

	"github.com/ovh/cds/sdk/telemetry"
	"github.com/rockbears/log"
)

// dequeueMaintenanceRepositoryEvent consumes events from the maintenance queue.
// This goroutine runs continuously and is NOT paused by maintenance mode,
// allowing manual workflow triggers to be processed even during maintenance.
func (s *Service) dequeueMaintenanceRepositoryEvent(ctx context.Context) {
	for {
		if ctx.Err() != nil {
			log.ErrorWithStackTrace(ctx, ctx.Err())
			return
		}

		// Dequeuing context
		var eventKey string
		if ctx.Err() != nil {
			return
		}

		// Get next EventKEY from maintenance queue
		if err := s.Cache.DequeueWithContext(ctx, repositoryEventMaintenanceQueue, 250*time.Millisecond, &eventKey); err != nil {
			continue
		}
		s.Dao.dequeuedRepositoryEventIncr()
		if eventKey == "" {
			continue
		}
		log.Info(ctx, "dequeueMaintenanceRepositoryEvent> work on event: %s", eventKey)
		ctx := telemetry.New(ctx, s, "hooks.dequeueMaintenanceRepositoryEvent", nil, trace.SpanKindUnspecified)
		if err := s.manageRepositoryEvent(ctx, eventKey); err != nil {
			log.ErrorWithStackTrace(ctx, err)
			continue
		}
	}
}
