package poller

import (
	"time"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/sdk"
)

//Insert insert a new poller in DB
func Insert(db gorp.SqlExecutor, poller *sdk.RepositoryPoller) error {
	poller.DateCreation = time.Now()
	dbPoller := RepositoryPoller(*poller)

	if err := db.Insert(&dbPoller); err != nil {
		return sdk.WrapError(err, "InsertPoller> Error")
	}

	newPoller := sdk.RepositoryPoller(dbPoller)
	*poller = newPoller

	return nil
}

//Delete delete a poller from DBpoller
func Delete(db gorp.SqlExecutor, poller *sdk.RepositoryPoller) error {
	dbPoller := RepositoryPoller(*poller)
	if _, err := db.Delete(&dbPoller); err != nil {
		return sdk.WrapError(err, "DeletePoller> Error")
	}
	return nil
}

// DeleteAll  Delete all the poller of the given application
func DeleteAll(db gorp.SqlExecutor, appID int64) error {
	if err := DeleteExecutionByApplicationID(db, appID); err != nil {
		return sdk.WrapError(err, "DeleteAll")
	}

	query := "DELETE FROM poller WHERE application_id = $1"
	if _, err := db.Exec(query, appID); err != nil {
		return sdk.WrapError(err, "DeleteAllPoller> Error")
	}
	return nil
}

//Update update the poller
func Update(db gorp.SqlExecutor, poller *sdk.RepositoryPoller) error {
	query := `
        UPDATE  poller
        SET enabled = $3, name = $4
        WHERE application_id = $1
        AND pipeline_id  = $2
    `
	if _, err := db.Exec(query, poller.Application.ID, poller.Pipeline.ID, poller.Enabled, poller.Name); err != nil {
		return sdk.WrapError(err, "UpdatePoller> Error")
	}
	return nil
}

// LoadAll retrieves all poller from database
func LoadAll(db gorp.SqlExecutor) ([]sdk.RepositoryPoller, error) {
	dbPollers := []RepositoryPoller{}
	if _, err := db.Select(&dbPollers, "SELECT * FROM poller"); err != nil {
		return nil, err
	}

	pollers, err := unwrapPollers(db, dbPollers)
	if err != nil {
		return nil, err
	}

	return pollers, nil
}

//LoadByApplication loads all pollers for an application
func LoadByApplication(db gorp.SqlExecutor, applicationID int64) ([]sdk.RepositoryPoller, error) {
	query := `
        SELECT application_id, pipeline_id, name, enabled, date_creation
        FROM poller
        WHERE application_id = $1
    `
	dbPollers := []RepositoryPoller{}
	if _, err := db.Select(&dbPollers, query, applicationID); err != nil {
		return nil, err
	}

	pollers, err := unwrapPollers(db, dbPollers)
	if err != nil {
		return nil, err
	}

	return pollers, nil
}

//LoadByApplicationAndPipeline loads the poller for an application/pipeline
func LoadByApplicationAndPipeline(db gorp.SqlExecutor, applicationID, pipelineID int64) (*sdk.RepositoryPoller, error) {
	query := `
        SELECT application_id, pipeline_id, name, enabled, date_creation
        FROM poller
        WHERE application_id = $1
		AND pipeline_id = $2
    `
	dbPoller := RepositoryPoller{}
	if err := db.SelectOne(&dbPoller, query, applicationID, pipelineID); err != nil {
		return nil, err
	}

	p := sdk.RepositoryPoller(dbPoller)

	return &p, nil
}

func unwrapPollers(db gorp.SqlExecutor, dbPollers []RepositoryPoller) ([]sdk.RepositoryPoller, error) {
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

func unwrapPoller(db gorp.SqlExecutor, dbPoller RepositoryPoller) (p sdk.RepositoryPoller, err error) {
	if err = dbPoller.PostGet(db); err != nil {
		return
	}
	p = sdk.RepositoryPoller(dbPoller)
	return
}
