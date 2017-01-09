package scheduler

import (
	"fmt"
	"time"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/database"
	"github.com/ovh/cds/engine/api/environment"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/queue"
	"github.com/ovh/cds/engine/log"
	"github.com/ovh/cds/sdk"
)

//Executer is the goroutine which run the pipelines
func Executer() {
	for {
		time.Sleep(5 * time.Second)
		ExecuterRun()
	}
}

//ExecuterRun is the core function of Executer goroutine
func ExecuterRun() ([]sdk.PipelineSchedulerExecution, error) {
	_db := database.DB()
	if _db == nil {
		return nil, fmt.Errorf("Database is unavailable")
	}
	db := database.DBMap(_db)
	tx, err := db.Begin()
	if err != nil {
		log.Warning("ExecuterRun> %s", err)
		return nil, err
	}
	defer tx.Rollback()

	//Starting with exclusive lock on the table
	if err := LockPipelineExecutions(tx); err != nil {
		return nil, err
	}

	//Load pending executions
	exs, err := LoadPendingExecutions(tx)
	if err != nil {
		log.Warning("ExecuterRun> %s", err)
		return nil, err
	}

	//Process all
	for i := range exs {
		if err := executerProcess(tx, &exs[i]); err != nil {
			log.Critical("ExecuterRun> Unable to process %v : %s", exs[i], err)
		}
	}

	//Commit
	if err := tx.Commit(); err != nil {
		log.Warning("ExecuterRun> %s", err)
		return nil, err
	}

	return exs, nil
}

func executerProcess(db gorp.SqlExecutor, e *sdk.PipelineSchedulerExecution) error {
	//Load the scheduler
	s, err := Load(db, e.PipelineSchedulerID)
	if err != nil {
		return err
	}

	//Load application
	app, err := application.LoadApplicationByID(db, s.ApplicationID)
	if err != nil {
		return err
	}

	//Load pipeline
	pip, err := pipeline.LoadPipelineByID(db, s.PipelineID, true)
	if err != nil {
		return err
	}

	//Load environnement
	env, err := environment.LoadEnvironmentByID(db, s.EnvironmentID)
	if err != nil {
		return err
	}

	//Create a new pipeline build
	pb, err := queue.RunPipeline(db, app.ProjectKey, app, pip.Name, env.Name, s.Args, -1, sdk.PipelineBuildTrigger{
		ManualTrigger:    false,
		ScheduledTrigger: true,
	}, nil)

	if err != nil {
		return err
	}

	//References pipeline build version in execution
	t := time.Now()
	e.ExecutionDate = &t
	e.PipelineBuildVersion = pb.Version
	e.Executed = true

	//Update execution in database
	return UpdateExecution(db, e)
}
