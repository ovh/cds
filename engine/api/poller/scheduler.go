package poller

import (
	"fmt"
	"time"

	"github.com/ovh/cds/engine/api/database"
	"github.com/ovh/cds/engine/log"
	"github.com/ovh/cds/sdk"
)

var (
	pollerStatus = "Not Running"
	pollingDelay = 60 * time.Second
)

//Scheduler is the goroutine which compute date of next execution for pipeline scheduler
func Scheduler() {
	for {
		time.Sleep(2 * time.Second)
		_, status, err := SchedulerRun()

		if err != nil {
			log.Critical("%s: %s", status, err)
		}
		pollerStatus = status
	}
}

//SchedulerRun is the core function of Scheduler goroutine
func SchedulerRun() ([]sdk.RepositoryPollerExecution, string, error) {
	_db := database.DB()
	if _db == nil {
		return nil, "Database is unavailable", fmt.Errorf("datase.DB failed")
	}
	db := database.DBMap(_db)
	tx, err := db.Begin()
	if err != nil {
		return nil, "poller.Scheduler.Run> Unable to start a transaction", err
	}
	defer tx.Rollback()

	//Starting with exclusive lock on the table
	if err := LockPollerExecutions(tx); err != nil {
		return nil, "OK", nil
	}

	//Load unscheduled pipelines
	ps, err := LoadUnscheduledPollers(tx)
	if err != nil {
		return nil, "poller.Scheduler.Run> Unable to load unscheduled pollers : %s", err
	}

	execs := []sdk.RepositoryPollerExecution{}
	for i := range ps {
		p := &ps[i]

		//Skip disabled scheduler
		if !p.Enabled {
			continue
		}

		//Skip if there is a pending execution
		if next, _ := LoadNextExecution(tx, p.ApplicationID, p.PipelineID); next != nil {
			log.Debug("poller.Scheduler.Run> Poller has already a pending execution")
			continue
		}

		e := sdk.RepositoryPollerExecution{
			ApplicationID:        p.ApplicationID,
			PipelineID:           p.PipelineID,
			ExecutionPlannedDate: time.Now().Add(pollingDelay),
		}

		//Insert execution
		if err := InsertExecution(tx, &e); err != nil {
			log.Warning("poller.Scheduler.Run> Unable to insert polling executions : %s", err)
			return execs, "poller.Scheduler.Run> Unable to insert polling executions : %s", err
		}

		execs = append(execs, e)
	}

	if err := tx.Commit(); err != nil {
		return nil, "poller.Scheduler.Run> Unable to commit transaction : %s", err
	}

	return execs, "OK", nil
}
