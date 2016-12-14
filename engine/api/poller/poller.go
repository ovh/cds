package poller

import (
	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/database"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/log"
	"github.com/ovh/cds/sdk"
)

//InsertPoller insert or update a new poller in DB
func InsertPoller(db database.Executer, poller *sdk.RepositoryPoller) error {
	query := `
        INSERT INTO poller (application_id, pipeline_id, name, enabled, date_creation)
        VALUES ($1, $2, $3, $4, now())
		RETURNING application_id, pipeline_id
    `
	if _, err := db.Exec(query, poller.Application.ID, poller.Pipeline.ID, poller.Name, poller.Enabled); err != nil {
		log.Warning("InsertPoller> Error :%s", err)
		return err
	}
	return nil
}

//DeletePoller delete a poller from DB
func DeletePoller(db database.Executer, poller *sdk.RepositoryPoller) error {
	query := `
        DELETE FROM poller
        WHERE application_id = $1
        AND pipeline_id = $2
    `
	if _, err := db.Exec(query, poller.Application.ID, poller.Pipeline.ID); err != nil {
		log.Warning("DeletePoller> Error :%s", err)
		return err
	}
	return nil
}

// DeleteAllPollers  Delete all the poller of the given application
func DeleteAllPollers(db database.Executer, appID int64) error {
	query := "DELETE FROM poller WHERE application_id = $1"
	if _, err := db.Exec(query, appID); err != nil {
		log.Warning("DeleteAllPoller> Error :%s", err)
		return err
	}
	return nil
}

//UpdatePoller update the poller
func UpdatePoller(db database.Executer, poller *sdk.RepositoryPoller) error {
	query := `
        UPDATE  poller 
        SET enabled = $3, name = $4
        WHERE application_id = $1
        AND pipeline_id  = $2
    `
	if _, err := db.Exec(query, poller.Application.ID, poller.Pipeline.ID, poller.Enabled, poller.Name); err != nil {
		log.Warning("UpdatePoller> Error :%s", err)
		return err
	}
	return nil
}

//LoadEnabledPollers load all RepositoryPoller
func LoadEnabledPollers(db database.Querier) ([]sdk.RepositoryPoller, error) {
	query := `
        SELECT application_id, pipeline_id, name, enabled, date_creation
        FROM poller
        WHERE enabled = true
    `
	return loadPollersByQUery(db, query)
}

//LoadEnabledPollersByProject load all RepositoryPoller for a project
func LoadEnabledPollersByProject(db database.Querier, projKey string) ([]sdk.RepositoryPoller, error) {
	query := `
        SELECT poller.application_id, poller.pipeline_id, poller.name, poller.enabled, poller.date_creation
        FROM poller, application, project
        WHERE poller.application_id = application.id
		AND application.project_id = project.id
		and project.projectkey = $1
		AND enabled = true
    `
	return loadPollersByQUery(db, query, projKey)
}

//LoadPollersByApplication loads all pollers for an application
func LoadPollersByApplication(db database.Querier, applicationID int64) ([]sdk.RepositoryPoller, error) {
	query := `
        SELECT application_id, pipeline_id, name, enabled, date_creation
        FROM poller
        WHERE application_id = $1
    `
	return loadPollersByQUery(db, query, applicationID)
}

//LoadPollerByApplicationAndPipeline loads all pollers for an application/pipeline
func LoadPollerByApplicationAndPipeline(db database.Querier, applicationID, pipelineID int64) (*sdk.RepositoryPoller, error) {
	query := `
        SELECT application_id, pipeline_id, name, enabled, date_creation
        FROM poller
        WHERE application_id = $1
		AND pipeline_id = $2
    `
	res, err := loadPollersByQUery(db, query, applicationID, pipelineID)
	if err != nil {
		log.Warning("LoadPollerByApplicationAndPipeline> Error :%s", err)
		return nil, err
	}
	if len(res) == 0 {
		return nil, sdk.ErrNotFound
	}
	return &res[0], nil
}

func loadPollersByQUery(db database.Querier, query string, args ...interface{}) ([]sdk.RepositoryPoller, error) {
	pollers := []sdk.RepositoryPoller{}
	rows, err := db.Query(query, args...)
	if err != nil {
		log.Warning("loadPollersByQuery> error querying poller : %s", err)
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var applicationID, pipelineID int64
		poller := sdk.RepositoryPoller{}
		if err := rows.Scan(&applicationID, &pipelineID, &poller.Name, &poller.Enabled, &poller.DateCreation); err != nil {
			log.Warning("loadPollersByQuery> error scanning poller : %s", err)
			return nil, err
		}
		app, err := application.LoadApplicationByID(db, applicationID)
		if err != nil {
			log.Warning("loadPollersByQuery> error loading application %d : %s", applicationID, err)
			return nil, err
		}
		pip, err := pipeline.LoadPipelineByID(db, pipelineID)
		if err != nil {
			log.Warning("loadPollersByQuery> error loading pipeline %d : %s", pipelineID, err)
			return nil, err
		}
		poller.Application = *app
		poller.Pipeline = *pip
		pollers = append(pollers, poller)
	}
	return pollers, nil
}
