package scheduler

import (
	"context"
	"time"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/environment"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/queue"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

//Executer is the goroutine which run the pipelines
func Executer(c context.Context, DBFunc func() *gorp.DbMap, store cache.Store) {
	tick := time.NewTicker(5 * time.Second).C
	for {
		select {
		case <-c.Done():
			if c.Err() != nil {
				log.Error("Exiting scheduler.Executer: %v", c.Err())
				return
			}
		case <-tick:
			ExecuterRun(DBFunc, store)
		}
	}
}

//ExecuterRun is the core function of Executer goroutine
func ExecuterRun(DBFunc func() *gorp.DbMap, store cache.Store) ([]sdk.PipelineSchedulerExecution, error) {
	db := DBFunc()
	if db == nil {
		return nil, sdk.WrapError(sdk.ErrServiceUnavailable, "ExecuterRun> Unable to load pending execution")
	}
	tx, errb := db.Begin()
	if errb != nil {
		log.Warning("ExecuterRun> %s", errb)
		return nil, errb
	}
	defer tx.Rollback()

	//Starting with exclusive lock on the table
	if err := LockPipelineExecutions(tx); err != nil {
		return nil, err
	}

	//Load pending executions
	exs, err := LoadPendingExecutions(tx)
	if err != nil {
		return nil, sdk.WrapError(err, "ExecuterRun> Unable to load pending execution")
	}

	var pbs []sdk.PipelineBuild
	//Process all
	for i := range exs {
		pb, err := executerProcess(DBFunc, store, tx, &exs[i])
		if err != nil {
			log.Error("ExecuterRun> Unable to process %+v : %s", exs[i], err)
		}
		if pb != nil {
			pbs = append(pbs, *pb)
		}
	}

	//Commit
	if err := tx.Commit(); err != nil {
		log.Warning("ExecuterRun> %s", err)
		return nil, err
	}

	for _, pb := range pbs {
		proj, errproj := project.Load(db, store, pb.Application.ProjectKey, nil)
		if errproj != nil {
			log.Warning("ExecuterRun> Unable to load project: %s", errproj)
		}

		app, errapp := application.LoadByID(db, store, pb.Application.ID, nil, application.LoadOptions.WithRepositoryManager)
		if errapp != nil {
			log.Warning("ExecuterRun> Unable to load app: %s", errapp)
		}

		if _, err := pipeline.UpdatePipelineBuildCommits(db, proj, &pb.Pipeline, app, &pb.Environment, &pb); err != nil {
			log.Warning("ExecuterRun> Unable to update pipeline build commits : %s", err)
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
	app, err := application.LoadByID(db, store, s.ApplicationID, nil, application.LoadOptions.WithRepositoryManager, application.LoadOptions.WithVariablesWithClearPassword)
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
