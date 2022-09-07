package api

import (
	"context"
	"time"

	"github.com/go-gorp/gorp"
	"github.com/rockbears/log"

	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/sdk"
)

func (api *API) cleanWorkflowRunSecrets(ctx context.Context) {
	// Load workflow run older than now - snapshot retention delay
	maxRetentionDate := time.Now().Add(-time.Hour * time.Duration(24*api.Config.Secrets.SnapshotRetentionDelay))

	db := api.mustDB()

	delay := 10 * time.Minute
	if api.Config.Secrets.SnapshotRetentionDelay > 0 {
		delay = time.Duration(api.Config.Secrets.SnapshotRetentionDelay) * time.Minute
	}

	limit := int64(100)
	if api.Config.Secrets.SnapshotCleanBatchSize > 0 {
		limit = api.Config.Secrets.SnapshotCleanBatchSize
	}

	log.Info(ctx, "Starting workflow run secrets clean routine")

	ticker := time.NewTicker(delay)

	for range ticker.C {
		runIDs, err := workflow.LoadRunsIDsCreatedBefore(ctx, db, maxRetentionDate, limit)
		if err != nil {
			log.ErrorWithStackTrace(ctx, err)
			continue
		}
		for _, id := range runIDs {
			if err := api.cleanWorkflowRunSecretsForRun(ctx, db, id); err != nil {
				log.ErrorWithStackTrace(ctx, err)
			}
		}
	}
}

func (api *API) cleanWorkflowRunSecretsForRun(ctx context.Context, db *gorp.DbMap, workflowRunID int64) error {
	tx, err := db.Begin()
	if err != nil {
		return sdk.WithStack(err)
	}
	defer tx.Rollback() // nolint
	if err := workflow.SetRunReadOnlyByID(ctx, tx, workflowRunID); err != nil {
		return sdk.WithStack(err)
	}
	if err := workflow.DeleteRunSecretsByWorkflowRunID(ctx, tx, workflowRunID); err != nil {
		return sdk.WithStack(err)
	}
	if err := tx.Commit(); err != nil {
		return sdk.WithStack(err)
	}
	return nil
}
