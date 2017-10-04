package scheduler

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/go-gorp/gorp"
	"github.com/gorhill/cronexpr"
	"github.com/lib/pq"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

var schedulerStatus = "Not Running"

//Scheduler is the goroutine which compute date of next execution for pipeline scheduler
func Scheduler(c context.Context, DBFunc func() *gorp.DbMap) {
	rand.Seed(time.Now().Unix())
	tick := time.NewTicker(2 * time.Second).C
	for {
		select {
		case <-c.Done():
			if c.Err() != nil {
				log.Error("Exiting scheduler.Scheduler: %v", c.Err())
				return
			}
		case <-tick:
			time.Sleep(time.Duration(rand.Intn(500)) * time.Millisecond)
			_, status, err := Run(DBFunc())

			if err != nil {
				log.Error("%s: %s", status, err)
			}
			schedulerStatus = status
		}
	}
}

//Run is the core function of Scheduler goroutine
func Run(db *gorp.DbMap) ([]sdk.PipelineSchedulerExecution, string, error) {
	//Load unscheduled pipelines
	ps, errl := LoadUnscheduledPipelines(db)
	if errl != nil {
		return nil, "Run> Unable to load unscheduled pipelines : %s", errl
	}

	execs := []sdk.PipelineSchedulerExecution{}

	for i := range ps {
		tx, errb := db.Begin()
		if errb != nil {
			return nil, "Run> Unable to start a transaction", errb
		}

		query := `
			SELECT 	pipeline_scheduler.* 
			FROM 	pipeline_scheduler 
			WHERE   pipeline_scheduler.id = $1
			FOR UPDATE NOWAIT`

		var gorpPS = &PipelineScheduler{}
		if err := tx.SelectOne(gorpPS, query, ps[i].ID); err != nil {
			if pqerr, ok := err.(*pq.Error); ok && pqerr.Code != "55P03" {
				log.Error("Run> Unable to lock to pipeline_scheduler %s", err)
			}
			_ = tx.Rollback()
			continue
		}

		//Reload the last execution
		ex, errex := LoadLastExecution(tx, gorpPS.ID)
		if errex != nil {
			log.Error("Run> Unable to load to pipeline scheduler execution %s", errex)
			_ = tx.Rollback()
			continue
		}

		//If the last execution has not been executed, it means that the scheduler is already scheduled
		if ex != nil && !ex.Executed {
			_ = tx.Rollback()
			continue
		}

		s := sdk.PipelineScheduler(*gorpPS)
		//Skip disabled scheduler
		if s.Disabled {
			_ = tx.Rollback()
			continue
		}

		//Compute a new execution
		e, errn := Next(tx, &s)
		if errn != nil {
			//Nothing to compute
			_ = tx.Rollback()
			continue
		}
		//Insert it
		if err := InsertExecution(tx, e); err != nil {
			_ = tx.Rollback()
			continue
		}
		execs = append(execs, *e)

		if err := tx.Commit(); err != nil {
			_ = tx.Rollback()
			return nil, "Run> Unable to commit a transaction : %s", err
		}
	}

	return execs, "OK", nil
}

//Next Compute the next PipelineSchedulerExecution
func Next(db gorp.SqlExecutor, s *sdk.PipelineScheduler) (*sdk.PipelineSchedulerExecution, error) {
	cronExpr, err := cronexpr.Parse(s.Crontab)
	if err != nil {
		log.Warning("scheduler.Next> Unable to parse cronexpr for ID %d : %s", s.ID, err)
		return nil, err
	}
	exec, err := LoadLastExecution(db, s.ID)
	if err != nil {
		return nil, nil
	}

	loc, err := time.LoadLocation(s.Timezone)
	if err != nil {
		return nil, err
	}

	t := time.Now().In(loc)
	if exec == nil {
		exec = &sdk.PipelineSchedulerExecution{
			Executed:      true,
			ExecutionDate: &t,
		}
	}

	if !exec.Executed {
		return nil, fmt.Errorf("Last execution %d not ran", s.ID)
	}
	//Don't take last execution date as reference: time.Now() is enough
	e := &sdk.PipelineSchedulerExecution{
		ExecutionPlannedDate: cronExpr.Next(t),
		PipelineSchedulerID:  s.ID,
		Executed:             false,
	}
	return e, nil
}

// Status returns Event status
func Status() string {
	if schedulerStatus != "OK" {
		return "⚠ " + schedulerStatus
	}
	return schedulerStatus
}
