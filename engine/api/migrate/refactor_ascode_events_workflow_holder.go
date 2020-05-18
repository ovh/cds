package migrate

import (
	"context"
	"database/sql"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/ascode"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func RefactorAsCodeEventsWorkflowHolder(ctx context.Context, db *gorp.DbMap) error {
	query := "SELECT id FROM as_code_events WHERE workflow_id IS NULL"
	rows, err := db.Query(query)
	if err == sql.ErrNoRows {
		return nil
	}
	if err != nil {
		return sdk.WithStack(err)
	}

	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			rows.Close() // nolint
			return sdk.WithStack(err)
		}
		ids = append(ids, id)
	}

	if err := rows.Close(); err != nil {
		return sdk.WithStack(err)
	}

	var mError = new(sdk.MultiError)
	for _, id := range ids {
		if err := refactorAsCodeEventsWorkflowHolder(ctx, db, id); err != nil {
			mError.Append(err)
			log.Error(ctx, "migrate.RefactorAsCodeEventsWorkflowHolder> unable to migrate as_code_event %d: %v", id, err)
		}
	}

	if mError.IsEmpty() {
		return nil
	}
	return mError
}

func refactorAsCodeEventsWorkflowHolder(ctx context.Context, db *gorp.DbMap, eventID int64) error {
	log.Info(ctx, "migrate.refactorAsCodeEventsWorkflowHolder> as_code_event %d migration begin", eventID)

	tx, err := db.Begin()
	if err != nil {
		return sdk.WithStack(err)
	}

	queryEvent := `
    SELECT id, pullrequest_id, pullrequest_url, username, creation_date, from_repository, migrate, data
	  FROM as_code_events
	  WHERE id = $1
	  AND workflow_id IS NULL
    FOR UPDATE SKIP LOCKED
  `

	defer tx.Rollback() // nolint

	var asCodeEvent sdk.AsCodeEvent
	if err := tx.QueryRow(queryEvent, eventID).Scan(
		&asCodeEvent.ID,
		&asCodeEvent.PullRequestID,
		&asCodeEvent.PullRequestURL,
		&asCodeEvent.Username,
		&asCodeEvent.CreateDate,
		&asCodeEvent.FromRepo,
		&asCodeEvent.Migrate,
		&asCodeEvent.Data,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil
		}
		return sdk.WrapError(err, "unable to select and lock as code event with id: %d", eventID)
	}

	queryWorkflow := `
    SELECT id
    FROM workflow
    WHERE from_repository = $1
    LIMIT 1
  `

	if err := tx.QueryRow(queryWorkflow, asCodeEvent.FromRepo).Scan(
		&asCodeEvent.WorkflowID,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil
		}
		return sdk.WrapError(err, "unable to select a workflow that match event repository address: %s for event %d", asCodeEvent.FromRepo, eventID)
	}

	if err := ascode.UpdateEvent(tx, &asCodeEvent); err != nil {
		return sdk.WithStack(err)
	}

	if err := tx.Commit(); err != nil {
		return sdk.WithStack(err)
	}

	log.Info(ctx, "migrate.refactorAsCodeEventsWorkflowHolder> as_code_event %d migration end", eventID)
	return nil
}
