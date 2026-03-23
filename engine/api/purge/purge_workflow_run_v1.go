package purge

import (
	"context"
	"time"

	"github.com/go-gorp/gorp"
	"github.com/rockbears/log"

	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/sdk"
)

// MarkOldWorkflowRunsV1 is a goroutine that periodically marks old workflow v1 runs as to_delete
// based on their last_modified date and the configured max retention days.
func MarkOldWorkflowRunsV1(ctx context.Context, DBFunc func() *gorp.DbMap, maxRetentionDays, schedulingSeconds, batchSize int64) {
	if maxRetentionDays <= 0 {
		log.Info(ctx, "purge> MarkOldWorkflowRunsV1 disabled (maxRetentionDays=%d)", maxRetentionDays)
		return
	}

	tickDuration := time.Duration(schedulingSeconds) * time.Second
	if tickDuration <= 0 {
		tickDuration = 60 * time.Second
	}
	if batchSize <= 0 {
		batchSize = 100
	}

	log.Info(ctx, "purge> MarkOldWorkflowRunsV1 enabled (maxRetentionDays=%d, schedulingSeconds=%f, batchSize=%d)", maxRetentionDays, tickDuration.Seconds(), batchSize)

	ticker := time.NewTicker(tickDuration)
	defer ticker.Stop()

	var cursor time.Time

	for {
		select {
		case <-ctx.Done():
			if ctx.Err() != nil {
				log.Error(ctx, "purge> MarkOldWorkflowRunsV1 exiting: %v", ctx.Err())
			}
			return
		case <-ticker.C:
			db := DBFunc()
			if db == nil {
				continue
			}
			var err error
			cursor, err = markOldWorkflowRunsV1Batch(ctx, db, maxRetentionDays, batchSize, cursor)
			if err != nil {
				ctx = sdk.ContextWithStacktrace(ctx, err)
				log.Error(ctx, "purge> MarkOldWorkflowRunsV1: %v", err)
				cursor = time.Time{}
			}
		}
	}
}

func markOldWorkflowRunsV1Batch(ctx context.Context, db *gorp.DbMap, maxRetentionDays, batchSize int64, cursor time.Time) (time.Time, error) {
	cutoff := time.Now().AddDate(0, 0, -int(maxRetentionDays))

	// Select only terminal runs (exclude non-terminal statuses)
	query := `
		SELECT id, last_modified
		FROM workflow_run
		WHERE to_delete = false
		AND status NOT IN ($1, $2, $3, $4, $5)
		AND last_modified < $6
		AND last_modified > $7
		ORDER BY last_modified ASC
		LIMIT $8
		FOR UPDATE SKIP LOCKED
	`

	type runRow struct {
		ID           int64     `db:"id"`
		LastModified time.Time `db:"last_modified"`
	}

	tx, err := db.Begin()
	if err != nil {
		return cursor, sdk.WithStack(err)
	}
	defer tx.Rollback() //nolint

	var rows []runRow
	if _, err := tx.Select(&rows, query,
		sdk.StatusWaiting, sdk.StatusBuilding, sdk.StatusCrafting, sdk.StatusScheduling, sdk.StatusPending,
		cutoff, cursor, batchSize,
	); err != nil {
		return cursor, sdk.WithStack(err)
	}

	if len(rows) == 0 {
		log.Info(ctx, "purge> MarkOldWorkflowRunsV1 found no runs to mark as to_delete (cursor: %v)", cursor)
		_ = tx.Commit()
		return time.Time{}, nil
	}

	ids := make([]int64, len(rows))
	for i := range rows {
		log.Info(ctx, "purge> MarkOldWorkflowRunsV1: marking run %d as to_delete (last_modified: %v)", rows[i].ID, rows[i].LastModified)
		ids[i] = rows[i].ID
	}

	if err := workflow.MarkWorkflowRunsAsDelete(tx, ids); err != nil {
		return cursor, err
	}

	if err := tx.Commit(); err != nil {
		return cursor, sdk.WithStack(err)
	}

	nextCursor := rows[len(rows)-1].LastModified
	log.Info(ctx, "purge> MarkOldWorkflowRunsV1: marked %d runs as to_delete (cursor: %v)", len(rows), nextCursor)

	return nextCursor, nil
}
