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

	if err := db.Insert(&dbPoller); err != nil {
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
	if _, err := db.Delete(&dbPoller); err != nil {
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

	pollers, err := unwrapPollers(db, dbPollers)
	if err != nil {
		return nil, err
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

	pollers, err := unwrapPollers(db, dbPollers)
	if err != nil {
		return nil, err
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

	pollers, err := unwrapPollers(db, dbPollers)
	if err != nil {
		return nil, err
	}

	return pollers, nil
}

//LoadPollerByApplicationAndPipeline loads the poller for an application/pipeline
func LoadPollerByApplicationAndPipeline(db gorp.SqlExecutor, applicationID, pipelineID int64) (*sdk.RepositoryPoller, error) {
	query := `
        SELECT application_id, pipeline_id, name, enabled, date_creation
        FROM poller
        WHERE application_id = $1
		AND pipeline_id = $2
    `
	dbPoller := database.RepositoryPoller{}
	if err := db.SelectOne(&dbPoller, query, applicationID, pipelineID); err != nil {
		return nil, err
	}

	p, err := unwrapPoller(db, dbPoller)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func postGet(db gorp.SqlExecutor, p *sdk.RepositoryPoller) error {
	app, err := application.LoadApplicationByID(db, p.ApplicationID)
	if err != nil {
		log.Warning("postGet> error loading application %d : %s", p.ApplicationID, err)
		return err
	}
	pip, err := pipeline.LoadPipelineByID(db, p.PipelineID, true)
	if err != nil {
		log.Warning("postGet> error loading pipeline %d : %s", p.PipelineID, err)
		return err
	}
	p.Application = *app
	p.Pipeline = *pip
	return nil
}

func unwrapPollers(db gorp.SqlExecutor, dbPollers []database.RepositoryPoller) ([]sdk.RepositoryPoller, error) {
	pollers := make([]sdk.RepositoryPoller, len(dbPollers))
	for i, p := range dbPollers {
		pl, err := unwrapPoller(db, p)
		if err != nil {
			return nil, err
		}
		pollers[i] = pl
	}
	return pollers, nil
}

func unwrapPoller(db gorp.SqlExecutor, dbPoller database.RepositoryPoller) (sdk.RepositoryPoller, error) {
	p := sdk.RepositoryPoller(dbPoller)
	return p, postGet(db, &p)
}
