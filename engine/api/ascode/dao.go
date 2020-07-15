package ascode

import (
	"context"
	"database/sql"

	"github.com/lib/pq"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/gorpmapping"
)

// LoadEventByWorkflowIDAndPullRequest returns a as code event if exists for given workflow holder and pull request info.
func LoadEventByWorkflowIDAndPullRequest(ctx context.Context, db gorp.SqlExecutor, workflowID int64, pullRequestRepo string, pullRequestID int64) (*sdk.AsCodeEvent, error) {
	query := gorpmapping.NewQuery(`
    SELECT *
    FROM as_code_events
    WHERE workflow_id = $1 AND from_repository = $2 AND pullrequest_id = $3
  `).Args(workflowID, pullRequestRepo, pullRequestID)
	var dbEvent dbAsCodeEvents
	if _, err := gorpmapping.Get(ctx, db, query, &dbEvent); err != nil {
		if err == sql.ErrNoRows {
			return nil, sdk.WithStack(sdk.ErrNotFound)
		}
		return nil, sdk.WrapError(err, "unable to load as code event")
	}
	event := sdk.AsCodeEvent(dbEvent)
	return &event, nil
}

func LoadEventsByWorkflowIDs(ctx context.Context, db gorp.SqlExecutor, workflowIDs []int64) ([]sdk.AsCodeEvent, error) {
	query := gorpmapping.NewQuery("SELECT * FROM as_code_events where workflow_id = ANY($1)").Args(pq.Int64Array(workflowIDs))
	var events []dbAsCodeEvents
	if err := gorpmapping.GetAll(ctx, db, query, &events); err != nil {
		return nil, sdk.WrapError(err, "Unable to load as code events")
	}

	asCodeEvents := make([]sdk.AsCodeEvent, len(events))
	for i := range events {
		asCodeEvents[i] = sdk.AsCodeEvent(events[i])
	}
	return asCodeEvents, nil
}

// LoadEventsByWorkflowID returns as code events for the given workflow.
func LoadEventsByWorkflowID(ctx context.Context, db gorp.SqlExecutor, workflowID int64) ([]sdk.AsCodeEvent, error) {
	query := gorpmapping.NewQuery(`
    SELECT *
    FROM as_code_events
    WHERE workflow_id = $1
  `).Args(workflowID)
	var dbEvents []dbAsCodeEvents
	if err := gorpmapping.GetAll(ctx, db, query, &dbEvents); err != nil {
		return nil, sdk.WrapError(err, "unable to load as code events")
	}
	events := make([]sdk.AsCodeEvent, len(dbEvents))
	for i := range dbEvents {
		events[i] = sdk.AsCodeEvent(dbEvents[i])
	}
	return events, nil
}

// UpsertEvent insert or update given ascode event.
func UpsertEvent(db gorp.SqlExecutor, event *sdk.AsCodeEvent) error {
	if event.ID == 0 {
		return insertEvent(db, event)
	}
	return UpdateEvent(db, event)
}

func insertEvent(db gorp.SqlExecutor, event *sdk.AsCodeEvent) error {
	dbEvent := dbAsCodeEvents(*event)
	if err := gorpmapping.Insert(db, &dbEvent); err != nil {
		return sdk.WrapError(err, "unable to insert as code event")
	}
	event.ID = dbEvent.ID
	return nil
}

// UpdateEvent in database.
func UpdateEvent(db gorp.SqlExecutor, event *sdk.AsCodeEvent) error {
	dbEvent := dbAsCodeEvents(*event)
	if err := gorpmapping.Update(db, &dbEvent); err != nil {
		return sdk.WrapError(err, "unable to update as code event")
	}
	return nil
}

func deleteEvent(db gorp.SqlExecutor, event *sdk.AsCodeEvent) error {
	dbEvent := dbAsCodeEvents(*event)
	if err := gorpmapping.Delete(db, &dbEvent); err != nil {
		return sdk.WrapError(err, "unable to delete as code event")
	}
	return nil
}
