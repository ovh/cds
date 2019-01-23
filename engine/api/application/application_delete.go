package application

import (
	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/sdk"
)

// DeleteApplication Delete the given application
func DeleteApplication(db gorp.SqlExecutor, applicationID int64) error {

	// Delete variables
	if err := DeleteAllVariable(db, applicationID); err != nil {
		return sdk.WrapError(err, "Cannot delete application variable")
	}

	// Delete groups
	query := `DELETE FROM application_group WHERE application_id = $1`
	if _, err := db.Exec(query, applicationID); err != nil {
		return sdk.WrapError(err, "Cannot delete application group")
	}

	// Delete application_key
	if err := DeleteAllApplicationKeys(db, applicationID); err != nil {
		return sdk.WrapError(err, "DeleteApplication")
	}

	query = `DELETE FROM application WHERE id=$1`
	if _, err := db.Exec(query, applicationID); err != nil {
		return sdk.WrapError(err, "Cannot delete application")
	}

	return nil
}

//DeleteAllApplicationKeys deletes all application keys
func DeleteAllApplicationKeys(db gorp.SqlExecutor, applicationID int64) error {
	query := `DELETE FROM application_key WHERE application_id = $1`
	if _, err := db.Exec(query, applicationID); err != nil {
		return sdk.WrapError(err, "Cannot delete application key")
	}
	return nil
}
