package poller

import (
	"context"
	"time"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

var (
	pollerStatus = "Not Running"
	pollingDelay = 60 * time.Second
)

//Scheduler is the goroutine which compute date of next execution for pipeline scheduler
func Scheduler(c context.Context, DBFunc func(context.Context) *gorp.DbMap) {
	tick := time.NewTicker(10 * time.Second).C
	for {
		select {
		case <-c.Done():
			if c.Err() != nil {
				log.Error("Exiting poller.Scheduler: %v", c.Err())
				return
			}
		case <-tick:
			_, status, err := SchedulerRun(DBFunc(c))
			if err != nil {
				log.Error("poller.Scheduler> %s: %s", status, err)
			}
			pollerStatus = status
		}
	}
}

//SchedulerRun is the core function of Scheduler goroutine
func SchedulerRun(db *gorp.DbMap) ([]sdk.RepositoryPollerExecution, string, error) {
	tx, err := db.Begin()
	if err != nil {
		return nil, "poller.Scheduler.Run> Unable to start a transaction", err
	}
	defer tx.Rollback()

	//Load unscheduled pipelines
	ps, err := LoadUnscheduledPollers(tx)
	if err != nil {
		return nil, "poller.Scheduler.Run> Unable to load unscheduled pollers : %s", err
	}

	execs := []sdk.RepositoryPollerExecution{}
	for i := range ps {
		p := &ps[i]
		log.Debug("poller.Scheduler.Run> Checking poller %s/%s", p.Application.Name, p.Pipeline.Name)

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
