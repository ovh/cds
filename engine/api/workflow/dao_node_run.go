package workflow

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/go-gorp/gorp"
	"github.com/ovh/venom"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

//LoadNodeRun load a specific node run on a workflow
func LoadNodeRun(db gorp.SqlExecutor, projectkey, workflowname string, number, id int64) (*sdk.WorkflowNodeRun, error) {
	var rr = NodeRun{}

	query := `select workflow_node_run.*
	from workflow_node_run
	join workflow_run on workflow_run.id = workflow_node_run.workflow_run_id
	join project on project.id = workflow_run.project_id
	join workflow on workflow.id = workflow_run.workflow_id
	where project.projectkey = $1
	and workflow.name = $2
	and workflow_run.num = $3
	and workflow_node_run.id = $4`

	if err := db.SelectOne(&rr, query, projectkey, workflowname, number, id); err != nil {
		return nil, sdk.WrapError(err, "workflow.LoadNodeRun> Unable to load workflow_node_run proj=%s, workflow=%s, num=%d, node=%d", projectkey, workflowname, number, id)
	}

	r := sdk.WorkflowNodeRun(rr)
	return &r, nil
}

//LoadAndLockNodeRunByID load and lock a specific node run on a workflow
func LoadAndLockNodeRunByID(db gorp.SqlExecutor, id int64, wait bool) (*sdk.WorkflowNodeRun, error) {
	var rr = NodeRun{}
	query := `select workflow_node_run.*
	from workflow_node_run
	where workflow_node_run.id = $1 for update`
	if !wait {
		query += " nowait"
	}
	if err := db.SelectOne(&rr, query, id); err != nil {
		return nil, sdk.WrapError(err, "workflow.LoadAndLockNodeRunByID> Unable to load workflow_node_run node=%d", id)
	}
	r := sdk.WorkflowNodeRun(rr)
	return &r, nil
}

//LoadNodeRunByID load a specific node run on a workflow
func LoadNodeRunByID(db gorp.SqlExecutor, id int64) (*sdk.WorkflowNodeRun, error) {
	var rr = NodeRun{}
	query := `select workflow_node_run.*
	from workflow_node_run
	where workflow_node_run.id = $1`
	if err := db.SelectOne(&rr, query, id); err != nil {
		return nil, sdk.WrapError(err, "workflow.LoadNodeRunByID> Unable to load workflow_node_run node=%d", id)
	}
	r := sdk.WorkflowNodeRun(rr)
	return &r, nil
}

//insertWorkflowNodeRun insert in table workflow_node_run
func insertWorkflowNodeRun(db gorp.SqlExecutor, n *sdk.WorkflowNodeRun) error {
	nodeRunDB := NodeRun(*n)
	if err := db.Insert(&nodeRunDB); err != nil {
		return err
	}
	n.ID = nodeRunDB.ID
	log.Debug("insertWorkflowNodeRun> new node run: %d (%d)", n.ID, n.WorkflowNodeID)
	return nil
}

//updateNodeRunStatus update status of a workflow run node
func updateNodeRunStatus(db gorp.SqlExecutor, ID int64, status string) error {
	//Update workflow node run status
	query := "UPDATE workflow_node_run SET status = $1, last_modified = $2, done = $3 WHERE id = $4"
	now := time.Now()
	if _, err := db.Exec(query, status, now, now, ID); err != nil {
		return sdk.WrapError(err, "UpdateNodeRunStatus> Unable to set workflow_node_run id %d with status %s", ID, status)
	}
	return nil
}

//UpdateNodeRun updates in table workflow_node_run
func UpdateNodeRun(db gorp.SqlExecutor, n *sdk.WorkflowNodeRun) error {
	log.Debug("workflow.UpdateNodeRun> node.id=%d, status=%s", n.ID, n.Status)
	nodeRunDB := NodeRun(*n)
	if _, err := db.Update(&nodeRunDB); err != nil {
		return err
	}
	return nil
}

type sqlNodeRun struct {
	ID                 int64          `db:"id"`
	HookEvent          sql.NullString `db:"hook_event"`
	Manual             sql.NullString `db:"manual"`
	SourceNodeRuns     sql.NullString `db:"source_node_runs"`
	Payload            sql.NullString `db:"payload"`
	PipelineParameters sql.NullString `db:"pipeline_parameters"`
	BuildParameters    sql.NullString `db:"build_parameters"`
	Tests              sql.NullString `db:"tests"`
	Commits            sql.NullString `db:"commits"`
	Stages             sql.NullString `db:"stages"`
	TriggersRun        sql.NullString `db:"triggers_run"`
}

//PostInsert is a db hook on WorkflowNodeRun in table workflow_node_run
//it stores columns hook_event, manual, trigger_id, payload, pipeline_parameters, tests, commits
func (r *NodeRun) PostInsert(db gorp.SqlExecutor) error {
	var rr = sqlNodeRun{ID: r.ID}
	if r.TriggersRun != nil {
		s, err := gorpmapping.JSONToNullString(r.TriggersRun)
		if err != nil {
			return sdk.WrapError(err, "NodeRun.PostInsert> unable to get json from TriggerRun")
		}
		rr.TriggersRun = s
	}
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
	if r.PipelineParameters != nil {
		s, err := gorpmapping.JSONToNullString(r.PipelineParameters)
		if err != nil {
			return sdk.WrapError(err, "NodeRun.PostInsert> unable to get json from pipeline_parameters")
		}
		rr.PipelineParameters = s
	}
	if r.BuildParameters != nil {
		s, err := gorpmapping.JSONToNullString(r.BuildParameters)
		if err != nil {
			return sdk.WrapError(err, "NodeRun.PostInsert> unable to get json from build_parameters")
		}
		rr.BuildParameters = s
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
	if err := gorpmapping.JSONNullString(rr.TriggersRun, &r.TriggersRun); err != nil {
		return sdk.WrapError(err, "NodeRun.PostGet> Error loading node run trigger %d", r.ID)
	}
	if err := gorpmapping.JSONNullString(rr.Stages, &r.Stages); err != nil {
		return sdk.WrapError(err, "NodeRun.PostGet> Error loading node run %d", r.ID)
	}
	for i := range r.Stages {
		s := &r.Stages[i]
		for j := range s.RunJobs {
			rj := &s.RunJobs[j]
			if rj.Status == sdk.StatusWaiting.String() {
				rj.QueuedSeconds = time.Now().Unix() - rj.Queued.Unix()
			}
		}
	}
	if err := gorpmapping.JSONNullString(rr.SourceNodeRuns, &r.SourceNodeRuns); err != nil {
		return sdk.WrapError(err, "NodeRun.PostGet> Error loading node run %d", r.ID)
	}
	if err := gorpmapping.JSONNullString(rr.Commits, &r.Commits); err != nil {
		return sdk.WrapError(err, "NodeRun.PostGet> Error loading node run %d", r.ID)
	}
	if rr.HookEvent.Valid {
		r.HookEvent = new(sdk.WorkflowNodeRunHookEvent)
	}
	if err := gorpmapping.JSONNullString(rr.HookEvent, r.HookEvent); err != nil {
		return sdk.WrapError(err, "NodeRun.PostGet> Error loading node run %d", r.ID)
	}
	if rr.Manual.Valid {
		r.Manual = new(sdk.WorkflowNodeRunManual)
	}
	if err := gorpmapping.JSONNullString(rr.Manual, r.Manual); err != nil {
		return sdk.WrapError(err, "NodeRun.PostGet> Error loading node run %d", r.ID)
	}
	if err := gorpmapping.JSONNullString(rr.Payload, &r.Payload); err != nil {
		return sdk.WrapError(err, "NodeRun.PostGet> Error loading node run %d", r.ID)
	}
	if err := gorpmapping.JSONNullString(rr.BuildParameters, &r.BuildParameters); err != nil {
		return sdk.WrapError(err, "NodeRun.PostGet> Error loading node run %d", r.ID)
	}
	if rr.Tests.Valid {
		r.Tests = new(venom.Tests)
	}
	if err := gorpmapping.JSONNullString(rr.Tests, r.Tests); err != nil {
		return sdk.WrapError(err, "NodeRun.PostGet> Error loading node run %d", r.ID)
	}

	arts, errA := loadArtifactByNodeRunID(db, r.ID)
	if errA != nil {
		return sdk.WrapError(errA, "NodeRun.PostGet> Error loading artifacts for run %d", r.ID)
	}
	r.Artifacts = arts

	return nil
}
