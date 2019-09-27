package ascode

import (
	"database/sql"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/sdk"
)

// LoadAsCodeByPRID Load as code events for the given pullrequest id
func LoadAsCodeByPRID(db gorp.SqlExecutor, ID int64) (sdk.AsCodeEvent, error) {
	query := "SELECT * FROM as_code_events WHERE pullrequest_id = $1"
	var event dbAsCodeEvents
	if err := db.SelectOne(&event, query, ID); err != nil {
		if err == sql.ErrNoRows {
			return sdk.AsCodeEvent{}, sdk.ErrNotFound
		}
		return sdk.AsCodeEvent{}, sdk.WrapError(err, "Unable to load as code event")
	}
	return sdk.AsCodeEvent(event), nil
}

// LoadAsCodeEventByWorkflowID Load as code events for the given workflow
func LoadAsCodeEventByWorkflowID(db gorp.SqlExecutor, workflowID int64) ([]sdk.AsCodeEvent, error) {
	query := "SELECT * FROM as_code_events where (data->'workflows')::jsonb ? $1;"
	var events []dbAsCodeEvents
	if _, err := db.Select(&events, query, workflowID); err != nil {
		return nil, sdk.WrapError(err, "Unable to load as code events")
	}

	asCodeEvents := make([]sdk.AsCodeEvent, len(events))
	for i := range events {
		asCodeEvents[i] = sdk.AsCodeEvent(events[i])
	}
	return asCodeEvents, nil
}

// LoadAsCodeEventByRepo Load as code events for the given repo
func LoadAsCodeEventByRepo(db gorp.SqlExecutor, fromRepo string) ([]sdk.AsCodeEvent, error) {
	query := "SELECT * FROM as_code_events where from_repository = $1;"
	var events []dbAsCodeEvents
	if _, err := db.Select(&events, query, fromRepo); err != nil {
		return nil, sdk.WrapError(err, "Unable to load as code events")
	}

	asCodeEvents := make([]sdk.AsCodeEvent, len(events))
	for i := range events {
		asCodeEvents[i] = sdk.AsCodeEvent(events[i])
	}
	return asCodeEvents, nil
}

func insertOrUpdateAsCodeEvent(db gorp.SqlExecutor, asCodeEvent *sdk.AsCodeEvent) error {
	if asCodeEvent.ID == 0 {
		return insertAsCodeEvent(db, asCodeEvent)
	}
	return updateAsCodeEvent(db, asCodeEvent)
}

func insertAsCodeEvent(db gorp.SqlExecutor, asCodeEvent *sdk.AsCodeEvent) error {
	dbEvent := dbAsCodeEvents(*asCodeEvent)
	if err := db.Insert(&dbEvent); err != nil {
		return sdk.WrapError(err, "unable to insert as code event")
	}
	asCodeEvent.ID = dbEvent.ID
	return nil
}

func updateAsCodeEvent(db gorp.SqlExecutor, asCodeEvent *sdk.AsCodeEvent) error {
	dbEvent := dbAsCodeEvents(*asCodeEvent)
	if _, err := db.Update(&dbEvent); err != nil {
		return sdk.WrapError(err, "unable to update as code event")
	}
	return nil
}

func DeleteAsCodeEvent(db gorp.SqlExecutor, asCodeEvent sdk.AsCodeEvent) error {
	dbEvent := dbAsCodeEvents(asCodeEvent)
	if _, err := db.Delete(&dbEvent); err != nil {
		return sdk.WrapError(err, "unable to delete as code event")
	}
	return nil
}
