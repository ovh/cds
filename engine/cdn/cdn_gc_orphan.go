package cdn

import (
	"context"
	"time"

	"github.com/rockbears/log"

	"github.com/ovh/cds/engine/cdn/item"
	"github.com/ovh/cds/sdk"
)

// itemOrphanCleanup is a goroutine that periodically checks for orphan CDN items
// (workflow v1 items whose workflow_run no longer exists in the CDS API) and marks
// them as to_delete. Work is distributed across CDN instances using
// SELECT FOR UPDATE SKIP LOCKED at the PostgreSQL level.
func (s *Service) itemOrphanCleanup(ctx context.Context) {
	frequency := s.Cfg.OrphanCleanup.FrequencySeconds
	if frequency <= 0 {
		frequency = 60
	}

	tick := time.NewTicker(time.Duration(frequency) * time.Second)
	defer tick.Stop()

	// Pagination cursor: the creation date of the last processed item.
	// Advances through items across batches, resets to zero when the end is reached.
	var createdCursor time.Time

	for {
		select {
		case <-ctx.Done():
			if ctx.Err() != nil {
				log.Error(ctx, "cdn:orphan-cleanup: %v", ctx.Err())
			}
			return
		case <-tick.C:
			nextCursor, err := s.cleanOrphanItemsV1(ctx, createdCursor)
			if err != nil {
				ctx = sdk.ContextWithStacktrace(ctx, err)
				log.Error(ctx, "cdn:orphan-cleanup: error: %v", err)
				continue
			}
			createdCursor = nextCursor
		}
	}
}

// cleanOrphanItemsV1 processes one batch of v1 items created after the given cursor,
// checking whether their associated workflow_run still exists via the CDS API.
// Items whose run no longer exists are marked as to_delete.
// Returns the creation date of the last item processed as cursor for the next batch,
// or zero time when the end is reached to restart scanning.
//
// The SELECT FOR UPDATE SKIP LOCKED ensures that multiple CDN instances
// work on different items concurrently without overlap.
func (s *Service) cleanOrphanItemsV1(ctx context.Context, createdCursor time.Time) (time.Time, error) {
	batchSize := s.Cfg.OrphanCleanup.BatchSize
	if batchSize <= 0 {
		batchSize = 100
	}

	gracePeriodDays := s.Cfg.OrphanCleanup.GracePeriodDays
	if gracePeriodDays <= 0 {
		gracePeriodDays = 180
	}

	tx, err := s.mustDBWithCtx(ctx).Begin()
	if err != nil {
		return createdCursor, sdk.WithStack(err)
	}
	defer tx.Rollback() // nolint

	// Lock a batch of v1 items created after the cursor. Other CDN instances
	// will skip these rows and pick the next ones.
	items, err := item.LoadOldestItemIDsForOrphanCleanupV1(tx, createdCursor, batchSize, gracePeriodDays)
	if err != nil {
		return createdCursor, err
	}

	// No more items after cursor: reset to scan from the beginning
	if len(items) == 0 {
		return time.Time{}, sdk.WithStack(tx.Commit())
	}

	log.Info(ctx, "cdn:orphan-cleanup: checking %d items (cursor=%v)", len(items), createdCursor)

	// Deduplicate run IDs to minimize API calls. A single workflow run
	// typically has many associated items (logs, artifacts...).
	runIDExists := make(map[int64]*bool) // nil = not checked yet
	for _, it := range items {
		if _, ok := runIDExists[it.RunID]; !ok {
			runIDExists[it.RunID] = nil
		}
	}

	// Check each unique run_id against the CDS API
	for runID := range runIDExists {
		exists, err := s.Client.WorkflowRunExist(ctx, runID)
		if err != nil {
			// On API error, log and skip — we don't want to wrongly mark
			// items as orphans if the API is temporarily unavailable.
			log.Warn(ctx, "cdn:orphan-cleanup: unable to check run_id %d: %v", runID, err)
			// Mark as "exists" to be safe; we'll retry next batch
			t := true
			runIDExists[runID] = &t
			continue
		}
		runIDExists[runID] = &exists
	}

	// Collect IDs of items whose run no longer exists
	var orphanIDs []string
	for _, it := range items {
		existsPtr := runIDExists[it.RunID]
		if existsPtr != nil && !*existsPtr {
			orphanIDs = append(orphanIDs, it.ID)
		}
	}

	if len(orphanIDs) > 0 {
		if err := item.MarkItemsAsToDelete(tx, orphanIDs); err != nil {
			return createdCursor, err
		}
		log.Info(ctx, "cdn:orphan-cleanup: marked %d items as to_delete (out of %d checked)", len(orphanIDs), len(items))
	} else {
		log.Debug(ctx, "cdn:orphan-cleanup: no orphan items found in this batch of %d", len(items))
	}

	if err := tx.Commit(); err != nil {
		return createdCursor, sdk.WithStack(err)
	}

	// Advance cursor to the creation date of the last item in this batch
	return items[len(items)-1].Created, nil
}
