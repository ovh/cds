package workflow

import (
	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/sdk"
)

// LoadAsCodeEvent Load as code events for the given workflow
func LoadAsCodeEvent(db gorp.SqlExecutor, workflowID int64) ([]sdk.AsCodeEvent, error) {
	query := `
		SELECT * FROM workflow_as_code_events WHERE workflow_id = $1
	`
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

func inserAsCodeEvent(db gorp.SqlExecutor, asCodeEvent sdk.AsCodeEvent) error {
	dbEvent := dbAsCodeEvents(asCodeEvent)
	if err := db.Insert(&dbEvent); err != nil {
		return sdk.WrapError(err, "unable to insert as code event")
	}
	return nil
}

func deleteAsCodeEvent(db gorp.SqlExecutor, asCodeEvent sdk.AsCodeEvent) error {
	dbEvent := dbAsCodeEvents(asCodeEvent)
	if _, err := db.Delete(&dbEvent); err != nil {
		return sdk.WrapError(err, "unable to delete as code event")
	}
	return nil
}
