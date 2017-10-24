package poller

import (
	"database/sql"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/sdk"
)

//InsertExecution a poller execution execution
func InsertExecution(db gorp.SqlExecutor, s *sdk.RepositoryPollerExecution) error {
	ds := RepositoryPollerExecution(*s)
	if err := db.Insert(&ds); err != nil {
		return sdk.WrapError(err, "poller.InsertExecution> Unable to insert poller execution execution : %T", err)
	}
	*s = sdk.RepositoryPollerExecution(ds)
	return nil
}

//UpdateExecution a poller execution execution
func UpdateExecution(db gorp.SqlExecutor, s *sdk.RepositoryPollerExecution) error {
	ds := RepositoryPollerExecution(*s)
	if n, err := db.Update(&ds); err != nil {
		return sdk.WrapError(err, "poller.UpdateExecution> Unable to update poller execution execution : %T ", err)
	} else if n == 0 {
		return sdk.ErrNotFound
	}
	*s = sdk.RepositoryPollerExecution(ds)
	return nil
}

//DeleteExecution deletes executions
func DeleteExecution(db gorp.SqlExecutor, s *sdk.RepositoryPollerExecution) error {
	ds := RepositoryPollerExecution(*s)
	if n, err := db.Delete(&ds); err != nil {
		return sdk.WrapError(err, "poller.DeleteExecution> Unable to delete poller execution execution : %T", err)
	} else if n == 0 {
		return sdk.ErrNotFound
	}
	*s = sdk.RepositoryPollerExecution(ds)
	return nil
}

// DeleteExecutionByApplicationID Delete poller execution for the given application
func DeleteExecutionByApplicationID(db gorp.SqlExecutor, appID int64) error {
	query := "DELETE FROM poller_execution WHERE application_id = $1"
	if _, err := db.Exec(query, appID); err != nil {
		return sdk.WrapError(err, "DeleteExecutionByApplicationID")
	}
	return nil
}

//LoadLastExecution loads last poller execution
func LoadLastExecution(db gorp.SqlExecutor, appID, pipID int64) (*sdk.RepositoryPollerExecution, error) {
	as := RepositoryPollerExecution{}
	if err := db.SelectOne(&as, "select * from poller_execution where application_id =$1 and pipeline_id = $2 order by execution_planned_date desc limit 1", appID, pipID); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, sdk.WrapError(err, "poller.LoadPendingExecutions> Unable to load poller execution execution : %T", err)
	}
	ps := sdk.RepositoryPollerExecution(as)
	return &ps, nil
}

//LoadNextExecution loads next poller execution
func LoadNextExecution(db gorp.SqlExecutor, appID, pipID int64) (*sdk.RepositoryPollerExecution, error) {
	as := RepositoryPollerExecution{}
	if err := db.SelectOne(&as, "select * from poller_execution where application_id =$1 and pipeline_id = $2 and executed = false order by execution_planned_date desc limit 1", appID, pipID); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, sdk.WrapError(err, "poller.LoadPendingExecutions> Unable to load poller execution execution : %T", err)
	}
	ps := sdk.RepositoryPollerExecution(as)
	return &ps, nil
}

//LoadPastExecutions loads all poller execution executed prior date 't'
func LoadPastExecutions(db gorp.SqlExecutor, appID, pipID int64) ([]sdk.RepositoryPollerExecution, error) {
	as := []RepositoryPollerExecution{}
	if _, err := db.Select(&as, "select * from poller_execution where application_id =$1 and pipeline_id = $2 and executed = true order by execution_date asc", appID, pipID); err != nil {
		return nil, sdk.WrapError(err, "poller.LoadPendingExecutions> Unable to load poller execution execution : %T", err)
	}
	ps := []sdk.RepositoryPollerExecution{}
	for _, s := range as {
		ps = append(ps, sdk.RepositoryPollerExecution(s))
	}
	return ps, nil
}

//LoadPendingExecutions loads all poller execution
func LoadPendingExecutions(db gorp.SqlExecutor) ([]sdk.RepositoryPollerExecution, error) {
	as := []RepositoryPollerExecution{}
	if _, err := db.Select(&as, "select * from poller_execution where executed = false and execution_planned_date <= now()"); err != nil {
		return nil, sdk.WrapError(err, "poller.LoadPendingExecutions> Unable to load poller execution execution : %T", err)
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

//LockPollerExecution locks table LockPollerExecutions
func LockPollerExecution(db gorp.SqlExecutor, id int64) error {
	_, err := db.Exec("SELECT * from poller_execution where id = $1 FOR UPDATE NOWAIT", id)
	return err
}
