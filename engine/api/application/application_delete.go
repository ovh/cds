package application

import (
	"fmt"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/trigger"
	"github.com/ovh/cds/engine/log"
)

// DeleteApplication Delete the given application
func DeleteApplication(db gorp.SqlExecutor, applicationID int64) error {

	// Delete variables
	if err := DeleteAllVariable(db, applicationID); err != nil {
		log.Warning("DeleteApplication> Cannot delete application variable: %s\n", err)
		return err
	}

	// Delete groups
	query := `DELETE FROM application_group WHERE application_id = $1`
	if _, err := db.Exec(query, applicationID); err != nil {
		log.Warning("DeleteApplication> Cannot delete application gorup: %s\n", err)
		return err
	}

	// Delete application_pipeline
	if err := DeleteAllApplicationPipeline(db, applicationID); err != nil {
		log.Warning("DeleteApplication> Cannot delete application pipeline: %s\n", err)
		return err
	}

	// Delete pipeline builds
	var ids []int64
	query = `SELECT id FROM pipeline_build WHERE application_id = $1`
	rows, err := db.Query(query, applicationID)
	if err != nil {
		return fmt.Errorf("DeleteApplication> Cannot select application pipeline build> %s\n", err)
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

	// Delete application artifact left
	query = `DELETE FROM artifact WHERE application_id = $1`
	if _, err = db.Exec(query, applicationID); err != nil {
		log.Warning("DeleteApplication> Cannot delete old artifacts: %s\n", err)
		return err
	}

	// Delete hook
	query = `DELETE FROM hook WHERE application_id = $1`
	if _, err := db.Exec(query, applicationID); err != nil {
		log.Warning("DeleteApplication> Cannot delete hook: %s\n", err)
		return err
	}

	// Delete poller execution
	query = `DELETE FROM poller_execution WHERE application_id = $1`
	if _, err := db.Exec(query, applicationID); err != nil {
		log.Warning("DeleteApplication> Cannot delete poller execution: %s\n", err)
		return err
	}

	// Delete poller
	query = `DELETE FROM poller WHERE application_id = $1`
	if _, err := db.Exec(query, applicationID); err != nil {
		log.Warning("DeleteApplication> Cannot delete poller: %s\n", err)
		return err
	}

	// Delete triggers
	if err := trigger.DeleteApplicationTriggers(db, applicationID); err != nil {
		return err
	}

	query = `DELETE FROM application WHERE id=$1`
	if _, err := db.Exec(query, applicationID); err != nil {
		log.Warning("DeleteApplication> Cannot delete application: %s\n", err)
		return err
	}

	// Update project
	query = `
		UPDATE project
		SET last_modified = current_timestamp
		WHERE id IN (
			select project_id from application where id = $1
		)
	`
	_, err = db.Exec(query, applicationID)
	return err
}
