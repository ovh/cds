package poller

import (
	"database/sql"

	"github.com/go-gorp/gorp"
	"github.com/ovh/cds/engine/api/database"
	"github.com/ovh/cds/engine/log"
	"github.com/ovh/cds/sdk"
)

//InsertExecution a poller execution execution
func InsertExecution(db gorp.SqlExecutor, s *sdk.RepositoryPollerExecution) error {
	ds := database.RepositoryPollerExecution(*s)
	if err := db.Insert(&ds); err != nil {
		log.Warning("poller.InsertExecution> Unable to insert poller execution execution : %T %s", err, err)
		return err
	}
	*s = sdk.RepositoryPollerExecution(ds)
	return nil
}

//UpdateExecution a poller execution execution
func UpdateExecution(db gorp.SqlExecutor, s *sdk.RepositoryPollerExecution) error {
	ds := database.RepositoryPollerExecution(*s)
	if n, err := db.Update(&ds); err != nil {
		log.Warning("poller.UpdateExecution> Unable to update poller execution execution : %T %s", err, err)
		return err
	} else if n == 0 {
		return sdk.ErrNotFound
	}
	*s = sdk.RepositoryPollerExecution(ds)
	return nil
}

//DeleteExecution deletes executions
func DeleteExecution(db gorp.SqlExecutor, s *sdk.RepositoryPollerExecution) error {
	ds := database.RepositoryPollerExecution(*s)
	if n, err := db.Delete(&ds); err != nil {
		log.Warning("poller.DeleteExecution> Unable to delete poller execution execution : %T %s", err, err)
		return err
	} else if n == 0 {
		return sdk.ErrNotFound
	}
	*s = sdk.RepositoryPollerExecution(ds)
	return nil
}

//LoadExecutions loads all poller execution
func LoadExecutions(db gorp.SqlExecutor, appID, pipID int64) ([]sdk.RepositoryPollerExecution, error) {
	as := []database.RepositoryPollerExecution{}
	if _, err := db.Select(&as, "select * from poller_execution where application_id =$1, pipeline_id = $2", appID, pipID); err != nil {
		log.Warning("poller.LoadPendingExecutions> Unable to load poller execution execution : %T %s", err, err)
		return nil, err
	}
	ps := []sdk.RepositoryPollerExecution{}
	for _, s := range as {
		ps = append(ps, sdk.RepositoryPollerExecution(s))
	}
	return ps, nil
}

//LoadLastExecution loads last poller execution
func LoadLastExecution(db gorp.SqlExecutor, appID, pipID int64) (*sdk.RepositoryPollerExecution, error) {
	as := database.RepositoryPollerExecution{}
	if err := db.SelectOne(&as, "select * from poller_execution where application_id =$1 and pipeline_id = $2 order by execution_planned_date desc limit 1", appID, pipID); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		log.Warning("poller.LoadPendingExecutions> Unable to load poller execution execution : %T %s", err, err)
		return nil, err
	}
	ps := sdk.RepositoryPollerExecution(as)
	return &ps, nil
}

//LoadLastExecutedExecution loads last poller execution
func LoadLastExecutedExecution(db gorp.SqlExecutor, appID, pipID int64) (*sdk.RepositoryPollerExecution, error) {
	as := database.RepositoryPollerExecution{}
	if err := db.SelectOne(&as, "select * from poller_execution where application_id =$1 and pipeline_id = $2 and executed = true order by execution_planned_date desc limit 1", appID, pipID); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		log.Warning("poller.LoadPendingExecutions> Unable to load poller execution execution : %T %s", err, err)
		return nil, err
	}
	ps := sdk.RepositoryPollerExecution(as)
	return &ps, nil
}

//LoadNextExecution loads next poller execution
func LoadNextExecution(db gorp.SqlExecutor, appID, pipID int64) (*sdk.RepositoryPollerExecution, error) {
	as := database.RepositoryPollerExecution{}
	if err := db.SelectOne(&as, "select * from poller_execution where application_id =$1 and pipeline_id = $2and executed = false order by execution_planned_date desc limit 1", appID, pipID); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		log.Warning("poller.LoadPendingExecutions> Unable to load poller execution execution : %T %s", err, err)
		return nil, err
	}
	ps := sdk.RepositoryPollerExecution(as)
	return &ps, nil
}

//LoadPastExecutions loads all poller execution executed prior date 't'
func LoadPastExecutions(db gorp.SqlExecutor, appID, pipID int64) ([]sdk.RepositoryPollerExecution, error) {
	as := []database.RepositoryPollerExecution{}
	if _, err := db.Select(&as, "select * from poller_execution where application_id =$1 and pipeline_id = $2 and executed = true order by execution_date asc", appID, pipID); err != nil {
		log.Warning("poller.LoadPendingExecutions> Unable to load poller execution execution : %T %s", err, err)
		return nil, err
	}
	ps := []sdk.RepositoryPollerExecution{}
	for _, s := range as {
		ps = append(ps, sdk.RepositoryPollerExecution(s))
	}
	return ps, nil
}

//LoadPendingExecutions loads all poller execution
func LoadPendingExecutions(db gorp.SqlExecutor) ([]sdk.RepositoryPollerExecution, error) {
	as := []database.RepositoryPollerExecution{}
	if _, err := db.Select(&as, "select * from poller_execution where executed = false and execution_planned_date <=  now()"); err != nil {
		log.Warning("poller.LoadPendingExecutions> Unable to load poller execution execution : %T %s", err, err)
		return nil, err
	}
	ps := []sdk.RepositoryPollerExecution{}
	for _, s := range as {
		ps = append(ps, sdk.RepositoryPollerExecution(s))
	}
	return ps, nil
}

//LoadUnscheduledPollers loads unscheduled pollers
func LoadUnscheduledPollers(db gorp.SqlExecutor) ([]sdk.RepositoryPoller, error) {
	ps, err := LoadAll(db)
	res := []sdk.RepositoryPoller{}
	if err != nil {
		return nil, err
	}
	for _, s := range ps {
		exec, err := LoadLastExecution(db, s.ApplicationID, s.PipelineID)
		if err != nil {
			return nil, err
		}
		if exec == nil || exec.Executed {
			res = append(res, s)
		}
	}
	return ps, nil
}

//LockPollerExecutions locks table LockPollerExecutions
func LockPollerExecutions(db gorp.SqlExecutor) error {
	_, err := db.Exec("LOCK TABLE poller_execution IN ACCESS EXCLUSIVE MODE NOWAIT")
	return err
}
