package scheduler

import (
	"context"
	"time"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

//Cleaner is the cleaner main goroutine
func Cleaner(c context.Context, DBFunc func() *gorp.DbMap, nbToKeep int) {
	tick := time.NewTicker(10 * time.Minute).C
	for {
		select {
		case <-c.Done():
			if c.Err() != nil {
				log.Error("Exiting scheduler.Cleaner: %v", c.Err())
				return
			}
		case <-tick:
			if _, err := CleanerRun(DBFunc(), nbToKeep); err != nil {
				log.Warning("cleander.Cleaner> Error : %s", err)
				continue
			}
		}
	}
}

//CleanerRun is the core function of the cleaner goroutine
func CleanerRun(db *gorp.DbMap, nbToKeep int) ([]sdk.PipelineSchedulerExecution, error) {
	log.Debug("CleanerRun> Deleting old executions...")

	tx, errb := db.Begin()
	if errb != nil {
		return nil, sdk.WrapError(errb, "CleanerRun> Unable to start a transaction")
	}
	defer tx.Rollback()

	//Starting with exclusive lock on the table
	if err := LockPipelineExecutions(tx); err != nil {
		log.Debug("CleanerRun> Unable to take lock : %s", err)
		return nil, nil
	}

	//Load schedulers
	ps, err := LoadAll(tx)
	if err != nil {
		return nil, sdk.WrapError(err, "CleanerRun> Unable to load pipeline schedulers")
	}

	deleted := []sdk.PipelineSchedulerExecution{}
	for _, s := range ps {
		exs, err := LoadPastExecutions(tx, s.ID)
		if err != nil {
			return nil, sdk.WrapError(err, "CleanerRun> Unable to load pipeline schedulers execution")
		}

		nbToDelete := len(exs) - nbToKeep
		nbDeleted := 0
		for i := range exs {
			if nbDeleted > nbToDelete {
				break
			}
			if err := DeleteExecution(tx, &exs[i]); err != nil {
				log.Error("CleanerRun> Unable to delete execution %d err:%s", exs[i].ID, err)
			}
			nbDeleted++
			deleted = append(deleted, exs[i])
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, sdk.WrapError(err, "CleanerRun> Unable to commit a transaction")
	}

	return deleted, nil
}
