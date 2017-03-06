package scheduler

import (
	"database/sql"
	"time"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/log"
	"github.com/ovh/cds/sdk"
)

func loadPipelineSchedulers(db gorp.SqlExecutor, query string, args ...interface{}) ([]sdk.PipelineScheduler, error) {
	s := []PipelineScheduler{}
	if _, err := db.Select(&s, query, args...); err != nil {
		log.Warning("loadPipelineScheduler> Unable to load pipeline scheduler : %T %s", err, err)
		return nil, err
	}
	ps := []sdk.PipelineScheduler{}
	for i := range s {
		if err := s[i].PostGet(db); err != nil {
			return nil, err
		}
		x := sdk.PipelineScheduler(s[i])
		var err error
		x.LastExecution, err = LoadLastExecutedExecution(db, x.ID)
		if err != nil {
			return nil, err
		}
		x.NextExecution, err = LoadNextExecution(db, x.ID, x.Timezone)
		if err != nil {
			return nil, err
		}
		ps = append(ps, x)
	}
	return ps, nil
}

// LoadAll retrieves all pipeline scheduler from database
func LoadAll(db gorp.SqlExecutor) ([]sdk.PipelineScheduler, error) {
	return loadPipelineSchedulers(db, "select * from pipeline_scheduler")
}

//Insert a pipeline scheduler
func Insert(db gorp.SqlExecutor, s *sdk.PipelineScheduler) error {
	if s.Timezone == "" {
		s.Timezone = "UTC"
	}
	ds := PipelineScheduler(*s)
	if err := db.Insert(&ds); err != nil {
		log.Warning("Insert> Unable to insert pipeline scheduler : %T %s", err, err)
		return err
	}
	*s = sdk.PipelineScheduler(ds)
	return nil
}

//Update a pipeline scheduler
func Update(db gorp.SqlExecutor, s *sdk.PipelineScheduler) error {
	ds := PipelineScheduler(*s)
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
	ds := PipelineScheduler(*s)
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
	ds := PipelineScheduler{}
	if err := db.SelectOne(&ds, "select * from pipeline_scheduler where id = $1", id); err != nil {
		log.Warning("Load> Unable to load pipeline scheduler : %T %s", err, err)
		return nil, err
	}
	s := sdk.PipelineScheduler(ds)
	return &s, nil
}

//InsertExecution a pipeline scheduler execution
func InsertExecution(db gorp.SqlExecutor, s *sdk.PipelineSchedulerExecution) error {
	ds := PipelineSchedulerExecution(*s)
	if err := db.Insert(&ds); err != nil {
		log.Warning("InsertExecution> Unable to insert pipeline scheduler execution : %T %s", err, err)
		return err
	}
	*s = sdk.PipelineSchedulerExecution(ds)
	return nil
}

//UpdateExecution a pipeline scheduler execution
func UpdateExecution(db gorp.SqlExecutor, s *sdk.PipelineSchedulerExecution) error {
	ds := PipelineSchedulerExecution(*s)
	if n, err := db.Update(&ds); err != nil {
		log.Warning("UpdateExecution> Unable to update pipeline scheduler execution : %T %s", err, err)
		return err
	} else if n == 0 {
		return sdk.ErrNotFound
	}
	*s = sdk.PipelineSchedulerExecution(ds)
	return nil
}

//DeleteExecution deletes executions
func DeleteExecution(db gorp.SqlExecutor, s *sdk.PipelineSchedulerExecution) error {
	ds := PipelineSchedulerExecution(*s)
	if n, err := db.Delete(&ds); err != nil {
		log.Warning("DeleteExecution> Unable to delete pipeline scheduler execution : %T %s", err, err)
		return err
	} else if n == 0 {
		return sdk.ErrNotFound
	}
	*s = sdk.PipelineSchedulerExecution(ds)
	return nil
}

//LoadExecutions loads all pipeline execution
func LoadExecutions(db gorp.SqlExecutor, schedulerID int64) ([]sdk.PipelineSchedulerExecution, error) {
	as := []PipelineSchedulerExecution{}
	if _, err := db.Select(&as, "select * from pipeline_scheduler_execution where pipeline_scheduler_id = $1", schedulerID); err != nil {
		log.Warning("LoadPendingExecutions> Unable to load pipeline scheduler execution : %T %s", err, err)
		return nil, err
	}
	ps := []sdk.PipelineSchedulerExecution{}
	for _, s := range as {
		ps = append(ps, sdk.PipelineSchedulerExecution(s))
	}
	return ps, nil
}

//LoadLastExecution loads last pipeline execution
func LoadLastExecution(db gorp.SqlExecutor, id int64) (*sdk.PipelineSchedulerExecution, error) {
	as := PipelineSchedulerExecution{}
	if err := db.SelectOne(&as, "select * from pipeline_scheduler_execution where pipeline_scheduler_id = $1 order by execution_planned_date desc limit 1", id); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		log.Warning("LoadPendingExecutions> Unable to load pipeline scheduler execution : %T %s", err, err)
		return nil, err
	}
	ps := sdk.PipelineSchedulerExecution(as)
	return &ps, nil
}

//LoadLastExecutedExecution loads last pipeline execution
func LoadLastExecutedExecution(db gorp.SqlExecutor, id int64) (*sdk.PipelineSchedulerExecution, error) {
	as := PipelineSchedulerExecution{}
	if err := db.SelectOne(&as, "select * from pipeline_scheduler_execution where pipeline_scheduler_id = $1 and executed = true order by execution_planned_date desc limit 1", id); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		log.Warning("LoadPendingExecutions> Unable to load pipeline scheduler execution : %T %s", err, err)
		return nil, err
	}
	ps := sdk.PipelineSchedulerExecution(as)
	return &ps, nil
}

//LoadNextExecution loads next pipeline execution
func LoadNextExecution(db gorp.SqlExecutor, id int64, timezone string) (*sdk.PipelineSchedulerExecution, error) {
	as := PipelineSchedulerExecution{}
	if err := db.SelectOne(&as, "select * from pipeline_scheduler_execution where pipeline_scheduler_id = $1 and executed = false order by execution_planned_date desc limit 1", id); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, sdk.WrapError(err, "LoadNextExecution> Unable to load pipeline scheduler execution")
	}
	if timezone == "" {
		timezone = "UTC"
	}
	t, errT := time.LoadLocation(timezone)
	if errT != nil {
		return nil, sdk.WrapError(errT, "LoadNextExecution> Cannot get timezone")
	}
	as.ExecutionPlannedDate = as.ExecutionPlannedDate.In(t)
	if as.ExecutionDate != nil {
		*as.ExecutionDate = as.ExecutionDate.In(t)
	}

	ps := sdk.PipelineSchedulerExecution(as)
	return &ps, nil
}

//LoadPastExecutions loads all pipeline execution executed prior date 't'
func LoadPastExecutions(db gorp.SqlExecutor, id int64) ([]sdk.PipelineSchedulerExecution, error) {
	as := []PipelineSchedulerExecution{}
	if _, err := db.Select(&as, "select * from pipeline_scheduler_execution where pipeline_scheduler_id = $1 and executed = true order by execution_date asc", id); err != nil {
		log.Warning("LoadPendingExecutions> Unable to load pipeline scheduler execution : %T %s", err, err)
		return nil, err
	}
	ps := []sdk.PipelineSchedulerExecution{}
	for _, s := range as {
		ps = append(ps, sdk.PipelineSchedulerExecution(s))
	}
	return ps, nil
}

//LoadPendingExecutions loads all pipeline execution
func LoadPendingExecutions(db gorp.SqlExecutor) ([]sdk.PipelineSchedulerExecution, error) {
	as := []PipelineSchedulerExecution{}
	if _, err := db.Select(&as, "select * from pipeline_scheduler_execution where executed = false and execution_planned_date <=  now()"); err != nil {
		log.Warning("LoadPendingExecutions> Unable to load pipeline scheduler execution : %T %s", err, err)
		return nil, err
	}
	ps := []sdk.PipelineSchedulerExecution{}
	for _, s := range as {
		ps = append(ps, sdk.PipelineSchedulerExecution(s))
	}
	return ps, nil
}

//LoadUnscheduledPipelines loads unscheduled pipelines
func LoadUnscheduledPipelines(db gorp.SqlExecutor) ([]sdk.PipelineScheduler, error) {
	ps, err := LoadAll(db)
	res := []sdk.PipelineScheduler{}
	if err != nil {
		return nil, err
	}
	for _, s := range ps {
		exec, err := LoadLastExecution(db, s.ID)
		if err != nil {
			return nil, err
		}
		if exec == nil || exec.Executed {
			res = append(res, s)
		}
	}
	return ps, nil
}

//LockPipelineExecutions locks table LockPipelineExecutions
func LockPipelineExecutions(db gorp.SqlExecutor) error {
	_, err := db.Exec("LOCK TABLE pipeline_scheduler_execution IN ACCESS EXCLUSIVE MODE NOWAIT")
	return err
}

//GetByApplication get all pipeline schedulers for an application
func GetByApplication(db gorp.SqlExecutor, app *sdk.Application) ([]sdk.PipelineScheduler, error) {
	return loadPipelineSchedulers(db, "select * from pipeline_scheduler where application_id = $1", app.ID)
}

//GetByPipeline get all pipeline schedulers for a pipeline
func GetByPipeline(db gorp.SqlExecutor, pip *sdk.Pipeline) ([]sdk.PipelineScheduler, error) {
	return loadPipelineSchedulers(db, "select * from pipeline_scheduler where pipeline_id = $1", pip.ID)
}

//GetByApplicationPipeline get all pipeline schedulers for a application/pipeline
func GetByApplicationPipeline(db gorp.SqlExecutor, app *sdk.Application, pip *sdk.Pipeline) ([]sdk.PipelineScheduler, error) {
	return loadPipelineSchedulers(db, "select * from pipeline_scheduler where application_id = $1 and pipeline_id = $2", app.ID, pip.ID)
}

//GetByApplicationPipelineEnv get all pipeline schedulers for a application/pipeline
func GetByApplicationPipelineEnv(db gorp.SqlExecutor, app *sdk.Application, pip *sdk.Pipeline, env *sdk.Environment) ([]sdk.PipelineScheduler, error) {
	return loadPipelineSchedulers(db, "select * from pipeline_scheduler where application_id = $1 and pipeline_id = $2 and environment_id = $3", app.ID, pip.ID, env.ID)
}
