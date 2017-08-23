package poller

import (
	"context"
	"time"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

//Cleaner is the cleaner main goroutine
func Cleaner(c context.Context, DBFunc func() *gorp.DbMap, nbToKeep int) {
	tick := time.NewTicker(30 * time.Minute).C
	for {
		select {
		case <-c.Done():
			if c.Err() != nil {
				log.Error("Exiting poller.Cleaner: %v", c.Err())
				return
			}
		case <-tick:
			if _, err := CleanerRun(DBFunc(), nbToKeep); err != nil {
				log.Warning("poller.Cleaner> Error : %s", err)
				continue
			}
		}
	}
}

//CleanerRun is the core function of the cleaner goroutine
func CleanerRun(db *gorp.DbMap, nbToKeep int) ([]sdk.RepositoryPollerExecution, error) {
	log.Debug("poller.CleanerRun> Deleting old executions...")
	tx, err := db.Begin()
	if err != nil {
		return nil, sdk.WrapError(err, "poller.CleanerRun> Unable to start a transaction")
	}
	defer tx.Rollback()

	//Load pollers
	ps, err := LoadAll(tx)
	if err != nil {
		return nil, sdk.WrapError(err, "poller.CleanerRun> Unable to load pipeline pollers")
	}

	deleted := []sdk.RepositoryPollerExecution{}
	for _, s := range ps {
		exs, err := LoadPastExecutions(tx, s.ApplicationID, s.PipelineID)
		if err != nil {
			return nil, sdk.WrapError(err, "poller.CleanerRun> Unable to load pipeline pollers execution")
		}

		nbToDelete := len(exs) - nbToKeep
		nbDeleted := 0
		for i := range exs {
			if nbDeleted > nbToDelete {
				break
			}
			if err := DeleteExecution(tx, &exs[i]); err != nil {
				log.Error("poller.CleanerRun> Unable to delete execution %d : %s", exs[i].ID, err)
			}
			nbDeleted++
			deleted = append(deleted, exs[i])
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, sdk.WrapError(err, "poller.CleanerRun> Unable to commit a transaction")
	}

	return deleted, nil
}
