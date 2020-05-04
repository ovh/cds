package ascode

import (
	"context"
	"database/sql"

	"github.com/lib/pq"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/sdk"
)

// LoadAsCodeByPRID Load as code events for the given pullrequest id
func LoadAsCodeByPRID(ctx context.Context, db gorp.SqlExecutor, ID int64) (sdk.AsCodeEvent, error) {
	query := gorpmapping.NewQuery("SELECT * FROM as_code_events WHERE pullrequest_id = $1").Args(ID)
	var event dbAsCodeEvents
	if _, err := gorpmapping.Get(ctx, db, query, &event); err != nil {
		if err == sql.ErrNoRows {
			return sdk.AsCodeEvent{}, sdk.WithStack(sdk.ErrNotFound)
		}
		return sdk.AsCodeEvent{}, sdk.WrapError(err, "Unable to load as code event")
	}
	return sdk.AsCodeEvent(event), nil
}

// LoadAsCodeEventByRepos Load as code events for the given repositories
func LoadAsCodeEventByRepos(ctx context.Context, db gorp.SqlExecutor, repos []string) ([]sdk.AsCodeEvent, error) {
	query := gorpmapping.NewQuery("SELECT * FROM as_code_events where from_repository = ANY($1)").Args(pq.StringArray(repos))
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

// LoadAsCodeEventByWorkflowID Load as code events for the given workflow
func LoadAsCodeEventByWorkflowID(ctx context.Context, db gorp.SqlExecutor, workflowID int64) ([]sdk.AsCodeEvent, error) {
	query := gorpmapping.NewQuery("SELECT * FROM as_code_events where (data->'workflows')::jsonb ? $1").Args(workflowID)
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

// LoadAsCodeEventByRepo Load as code events for the given repo
func LoadAsCodeEventByRepo(ctx context.Context, db gorp.SqlExecutor, fromRepo string) ([]sdk.AsCodeEvent, error) {
	query := gorpmapping.NewQuery("SELECT * FROM as_code_events where from_repository = $1").Args(fromRepo)
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

func InsertOrUpdateAsCodeEvent(db gorp.SqlExecutor, asCodeEvent *sdk.AsCodeEvent) error {
	if asCodeEvent.ID == 0 {
		return insertAsCodeEvent(db, asCodeEvent)
	}
	return updateAsCodeEvent(db, asCodeEvent)
}

func insertAsCodeEvent(db gorp.SqlExecutor, asCodeEvent *sdk.AsCodeEvent) error {
	dbEvent := dbAsCodeEvents(*asCodeEvent)
	if err := gorpmapping.Insert(db, &dbEvent); err != nil {
		return sdk.WrapError(err, "unable to insert as code event")
	}
	asCodeEvent.ID = dbEvent.ID
	return nil
}

func updateAsCodeEvent(db gorp.SqlExecutor, asCodeEvent *sdk.AsCodeEvent) error {
	dbEvent := dbAsCodeEvents(*asCodeEvent)
	if err := gorpmapping.Update(db, &dbEvent); err != nil {
		return sdk.WrapError(err, "unable to update as code event")
	}
	return nil
}

func DeleteAsCodeEvent(db gorp.SqlExecutor, asCodeEvent sdk.AsCodeEvent) error {
	dbEvent := dbAsCodeEvents(asCodeEvent)
	if err := gorpmapping.Delete(db, &dbEvent); err != nil {
		return sdk.WrapError(err, "unable to delete as code event")
	}
	return nil
}
