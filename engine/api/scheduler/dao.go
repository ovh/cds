package scheduler

import (
	"time"

	"github.com/go-gorp/gorp"
	"github.com/ovh/cds/engine/api/database"
	"github.com/ovh/cds/engine/log"
	"github.com/ovh/cds/sdk"
)

// LoadAll retrieves all pipeline scheduler from database
func LoadAll(db gorp.SqlExecutor) ([]sdk.PipelineScheduler, error) {
	s := []database.PipelineScheduler{}
	if _, err := db.Select(&s, "select * from pipeline_scheduler"); err != nil {
		log.Warning("LoadAll> Unable to load pipeline scheduler : %T %s", err, err)
		return nil, err
	}
	ps := []sdk.PipelineScheduler{}
	for i := range s {
		if err := s[i].PostGet(db); err != nil {
			return nil, err
		}
		ps = append(ps, sdk.PipelineScheduler(s[i]))
	}
	return ps, nil
}

//Insert a pipeline scheduler
func Insert(db gorp.SqlExecutor, s *sdk.PipelineScheduler) error {
	ds := database.PipelineScheduler(*s)
	if err := db.Insert(&ds); err != nil {
		log.Warning("Insert> Unable to insert pipeline scheduler : %T %s", err, err)
		return err
	}
	*s = sdk.PipelineScheduler(ds)
	return nil
}

//Update a pipeline scheduler
func Update(db gorp.SqlExecutor, s *sdk.PipelineScheduler) error {
	ds := database.PipelineScheduler(*s)
	if n, err := db.Update(&ds); err != nil {
		log.Warning("Update> Unable to update pipeline scheduler : %T %s", err, err)
		return err
	} else if n == 0 {
		return sdk.ErrNotFound
	}
	*s = sdk.PipelineScheduler(ds)
	return nil
}

//Delete a pipeline scheduler
func Delete(db gorp.SqlExecutor, s *sdk.PipelineScheduler) error {
	ds := database.PipelineScheduler(*s)
	if n, err := db.Delete(&ds); err != nil {
		log.Warning("Delete> Unable to delete pipeline scheduler : %T %s", err, err)
		return err
	} else if n == 0 {
		return sdk.ErrNotFound
	}
	*s = sdk.PipelineScheduler(ds)
	return nil
}

//Load loads a PipelineScheduler by id
func Load(db gorp.SqlExecutor, id int64) (*sdk.PipelineScheduler, error) {
	ds := database.PipelineScheduler{}
	if err := db.SelectOne(&ds, "select * from pipeline_scheduler where id = $1", id); err != nil {
		log.Warning("Load> Unable to load pipeline scheduler : %T %s", err, err)
		return nil, err
	}
	s := sdk.PipelineScheduler(ds)
	return &s, nil
}

//InsertExecution a pipeline scheduler execution
func InsertExecution(db gorp.SqlExecutor, s *sdk.PipelineSchedulerExecution) error {
	ds := database.PipelineSchedulerExecution(*s)
	if err := db.Insert(&ds); err != nil {
		log.Warning("InsertExecution> Unable to insert pipeline scheduler execution : %T %s", err, err)
		return err
	}
	*s = sdk.PipelineSchedulerExecution(ds)
	return nil
}

//UpdateExecution a pipeline scheduler execution
func UpdateExecution(db gorp.SqlExecutor, s *sdk.PipelineSchedulerExecution) error {
	ds := database.PipelineSchedulerExecution(*s)
	if n, err := db.Update(&ds); err != nil {
		log.Warning("UpdateExecution> Unable to insert pipeline scheduler execution : %T %s", err, err)
		return err
	} else if n == 0 {
		return sdk.ErrNotFound
	}
	*s = sdk.PipelineSchedulerExecution(ds)
	return nil
}

//LoadPendingExecutions loads all pipeline execution until a date
func LoadPendingExecutions(db gorp.SqlExecutor, until time.Time) ([]sdk.PipelineSchedulerExecution, error) {
	s := []database.PipelineSchedulerExecution{}
	if _, err := db.Select(&s, "select * from pipeline_scheduler_execution where executed = false and execution_planned_date >=  now() + interval '5 second' "); err != nil {
		log.Warning("LoadPendingExecutions> Unable to load pipeline scheduler execution : %T %s", err, err)
		return nil, err
	}
	ps := []sdk.PipelineSchedulerExecution{}
}
