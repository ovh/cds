package poller

import (
	"time"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/log"
	"github.com/ovh/cds/sdk"
)

//Cleaner is the cleaner main goroutine
func Cleaner(DBFunc func() *gorp.DbMap, nbToKeep int) {
	defer log.Critical("poller.Cleaner> has been exited !")
	for {
		time.Sleep(30 * time.Minute)
		_, err := CleanerRun(DBFunc(), nbToKeep)
		if err != nil {
			log.Warning("poller.Cleaner> Error : %s", err)
			continue
		}
	}
}

//CleanerRun is the core function of the cleaner goroutine
func CleanerRun(db *gorp.DbMap, nbToKeep int) ([]sdk.RepositoryPollerExecution, error) {
	log.Debug("poller.CleanerRun> Deleting old executions...")
	tx, err := db.Begin()
	if err != nil {
		log.Warning("poller.CleanerRun> Unable to start a transaction : %s", err)
		return nil, err
	}
	defer tx.Rollback()

	//Load pollers
	ps, err := LoadAll(tx)
	if err != nil {
		log.Warning("poller.CleanerRun> Unable to load pipeline pollers : %s", err)
		return nil, err
	}

	deleted := []sdk.RepositoryPollerExecution{}
	for _, s := range ps {
		exs, err := LoadPastExecutions(tx, s.ApplicationID, s.PipelineID)
		if err != nil {
			log.Warning("poller.CleanerRun> Unable to load pipeline pollers execution : %s", err)
			return nil, err
		}

		nbToDelete := len(exs) - nbToKeep
		nbDeleted := 0
		for i := range exs {
			if nbDeleted > nbToDelete {
				break
			}
			if err := DeleteExecution(tx, &exs[i]); err != nil {
				log.Critical("poller.CleanerRun> Unable to delete execution %d : %s", exs[i].ID, err)
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
