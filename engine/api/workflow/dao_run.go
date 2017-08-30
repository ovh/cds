package workflow

import (
	"database/sql"
	"encoding/json"
	"time"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/sdk"
)

// insertWorkflowRun inserts in table "workflow_run""
func insertWorkflowRun(db gorp.SqlExecutor, w *sdk.WorkflowRun) error {
	runDB := Run(*w)
	if err := db.Insert(&runDB); err != nil {
		return sdk.WrapError(err, "insertWorkflowRun> Unable to insert run")
	}
	w.ID = runDB.ID
	return nil
}

// updateWorkflowRun updates in table "workflow_run""
func updateWorkflowRun(db gorp.SqlExecutor, w *sdk.WorkflowRun) error {
	w.LastModified = time.Now()
	runDB := Run(*w)
	if _, err := db.Update(&runDB); err != nil {
		return sdk.WrapError(err, "updateWorkflowRun> Unable to update run")
	}
	w.ID = runDB.ID
	return nil
}

//PostInsert is a db hook on WorkflowRun
func (r *Run) PostInsert(db gorp.SqlExecutor) error {
	w, errw := json.Marshal(r.Workflow)
	if errw != nil {
		return sdk.WrapError(errw, "Run.PostInsert> Unable to marshal workflow")
	}

	i, erri := json.Marshal(r.Infos)
	if erri != nil {
		return sdk.WrapError(erri, "Run.PostInsert> Unable to marshal infos")
	}

	if _, err := db.Exec("update workflow_run set workflow = $3, infos = $2 where id = $1", r.ID, i, w); err != nil {
		return sdk.WrapError(err, "Run.PostInsert> Unable to store marshalled infos")
	}

	return nil
}

//PostUpdate is a db hook on WorkflowRun
func (r *Run) PostUpdate(db gorp.SqlExecutor) error {
	return r.PostInsert(db)
}

//PostGet is a db hook on WorkflowRun
//It loads column workflow wich is in JSONB in table workflow_run
func (r *Run) PostGet(db gorp.SqlExecutor) error {
	var res = struct {
		W sql.NullString `db:"workflow"`
		I sql.NullString `db:"infos"`
	}{}

	if err := db.SelectOne(&res, "select workflow, infos from workflow_run where id = $1", r.ID); err != nil {
		return sdk.WrapError(err, "Run.PostGet> Unable to load marshalled workflow")
	}
	if res.W.Valid {
		w := sdk.Workflow{}
		if err := json.Unmarshal([]byte(res.W.String), &w); err != nil {
			return sdk.WrapError(err, "Run.PostGet> Unable to unmarshal workflow")
		}
		r.Workflow = w
	}

	if res.I.Valid {
		i := []sdk.WorkflowRunInfo{}
		if err := json.Unmarshal([]byte(res.I.String), &i); err != nil {
			return sdk.WrapError(err, "Run.PostGet> Unable to unmarshal infos")
		}
		r.Infos = i
	}

	return nil
}

// LoadLastRun returns the last run for a workflow
func LoadLastRun(db gorp.SqlExecutor, projectkey, workflowname string) (*sdk.WorkflowRun, error) {
	query := `select workflow_run.*
	from workflow_run 
	join project on workflow_run.project_id = project.id 
	join workflow on workflow_run.workflow_id = workflow.id
	where project.projectkey = $1 
	and workflow.name = $2 
	order by workflow_run.num desc limit 1`
	return loadRun(db, query, projectkey, workflowname)
}

// LoadRun returns a specific run
func LoadRun(db gorp.SqlExecutor, projectkey, workflowname string, number int64) (*sdk.WorkflowRun, error) {
	query := `select workflow_run.* 
	from workflow_run 
	join project on workflow_run.project_id = project.id 
	join workflow on workflow_run.workflow_id = workflow.id
	where project.projectkey = $1 
	and workflow.name = $2 
	and workflow_run.num = $3`
	return loadRun(db, query, projectkey, workflowname, number)
}

// LoadRunByIDAndProjectKey returns a specific run
func LoadRunByIDAndProjectKey(db gorp.SqlExecutor, projectkey string, id int64) (*sdk.WorkflowRun, error) {
	query := `select workflow_run.* 
	from workflow_run 
	join project on workflow_run.project_id = project.id 
	where project.projectkey = $1 
	and workflow_run.id = $2`
	return loadRun(db, query, projectkey, id)
}

func LoadRunByID(db gorp.SqlExecutor, id int64) (*sdk.WorkflowRun, error) {
	query := `select workflow_run.* 
	from workflow_run 
	where workflow_run.id = $1`
	return loadRun(db, query, id)
}

func loadAndLockRunByID(db gorp.SqlExecutor, id int64) (*sdk.WorkflowRun, error) {
	query := `select workflow_run.* 
	from workflow_run 
	where workflow_run.id = $1 for update nowait`
	return loadRun(db, query, id)
}

//LoadRuns loads all runs
//It retuns runs, offset, limit count and an error
func LoadRuns(db gorp.SqlExecutor, projectkey, workflowname string, offset, limit int) ([]sdk.WorkflowRun, int, int, int, error) {
	queryCount := `select count(workflow_run.id)
	from workflow_run 
	join project on workflow_run.project_id = project.id 
	join workflow on workflow_run.workflow_id = workflow.id
	where project.projectkey = $1 
	and workflow.name = $2`

	count, errc := db.SelectInt(queryCount, projectkey, workflowname)
	if errc != nil {
		return nil, 0, 0, 0, sdk.WrapError(errc, "LoadRuns> unable to load runs")
	}
	if count == 0 {
		return nil, 0, 0, 0, nil
	}

	query := `select workflow_run.* 
	from workflow_run 
	join project on workflow_run.project_id = project.id 
	join workflow on workflow_run.workflow_id = workflow.id
	where project.projectkey = $1 
	and workflow.name = $2 
	order by workflow_run.start desc 
	limit $3 offset $4`

	runs := []Run{}
	if _, err := db.Select(&runs, query, projectkey, workflowname, limit, offset); err != nil {
		return nil, 0, 0, 0, sdk.WrapError(errc, "LoadRuns> unable to load runs")
	}
	wruns := make([]sdk.WorkflowRun, len(runs))
	for i := range runs {
		wruns[i] = sdk.WorkflowRun(runs[i])
	}

	return wruns, offset, limit, int(count), nil
}

func loadRun(db gorp.SqlExecutor, query string, args ...interface{}) (*sdk.WorkflowRun, error) {
	runDB := &Run{}
	if err := db.SelectOne(runDB, query, args...); err != nil {
		if err == sql.ErrNoRows {
			return nil, sdk.ErrWorkflowNotFound
		}
		return nil, err
	}
	wr := sdk.WorkflowRun(*runDB)

	q := "select workflow_node_run.* from workflow_node_run where workflow_run_id = $1 ORDER BY workflow_node_run.sub_num DESC"
	dbNodeRuns := []NodeRun{}
	if _, err := db.Select(&dbNodeRuns, q, wr.ID); err != nil {
		if err != sql.ErrNoRows {
			return nil, sdk.WrapError(err, "loadRun> Unable to load workflow nodes run")
		}
	}

	for _, n := range dbNodeRuns {
		if err := n.PostGet(db); err != nil {
			return nil, sdk.WrapError(err, "loadRun> Unable to load workflow nodes run")
		}
		wnr := sdk.WorkflowNodeRun(n)
		if wr.WorkflowNodeRuns == nil {
			wr.WorkflowNodeRuns = make(map[int64][]sdk.WorkflowNodeRun)
		}
		wr.WorkflowNodeRuns[wnr.WorkflowNodeID] = append(wr.WorkflowNodeRuns[wnr.WorkflowNodeID], wnr)
	}

	return &wr, nil
}
