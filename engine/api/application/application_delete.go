package application

import (
	"fmt"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/trigger"
	"github.com/ovh/cds/sdk"
)

// DeleteApplication Delete the given application
func DeleteApplication(db gorp.SqlExecutor, applicationID int64) error {

	// Delete variables
	if err := DeleteAllVariable(db, applicationID); err != nil {
		return sdk.WrapError(err, "DeleteApplication> Cannot delete application variable")
	}

	// Delete groups
	query := `DELETE FROM application_group WHERE application_id = $1`
	if _, err := db.Exec(query, applicationID); err != nil {
		return sdk.WrapError(err, "DeleteApplication> Cannot delete application group")
	}

	// Delete application_pipeline
	if err := DeleteAllApplicationPipeline(db, applicationID); err != nil {
		return sdk.WrapError(err, "DeleteApplication> Cannot delete application pipeline")
	}

	// Delete pipeline builds
	//FIXME
	var ids []int64
	query = `SELECT id FROM pipeline_build WHERE application_id = $1`
	rows, err := db.Query(query, applicationID)
	if err != nil {
		return fmt.Errorf("DeleteApplication> Cannot select application pipeline build> %s", err)
	}
	var id int64
	for rows.Next() {
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return err
		}
		ids = append(ids, id)
	}
	rows.Close()
	for _, id := range ids {
		if err := pipeline.DeletePipelineBuildByID(db, id); err != nil {
			return fmt.Errorf("DeleteApplication> Cannot delete pb %d> %s", id, err)
		}
	}

	// Delete application_key
	if err := DeleteAllApplicationKeys(db, applicationID); err != nil {
		return sdk.WrapError(err, "DeleteApplication")
	}

	// Delete application artifact left
	query = `DELETE FROM artifact WHERE application_id = $1`
	if _, err = db.Exec(query, applicationID); err != nil {
		return sdk.WrapError(err, "DeleteApplication> Cannot delete old artifacts")
	}

	// Delete hook
	query = `DELETE FROM hook WHERE application_id = $1`
	if _, err := db.Exec(query, applicationID); err != nil {
		return sdk.WrapError(err, "DeleteApplication> Cannot delete hook")
	}

	// Delete poller execution
	query = `DELETE FROM poller_execution WHERE application_id = $1`
	if _, err := db.Exec(query, applicationID); err != nil {
		return sdk.WrapError(err, "DeleteApplication> Cannot delete poller execution")
	}

	// Delete poller
	query = `DELETE FROM poller WHERE application_id = $1`
	if _, err := db.Exec(query, applicationID); err != nil {
		return sdk.WrapError(err, "DeleteApplication> Cannot delete poller")
	}

	// Delete triggers
	if err := trigger.DeleteApplicationTriggers(db, applicationID); err != nil {
		return err
	}

	query = `DELETE FROM application WHERE id=$1`
	if _, err := db.Exec(query, applicationID); err != nil {
		return sdk.WrapError(err, "DeleteApplication> Cannot delete application")
	}
	return nil
}

//DeleteAllApplicationKeys deletes all application keys
func DeleteAllApplicationKeys(db gorp.SqlExecutor, applicationID int64) error {
	query := `DELETE FROM application_key WHERE application_id = $1`
	if _, err := db.Exec(query, applicationID); err != nil {
		return sdk.WrapError(err, "DeleteAllApplicationKey> Cannot delete application key")
	}
	return nil
}
