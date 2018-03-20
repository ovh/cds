package scheduler

import (
	"fmt"
	"time"

	"github.com/go-gorp/gorp"
	"github.com/gorhill/cronexpr"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

var schedulerStatus = "Not Running"

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

		s, errlock := loadAndLockPipelineScheduler(tx, ps[i].ID)
		if errlock != nil {
			log.Error("Run> Unable to load to pipeline scheduler %s", errlock)
			_ = tx.Rollback()
			continue
		}
		if s == nil {
			_ = tx.Rollback()
			continue
		}

		//Reload the last execution
		ex, errex := LoadLastExecution(tx, s.ID)
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

		//Skip disabled scheduler
		if s.Disabled {
			_ = tx.Rollback()
			continue
		}

		//Compute a new execution
		e, errn := Next(tx, s)
		if errn != nil {
			//Nothing to compute
			log.Error("Run> Error while compute next execution: %s", errn)
			_ = tx.Rollback()
			continue
		}
		//Insert it
		if err := InsertExecution(tx, e); err != nil {
			log.Error("Run> Error while insert Execution: %s", err)
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
		// Next from now + 10 seconds, to potentially avoid desyncronisation time with many instances of API
		ExecutionPlannedDate: cronExpr.Next(t.Add(10 * time.Second)),
		PipelineSchedulerID:  s.ID,
		Executed:             false,
	}
	return e, nil
}

// Status returns Event status
func Status() sdk.MonitoringStatusLine {
	if schedulerStatus != "OK" {
		return sdk.MonitoringStatusLine{Component: "Scheduler", Value: "⚠ " + schedulerStatus, Status: sdk.MonitoringStatusWarn}
	}
	return sdk.MonitoringStatusLine{Component: "Scheduler", Value: schedulerStatus, Status: sdk.MonitoringStatusOK}
}
