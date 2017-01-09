package scheduler

import (
	"fmt"
	"time"

	"github.com/go-gorp/gorp"
	"github.com/gorhill/cronexpr"

	"github.com/ovh/cds/engine/api/database"
	"github.com/ovh/cds/engine/log"
	"github.com/ovh/cds/sdk"
)

func Scheduler() {
	for {
		time.Sleep(1 * time.Second)
		SchedulerRun()
	}
}

func SchedulerRun() ([]sdk.PipelineSchedulerExecution, error) {
	_db := database.DB()
	if _db == nil {
		return nil, fmt.Errorf("Database is unavailable")
	}
	db := database.DBMap(_db)
	tx, err := db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	//Starting with exclusive lock on the table
	if err := LockPipelineExecutions(tx); err != nil {
		return nil, err
	}

	//Load unscheduled pipelines
	ps, err := LoadUnscheduledPipelines(tx)
	if err != nil {
		return nil, err
	}

	execs := []sdk.PipelineSchedulerExecution{}

	for i := range ps {
		//Skip disabled scheduler
		if ps[i].Disabled {
			continue
		}

		//Compute a new execution
		e, err := Next(tx, &ps[i])
		if err != nil {
			//Nothing to compute
			continue
		}
		//Insert it
		if err := InsertExecution(tx, e); err != nil {
			return nil, err
		}
		execs = append(execs, *e)
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return execs, nil
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

	if exec == nil {
		t := time.Now()
		exec = &sdk.PipelineSchedulerExecution{
			Executed:      true,
			ExecutionDate: &t,
		}
	}

	if !exec.Executed {
		return nil, fmt.Errorf("Last execution %d not ran", s.ID)
	}
	nextTime := cronExpr.Next(*exec.ExecutionDate)
	e := &sdk.PipelineSchedulerExecution{
		ExecutionPlannedDate: nextTime,
		PipelineSchedulerID:  s.ID,
		Executed:             false,
	}
	return e, nil
}
