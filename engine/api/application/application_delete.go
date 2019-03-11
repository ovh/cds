package application

import (
	"github.com/go-gorp/gorp"
	"github.com/lib/pq"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/sdk"
)

// DeleteApplication Delete the given application
func DeleteApplication(db gorp.SqlExecutor, applicationID int64) error {
	// Delete variables
	if err := DeleteAllVariable(db, applicationID); err != nil {
		return err
	}

	// Delete application_key
	if err := DeleteAllApplicationKeys(db, applicationID); err != nil {
		return err
	}

	query := `DELETE FROM application WHERE id=$1`
	if _, err := db.Exec(query, applicationID); err != nil {
		if e, ok := err.(*pq.Error); ok {
			switch e.Code {
			case gorpmapping.ViolateForeignKeyPGCode:
				err = sdk.NewErrorWithStack(err, sdk.ErrApplicationUsedByWorkflow)
			}
		}
		return sdk.WrapError(err, "cannot delete application")
	}

	return nil
}

//DeleteAllApplicationKeys deletes all application keys
func DeleteAllApplicationKeys(db gorp.SqlExecutor, applicationID int64) error {
	query := `DELETE FROM application_key WHERE application_id = $1`
	if _, err := db.Exec(query, applicationID); err != nil {
		return sdk.WrapError(err, "cannot delete application key")
	}
	return nil
}
