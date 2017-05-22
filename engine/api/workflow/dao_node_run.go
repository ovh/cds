package workflow

import (
	"database/sql"
	"fmt"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/sdk"
)

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
