package poller

import (
	"time"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/database"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/log"
	"github.com/ovh/cds/sdk"
)

//InsertPoller insert or update a new poller in DB
func InsertPoller(db gorp.SqlExecutor, poller *sdk.RepositoryPoller) error {
	poller.DateCreation = time.Now()
	dbPoller := database.RepositoryPoller(*poller)

	if err := db.Insert(dbPoller); err != nil {
		log.Warning("InsertPoller> Error :%s", err)
		return err
	}

	newPoller := sdk.RepositoryPoller(dbPoller)
	*poller = newPoller

	return nil
}

//DeletePoller delete a poller from DB
func DeletePoller(db gorp.SqlExecutor, poller *sdk.RepositoryPoller) error {
	dbPoller := database.RepositoryPoller(*poller)
	if _, err := db.Delete(dbPoller); err != nil {
		log.Warning("DeletePoller> Error :%s", err)
		return err
	}
	return nil
}

// DeleteAllPollers  Delete all the poller of the given application
func DeleteAllPollers(db gorp.SqlExecutor, appID int64) error {
	query := "DELETE FROM poller WHERE application_id = $1"
	if _, err := db.Exec(query, appID); err != nil {
		log.Warning("DeleteAllPoller> Error :%s", err)
		return err
	}
	return nil
}

//UpdatePoller update the poller
func UpdatePoller(db gorp.SqlExecutor, poller *sdk.RepositoryPoller) error {
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
func LoadEnabledPollers(db gorp.SqlExecutor) ([]sdk.RepositoryPoller, error) {
	dbPollers := []database.RepositoryPoller{}
	if _, err := db.Select(&dbPollers, "SELECT * FROM poller WHERE enabled = true"); err != nil {
		return nil, err
	}

	pollers := make([]sdk.RepositoryPoller, len(dbPollers))
	for i, p := range dbPollers {
		pollers[i] = sdk.RepositoryPoller(p)
	}

	return pollers, nil
}

//LoadEnabledPollersByProject load all RepositoryPoller for a project
func LoadEnabledPollersByProject(db gorp.SqlExecutor, projKey string) ([]sdk.RepositoryPoller, error) {
	query := `
        SELECT poller.application_id, poller.pipeline_id, poller.name, poller.enabled, poller.date_creation
        FROM poller, application, project
        WHERE poller.application_id = application.id
		AND application.project_id = project.id
		and project.projectkey = $1
		AND enabled = true
    `
	dbPollers := []database.RepositoryPoller{}
	if _, err := db.Select(&dbPollers, query, projKey); err != nil {
		return nil, err
	}

	pollers := make([]sdk.RepositoryPoller, len(dbPollers))
	for i, p := range dbPollers {
		pollers[i] = sdk.RepositoryPoller(p)
	}

	return pollers, nil
}

//LoadPollersByApplication loads all pollers for an application
func LoadPollersByApplication(db gorp.SqlExecutor, applicationID int64) ([]sdk.RepositoryPoller, error) {
	query := `
        SELECT application_id, pipeline_id, name, enabled, date_creation
        FROM poller
        WHERE application_id = $1
    `
	dbPollers := []database.RepositoryPoller{}
	if _, err := db.Select(&dbPollers, query, applicationID); err != nil {
		return nil, err
	}

	pollers := make([]sdk.RepositoryPoller, len(dbPollers))
	for i, p := range dbPollers {
		pollers[i] = sdk.RepositoryPoller(p)
	}

	return pollers, nil
}

//LoadPollerByApplicationAndPipeline loads all pollers for an application/pipeline
func LoadPollerByApplicationAndPipeline(db gorp.SqlExecutor, applicationID, pipelineID int64) (*sdk.RepositoryPoller, error) {
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

func loadPollersByQUery(db gorp.SqlExecutor, query string, args ...interface{}) ([]sdk.RepositoryPoller, error) {
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
		pip, err := pipeline.LoadPipelineByID(db, pipelineID, true)
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
