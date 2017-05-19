package workflow

import (
	"database/sql"
	"encoding/json"
	"strconv"
	"time"

	"github.com/go-gorp/gorp"

	"fmt"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
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
	b, err := json.Marshal(r.Workflow)
	if err != nil {
		return sdk.WrapError(err, "Run.PostInsert> Unable to marshal workflow")
	}
	if _, err := db.Exec("update workflow_run set workflow = $2 where id = $1", r.ID, b); err != nil {
		return sdk.WrapError(err, "Run.PostInsert> Unable to store marshalled workflow")
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
	b, err := db.SelectStr("select workflow from workflow_run where id = $1", r.ID)
	if err != nil {
		return sdk.WrapError(err, "Run.PostGet> Unable to load marshalled workflow")
	}
	w := sdk.Workflow{}
	if err := json.Unmarshal([]byte(b), &w); err != nil {
		return sdk.WrapError(err, "Run.PostGet> Unable to unmarshal workflow")
	}
	r.Workflow = w
	return nil
}

//insertWorkflowNodeRun insert in table workflow_node_run
func insertWorkflowNodeRun(db gorp.SqlExecutor, n *sdk.WorkflowNodeRun) error {
	nodeRunDB := NodeRun(*n)
	if err := db.Insert(&nodeRunDB); err != nil {
		return err
	}
	n.ID = nodeRunDB.ID
	return nil
}

//updateWorkflowNodeRun updates in table workflow_node_run
func updateWorkflowNodeRun(db gorp.SqlExecutor, n *sdk.WorkflowNodeRun) error {
	nodeRunDB := NodeRun(*n)
	if _, err := db.Update(&nodeRunDB); err != nil {
		return err
	}
	return nil
}

type sqlNodeRun struct {
	ID                int64          `db:"id"`
	HookEvent         sql.NullString `db:"hook_event"`
	Manual            sql.NullString `db:"manual"`
	SourceNodeRuns    sql.NullString `db:"source_node_runs"`
	Payload           sql.NullString `db:"payload"`
	PipelineParameter sql.NullString `db:"pipeline_parameters"`
	Tests             sql.NullString `db:"tests"`
	Commits           sql.NullString `db:"commits"`
	Stages            sql.NullString `db:"stages"`
}

//PostInsert is a db hook on WorkflowNodeRun in table workflow_node_run
//it stores columns hook_event, manual, trigger_id, payload, pipeline_parameters, tests, commits
func (r *NodeRun) PostInsert(db gorp.SqlExecutor) error {
	var rr = sqlNodeRun{ID: r.ID}
	if r.Stages != nil {
		s, err := gorpmapping.JSONToNullString(r.Stages)
		if err != nil {
			return sdk.WrapError(err, "NodeRun.PostInsert> unable to get json from Stages")
		}
		rr.Stages = s
	}
	if r.SourceNodeRuns != nil {
		s, err := gorpmapping.JSONToNullString(r.SourceNodeRuns)
		if err != nil {
			return sdk.WrapError(err, "NodeRun.PostInsert> unable to get json from SourceNodeRuns")
		}
		rr.SourceNodeRuns = s
	}
	if r.HookEvent != nil {
		s, err := gorpmapping.JSONToNullString(r.HookEvent)
		if err != nil {
			return sdk.WrapError(err, "NodeRun.PostInsert> unable to get json from hook_event")
		}
		rr.HookEvent = s
	}
	if r.Manual != nil {
		s, err := gorpmapping.JSONToNullString(r.Manual)
		if err != nil {
			return sdk.WrapError(err, "NodeRun.PostInsert> unable to get json from manual")
		}
		rr.Manual = s
	}
	if r.Payload != nil {
		s, err := gorpmapping.JSONToNullString(r.Payload)
		if err != nil {
			return sdk.WrapError(err, "NodeRun.PostInsert> unable to get json from payload")
		}
		rr.Payload = s
	}
	if r.PipelineParameter != nil {
		s, err := gorpmapping.JSONToNullString(r.PipelineParameter)
		if err != nil {
			return sdk.WrapError(err, "NodeRun.PostInsert> unable to get json from pipeline_parameters")
		}
		rr.PipelineParameter = s
	}
	if r.Tests != nil {
		s, err := gorpmapping.JSONToNullString(r.Tests)
		if err != nil {
			return sdk.WrapError(err, "NodeRun.PostInsert> unable to get json from tests")
		}
		rr.Tests = s
	}
	if r.Commits != nil {
		s, err := gorpmapping.JSONToNullString(r.Commits)
		if err != nil {
			return sdk.WrapError(err, "NodeRun.PostInsert> unable to get json from commits")
		}
		rr.Commits = s
	}

	if n, err := db.Update(&rr); err != nil {
		return sdk.WrapError(err, "NodeRun.PostInsert> unable to update workflow_node_run id=%d", rr.ID)
	} else if n == 0 {
		return fmt.Errorf("workflow_node_run=%d was not updated", rr.ID)
	}
	if r.Artifacts != nil {
		//TODO
	}
	return nil
}

//PostUpdate is a db hook on WorkflowNodeRun in table workflow_node_run
//it stores columns hook_event, manual, trigger_id, payload, pipeline_parameters, tests, commits
func (r *NodeRun) PostUpdate(db gorp.SqlExecutor) error {
	return r.PostInsert(db)
}

//PostGet is a db hook
func (r *NodeRun) PostGet(db gorp.SqlExecutor) error {
	var rr = &sqlNodeRun{}

	query := "select * from workflow_node_run where id = $1"

	if err := db.SelectOne(rr, query, r.ID); err != nil {
		return sdk.WrapError(err, "NodeRun.PostGet> Unable to load workflow_node_run id=%d", r.ID)
	}
	if err := gorpmapping.JSONNullString(rr.Stages, &r.Stages); err != nil {
		return sdk.WrapError(err, "NodeRun.PostGet> Error loading node run %d", r.ID)
	}
	if err := gorpmapping.JSONNullString(rr.SourceNodeRuns, &r.SourceNodeRuns); err != nil {
		return sdk.WrapError(err, "NodeRun.PostGet> Error loading node run %d", r.ID)
	}
	if err := gorpmapping.JSONNullString(rr.Commits, &r.Commits); err != nil {
		return sdk.WrapError(err, "NodeRun.PostGet> Error loading node run %d", r.ID)
	}
	if err := gorpmapping.JSONNullString(rr.HookEvent, r.HookEvent); err != nil {
		return sdk.WrapError(err, "NodeRun.PostGet> Error loading node run %d", r.ID)
	}
	if err := gorpmapping.JSONNullString(rr.Manual, r.Manual); err != nil {
		return sdk.WrapError(err, "NodeRun.PostGet> Error loading node run %d", r.ID)
	}
	if err := gorpmapping.JSONNullString(rr.Payload, &r.Payload); err != nil {
		return sdk.WrapError(err, "NodeRun.PostGet> Error loading node run %d", r.ID)
	}
	if err := gorpmapping.JSONNullString(rr.PipelineParameter, &r.PipelineParameter); err != nil {
		return sdk.WrapError(err, "NodeRun.PostGet> Error loading node run %d", r.ID)
	}
	if err := gorpmapping.JSONNullString(rr.Tests, r.Tests); err != nil {
		return sdk.WrapError(err, "NodeRun.PostGet> Error loading node run %d", r.ID)
	}

	//TODO artifacts

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

// LoadRunByID returns a specific run
func LoadRunByID(db gorp.SqlExecutor, id int64) (*sdk.WorkflowRun, error) {
	query := `select workflow_run.* 
	from workflow_run 
	where workflow_run.id = $1`
	return loadRun(db, query, id)
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

	q := "select workflow_node_run.* from workflow_node_run where workflow_run_id = $1"
	dbNodeRuns := []NodeRun{}
	if _, err := db.Select(&dbNodeRuns, q, wr.ID); err != nil {
		if err != sql.ErrNoRows {
			return nil, sdk.WrapError(err, "loadRun> Unable to load workflow nodes run")
		}
	}

	for _, n := range dbNodeRuns {
		wnr := sdk.WorkflowNodeRun(n)
		wr.WorkflowNodeRuns = append(wr.WorkflowNodeRuns, wnr)
	}

	return &wr, nil
}

func insertWorkflowNodeJobRun(db gorp.SqlExecutor, j *sdk.WorkflowNodeJobRun) error {
	dbj := JobRun(*j)
	if err := db.Insert(&dbj); err != nil {
		return err
	}
	j.ID = dbj.ID

	log.Debug("insertWorkflowNodeJobRun> %d", j.ID)

	return nil
}

func keyBookJob(id int64) string {
	return cache.Key("book", "job", strconv.FormatInt(id, 10))
}

// PostInsert is a db hook on workflow_node_run_job
func (j *JobRun) PostInsert(s gorp.SqlExecutor) error {
	return j.PostUpdate(s)
}

// PostUpdate is a db hook on workflow_node_run_job
func (j *JobRun) PostUpdate(s gorp.SqlExecutor) error {
	jobJSON, err := json.Marshal(j.Job)
	if err != nil {
		return err
	}

	paramsJSON, errP := json.Marshal(j.Variables)
	if errP != nil {
		return errP
	}

	spawnJSON, errJ := json.Marshal(j.SpawnInfos)
	if errJ != nil {
		return errJ
	}

	query := "update workflow_node_run_job set job = $2, variables = $3, spawninfos= $4 where id = $1"
	if _, err := s.Exec(query, j.ID, jobJSON, paramsJSON, spawnJSON); err != nil {
		return err
	}

	return nil
}

// PostGet is a db hook on workflow_node_run_job
func (j *JobRun) PostGet(s gorp.SqlExecutor) error {
	h := sdk.Hatchery{}
	if cache.Get(keyBookJob(j.ID), &h) {
		j.BookedBy = h
	}

	query := "SELECT job, variables, spawninfos FROM workflow_node_run_job WHERE id = $1"
	var params, job, spawn []byte
	if err := s.QueryRow(query, j.ID).Scan(&job, &params, &spawn); err != nil {
		return err
	}

	if err := json.Unmarshal(job, &j.Job); err != nil {
		return err
	}
	if err := json.Unmarshal(params, &j.Variables); err != nil {
		return err
	}
	if err := json.Unmarshal(spawn, &j.SpawnInfos); err != nil {
		return err
	}

	j.QueuedSeconds = time.Now().Unix() - j.Queued.Unix()

	return nil
}
