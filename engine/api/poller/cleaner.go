package poller

import (
	"fmt"
	"time"

	"github.com/ovh/cds/engine/api/database"
	"github.com/ovh/cds/engine/log"
	"github.com/ovh/cds/sdk"
)

//Cleaner is the cleaner main goroutine
func Cleaner(nbToKeep int) {
	for {
		CleanerRun(nbToKeep)
		time.Sleep(10 * time.Minute)
	}
}

//CleanerRun is the core function of the cleaner goroutine
func CleanerRun(nbToKeep int) ([]sdk.RepositoryPollerExecution, error) {
	log.Debug("poller.CleanerRun> Deleting old executions...")
	_db := database.DB()
	if _db == nil {
		return nil, fmt.Errorf("Database is unavailable")
	}
	db := database.DBMap(_db)
	tx, err := db.Begin()
	if err != nil {
		log.Warning("poller.CleanerRun> Unable to start a transaction : %s", err)
		return nil, err
	}
	defer tx.Rollback()

	//Starting with exclusive lock on the table
	if err := LockPollerExecutions(tx); err != nil {
		log.Debug("poller.CleanerRun> Unable to take lock : %s", err)
		return nil, err
	}

	//Load schedulers
	ps, err := LoadAll(tx)
	if err != nil {
		log.Warning("poller.CleanerRun> Unable to load pipeline schedulers : %s", err)
		return nil, err
	}

	deleted := []sdk.RepositoryPollerExecution{}
	for _, s := range ps {
		exs, err := LoadPastExecutions(tx, s.ApplicationID, s.PipelineID)
		if err != nil {
			log.Warning("poller.CleanerRun> Unable to load pipeline schedulers execution : %s", err)
			return nil, err
		}

		nbToDelete := len(exs) - nbToKeep
		nbDeleted := 0
		for i := range exs {
			if nbDeleted > nbToDelete {
				break
			}
			if err := DeleteExecution(tx, &exs[i]); err != nil {
				log.Critical("poller.CleanerRun> Unable to delete execution %d", exs[i].ID)
			}
			nbDeleted++
			deleted = append(deleted, exs[i])
		}
	}

	if err := tx.Commit(); err != nil {
		log.Warning("poller.CleanerRun> Unable to commit a transaction : %s", err)
		return nil, err
	}

	return deleted, nil
}
