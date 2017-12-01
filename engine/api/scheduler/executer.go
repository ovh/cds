package scheduler

import (
	"time"

	"github.com/go-gorp/gorp"
	"github.com/lib/pq"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/environment"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/queue"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

//ExecuterRun is the core function of Executer goroutine
func ExecuterRun(DBFunc func() *gorp.DbMap, store cache.Store) ([]sdk.PipelineSchedulerExecution, error) {
	db := DBFunc()
	if db == nil {
		return nil, sdk.WrapError(sdk.ErrServiceUnavailable, "ExecuterRun> Unable to load pending execution")
	}

	//Load pending executions
	exs, err := LoadPendingExecutions(db)
	if err != nil {
		return nil, sdk.WrapError(err, "ExecuterRun> Unable to load pending execution")
	}

	//Process all
	//We are opening a new tx for each execution with a lock on each execution: one by one
	for i := range exs {
		var gorpEx = &PipelineSchedulerExecution{}
		tx, errb := db.Begin()
		if errb != nil {
			log.Warning("ExecuterRun> %s", errb)
			return nil, errb
		}

		s, errlock := loadAndLockPipelineScheduler(tx, exs[i].PipelineSchedulerID)
		if errlock != nil {
			log.Error("ExecuterRun> Unable to load to pipeline scheduler %d %s", exs[i].PipelineSchedulerID, errlock)
			_ = tx.Rollback()
			continue
		}
		if s == nil {
			_ = tx.Rollback()
			continue
		}

		query := "SELECT * FROM pipeline_scheduler_execution WHERE id = $1 and executed = 'false' FOR UPDATE NOWAIT"
		if err := tx.SelectOne(gorpEx, query, exs[i].ID); err != nil {
			pqerr, ok := err.(*pq.Error)
			// Cannot get lock (FOR UPDATE NOWAIT), someone else is on it
			if ok && pqerr.Code != "55P03" {
				log.Error("ExecuterRun> Unable to get lock on the pipeline_scheduler_execution %d: %v", exs[i].ID, err)
			}
			//Rollback
			if err := tx.Rollback(); err != nil {
				log.Warning("ExecuterRun> %s", err)
				return nil, err
			}
			continue
		}

		ex := sdk.PipelineSchedulerExecution(*gorpEx)
		if _, errProcess := executerProcess(DBFunc, store, tx, &ex); errProcess != nil {
			log.Error("ExecuterRun> Unable to process %+v : %s", ex, errProcess)
			_ = tx.Rollback()
			continue
		}

		nextExec, errNext := Next(tx, s)
		if errNext != nil {
			log.Error("ExecuterRun> Unable to compute next execution %+v : %s", ex, errNext)
			_ = tx.Rollback()
			continue
		}
		if err := InsertExecution(tx, nextExec); err != nil {
			log.Error("ExecuterRun> Unable to compute next execution %+v : %s", nextExec, errNext)
			_ = tx.Rollback()
			continue
		}

		//Commit
		if err := tx.Commit(); err != nil {
			log.Warning("ExecuterRun> %s", err)
			return nil, err
		}
	}

	return exs, nil
}

func executerProcess(DBFunc func() *gorp.DbMap, store cache.Store, db gorp.SqlExecutor, e *sdk.PipelineSchedulerExecution) (*sdk.PipelineBuild, error) {
	//Load the scheduler
	s, err := Load(db, e.PipelineSchedulerID)
	if err != nil {
		return nil, err
	}

	//Load application
	app, err := application.LoadByID(db, store, s.ApplicationID, nil, application.LoadOptions.WithVariablesWithClearPassword)
	if err != nil {
		return nil, err
	}

	//Load pipeline
	pip, err := pipeline.LoadPipelineByID(db, s.PipelineID, true)
	if err != nil {
		return nil, err
	}

	//Load environnement
	env, err := environment.LoadEnvironmentByID(db, s.EnvironmentID)
	if err != nil {
		return nil, err
	}

	//Create a new pipeline build
	pb, err := queue.RunPipeline(DBFunc, store, db, app.ProjectKey, app, pip.Name, env.Name, s.Args, -1, sdk.PipelineBuildTrigger{
		ManualTrigger:    false,
		ScheduledTrigger: true,
	}, nil)

	if err != nil {
		return nil, err
	}

	//References pipeline build version in execution
	t := time.Now()
	e.ExecutionDate = &t
	e.PipelineBuildVersion = pb.Version
	e.Executed = true

	//Update execution in database
	if err := UpdateExecution(db, e); err != nil {
		return nil, err
	}

	return pb, nil
}
