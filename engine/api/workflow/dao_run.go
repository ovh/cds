package workflow

import (
	"github.com/go-gorp/gorp"

	"encoding/json"

	"database/sql"

	"github.com/ovh/cds/sdk"
)

func insertWorkflowRun(db gorp.SqlExecutor, w *sdk.WorkflowRun) error {
	runDB := Run(*w)
	if err := db.Insert(&runDB); err != nil {
		return sdk.WrapError(err, "insertWorkflowRun> Unable to insert run")
	}

	w.ID = runDB.ID

	return nil
}

//PostInsert is a db hook
func (r *Run) PostInsert(db gorp.SqlExecutor) error {
	b, err := json.Marshal(r.Workflow)
	if err != nil {
		return sdk.WrapError(err, "Run.PostInsert> Unable to marshal workflow")
	}

	if _, err := db.Exec("update workflow_run set workflow = $2 where id = $1", r.ID, b); err != nil {
		return sdk.WrapError(err, "Run.PostInsert> Unable to store marshalled workflow")
	}

	return nil
}

//PostGet is a db hook
func (r *Run) PostGet(db gorp.SqlExecutor) error {
	b, err := db.SelectStr("select workflow from workflow_run where id = $1", r.ID)
	if err != nil {
		return sdk.WrapError(err, "Run.PostGet> Unable to load marshalled workflow")
	}

	w := &sdk.Workflow{}
	if err := json.Unmarshal([]byte(b), w); err != nil {
		return sdk.WrapError(err, "Run.PostGet> Unable to unmarshal workflow")
	}

	r.Workflow = *w
	return nil
}

func insertWorkflowNodeRun(db gorp.SqlExecutor, n *sdk.WorkflowNodeRun) error {
	nodeRunDB := NodeRun(*n)
	if err := db.Insert(&nodeRunDB); err != nil {
		return err
	}
	n.ID = nodeRunDB.ID
	return nil
}

//PostInsert is a db hook
func (r *NodeRun) PostInsert(db gorp.SqlExecutor) error {
	var u = struct {
		TriggerID         sql.NullInt64  `db:"trigger_id"`
		HookEvent         sql.NullString `db:"hook_event"`
		Manual            sql.NullString `db:"manual"`
		Payload           sql.NullString `db:"payload"`
		PipelineParameter sql.NullString `db:"pipeline_parameter"`
		Artifacts         sql.NullString `db:"artifacts"`
	}{}

	if r.TriggerID != 0 {

	}
	if r.HookEvent != nil {

	}
	if r.Manual != nil {

	}
	if r.Payload != nil {

	}
	if r.PipelineParameter != nil {

	}
	if r.Artifacts != nil {

	}
	if r.Tests != nil {

	}
	if r.Commits != nil {

	}
}

//PostUpdate is a db hook
func (r *NodeRun) PostUpdate(db gorp.SqlExecutor) error {

}

//PostGet is a db hook
func (r *NodeRun) PostGet(db gorp.SqlExecutor) error {

}

func loadWorkflowRun(db gorp.SqlExecutor, query string, args ...interface{}) (*sdk.WorkflowRun, error) {
	runDB := &Run{}
	if err := db.SelectOne(runDB, query, args...); err != nil {
		if err == sql.ErrNoRows {
			return nil, sdk.ErrWorkflowNotFound
		}
		return nil, err
	}
	w := sdk.WorkflowRun(*runDB)
	return &w, nil
}

// LoadLastRun returns the last run for a workflow
func LoadLastRun(db gorp.SqlExecutor, projectkey, workflowname string) (*sdk.WorkflowRun, error) {
	query := "select workflow_run.* from workflow where project_key = $1, workflow_name = $2 order by num desc limit 1"
	return loadRun(db, query, projectkey, workflowname)
}

// LoadRun returns a specific run
func LoadRun(db gorp.SqlExecutor, projectkey, workflowname string, number int64) (*sdk.WorkflowRun, error) {
	query := "select workflow_run.* from workflow where project_key = $1, workflow_name = $2 and num = $3"
	return loadRun(db, query, projectkey, workflowname, number)
}

func loadRun(db gorp.SqlExecutor, query string, args ...interface{}) (*sdk.WorkflowRun, error) {
	wr, err := loadWorkflowRun(db, query, args...)
	if err != nil {
		return nil, sdk.WrapError(err, "loadRun> Unable to load workflow run")
	}

	q := "select workflow_node_run.* from workflow_node_run where workflow_run_id = $1"
	dbNodeRuns := []NodeRun{}
	if _, err := db.Select(&dbNodeRuns, q, wr.ID); err != nil {
		if err != sql.ErrNoRows {
			return nil, sdk.WrapError(err, "loadRun> Unable to load workflow nodes run")
		}
	}

	for _, n := range dbNodeRuns {
		event := struct {
			EventHook sql.NullString `db:"hook_event"`
			Manual    sql.NullString `db:"manual"`
			TriggerID sql.NullInt64  `db:"trigger_id"`
		}{}

		if err := db.SelectOne(&event, "select hook_event, manual, trigger_id from workflow_node_run where id = $1", n.ID); err != nil {
			return nil, sdk.WrapError(err, "loadRun> Unable to load events")
		}

		//load event_hook
		if event.EventHook.Valid {
			e := &sdk.WorkflowNodeRunHookEvent{}
			if err := json.Unmarshal([]byte(event.EventHook.String), e); err != nil {
				return nil, sdk.WrapError(err, "loadRun> Unable to unmarshal hook_event")
			}
			n.HookEvent = e
		}

		//Load manual
		if event.Manual.Valid {
			e := &sdk.WorkflowNodeRunManual{}
			if err := json.Unmarshal([]byte(event.Manual.String), e); err != nil {
				return nil, sdk.WrapError(err, "loadRun> Unable to unmarshal manual")
			}
			n.Manual = e
		}

		//Load trigger_id
		if event.TriggerID.Valid {
			n.TriggerID = event.TriggerID.Int64
		}

		wnr := sdk.WorkflowNodeRun(n)
		wr.WorkflowNodeRuns = append(wr.WorkflowNodeRuns, wnr)
	}

	return wr, nil
}
