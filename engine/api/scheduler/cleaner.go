package scheduler

import (
	"fmt"
	"time"

	"github.com/ovh/cds/engine/api/database"
	"github.com/ovh/cds/engine/log"
	"github.com/ovh/cds/sdk"
)

//Cleaner is the cleaner main goroutine
func Cleaner(delay time.Duration) {
	for {
		time.Sleep(10 * time.Minute)
		CleanerRun(delay)
	}
}

//CleanerRun is the core function of the cleaner goroutine
func CleanerRun(delay time.Duration) ([]sdk.PipelineSchedulerExecution, error) {
	_db := database.DB()
	if _db == nil {
		return nil, fmt.Errorf("Database is unavailable")
	}
	db := database.DBMap(_db)
	tx, err := db.Begin()
	if err != nil {
		log.Warning("CleanerRun> Unable to start a transaction : %s", err)
		return nil, err
	}
	defer tx.Rollback()

	//Starting with exclusive lock on the table
	if err := LockPipelineExecutions(tx); err != nil {
		return nil, err
	}

	log.Debug("CleanerRun> Deleting old executions...")

	t := time.Now().Add(delay * -1)
	exs, err := LoadPastExecutions(tx, t)
	if err != nil {
		return nil, err
	}

	for i := range exs {
		if err := DeleteExecution(tx, &exs[i]); err != nil {
			log.Critical("CleanerRun> Unable to delete execution %d", exs[i].ID)
		}
	}

	if err := tx.Commit(); err != nil {
		log.Warning("CleanerRun> Unable to commit a transaction : %s", err)
		return nil, err
	}

	return exs, nil
}
