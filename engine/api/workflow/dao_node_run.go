package workflow

import (
	"time"

	"github.com/go-gorp/gorp"
	"github.com/ovh/venom"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

//LoadNodeRun load a specific node run on a workflow
func LoadNodeRun(db gorp.SqlExecutor, projectkey, workflowname string, number, id int64, withArtifacts bool) (*sdk.WorkflowNodeRun, error) {
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

	r, err := fromDBNodeRun(rr)
	if err != nil {
		return nil, sdk.WrapError(err, "LoadNodeRun>")
	}

	if withArtifacts {
		arts, errA := loadArtifactByNodeRunID(db, r.ID)
		if errA != nil {
			return nil, sdk.WrapError(errA, "LoadNodeRun>Error loading artifacts for run %d", r.ID)
		}
		r.Artifacts = arts
	}

	return r, nil

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
	return fromDBNodeRun(rr)
}

//LoadNodeRunByID load a specific node run on a workflow
func LoadNodeRunByID(db gorp.SqlExecutor, id int64, withArtifacts bool) (*sdk.WorkflowNodeRun, error) {
	var rr = NodeRun{}
	query := `select workflow_node_run.*
	from workflow_node_run
	where workflow_node_run.id = $1`
	if err := db.SelectOne(&rr, query, id); err != nil {
		return nil, sdk.WrapError(err, "workflow.LoadNodeRunByID> Unable to load workflow_node_run node=%d", id)
	}

	r, err := fromDBNodeRun(rr)
	if err != nil {
		return nil, sdk.WrapError(err, "LoadNodeRun>")
	}

	if withArtifacts {
		arts, errA := loadArtifactByNodeRunID(db, r.ID)
		if errA != nil {
			return nil, sdk.WrapError(errA, "LoadNodeRun>Error loading artifacts for run %d", r.ID)
		}
		r.Artifacts = arts
	}

	return r, nil

}

//insertWorkflowNodeRun insert in table workflow_node_run
func insertWorkflowNodeRun(db gorp.SqlExecutor, n *sdk.WorkflowNodeRun) error {
	nodeRunDB, err := makeDBNodeRun(*n)
	if err != nil {
		return err
	}
	if err := db.Insert(nodeRunDB); err != nil {
		return err
	}
	n.ID = nodeRunDB.ID

	log.Debug("insertWorkflowNodeRun> new node run: %d (%d)", n.ID, n.WorkflowNodeID)
	return nil
}

func fromDBNodeRun(rr NodeRun) (*sdk.WorkflowNodeRun, error) {
	r := new(sdk.WorkflowNodeRun)
	r.WorkflowRunID = rr.WorkflowRunID
	r.ID = rr.ID
	r.WorkflowNodeID = rr.WorkflowNodeID
	r.Number = rr.Number
	r.SubNumber = rr.SubNumber
	r.Status = rr.Status
	r.Start = rr.Start
	r.Done = rr.Done
	r.LastModified = rr.LastModified
	//r.Done              = rr. <<---- WHERE ?

	if err := gorpmapping.JSONNullString(rr.TriggersRun, &r.TriggersRun); err != nil {
		return nil, sdk.WrapError(err, "fromDBNodeRun>Error loading node run trigger %d", r.ID)
	}
	if err := gorpmapping.JSONNullString(rr.Stages, &r.Stages); err != nil {
		return nil, sdk.WrapError(err, "fromDBNodeRun>Error loading node run %d", r.ID)
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
		return nil, sdk.WrapError(err, "fromDBNodeRun>Error loading node run %d", r.ID)
	}
	if err := gorpmapping.JSONNullString(rr.Commits, &r.Commits); err != nil {
		return nil, sdk.WrapError(err, "fromDBNodeRun>Error loading node run %d", r.ID)
	}
	if rr.HookEvent.Valid {
		r.HookEvent = new(sdk.WorkflowNodeRunHookEvent)
		if err := gorpmapping.JSONNullString(rr.HookEvent, r.HookEvent); err != nil {
			return nil, sdk.WrapError(err, "fromDBNodeRun>Error loading node run %d", r.ID)
		}
	}
	if rr.Manual.Valid {
		r.Manual = new(sdk.WorkflowNodeRunManual)
		if err := gorpmapping.JSONNullString(rr.Manual, r.Manual); err != nil {
			return nil, sdk.WrapError(err, "fromDBNodeRun>Error loading node run %d", r.ID)
		}
	}
	if err := gorpmapping.JSONNullString(rr.Payload, &r.Payload); err != nil {
		return nil, sdk.WrapError(err, "fromDBNodeRun>Error loading node run %d", r.ID)
	}
	if err := gorpmapping.JSONNullString(rr.BuildParameters, &r.BuildParameters); err != nil {
		return nil, sdk.WrapError(err, "fromDBNodeRun>Error loading node run %d", r.ID)
	}
	if rr.PipelineParameters.Valid {
		if err := gorpmapping.JSONNullString(rr.PipelineParameters, r.PipelineParameters); err != nil {
			return nil, sdk.WrapError(err, "fromDBNodeRun>Error loading node run %d", r.ID)
		}
	}
	if rr.Tests.Valid {
		r.Tests = new(venom.Tests)
		if err := gorpmapping.JSONNullString(rr.Tests, r.Tests); err != nil {
			return nil, sdk.WrapError(err, "fromDBNodeRun>Error loading node run %d", r.ID)
		}
	}

	return r, nil
}

func makeDBNodeRun(n sdk.WorkflowNodeRun) (*NodeRun, error) {
	nodeRunDB := new(NodeRun)
	nodeRunDB.ID = n.ID
	nodeRunDB.WorkflowRunID = n.WorkflowRunID
	nodeRunDB.WorkflowNodeID = n.WorkflowNodeID
	nodeRunDB.Number = n.Number
	nodeRunDB.SubNumber = n.SubNumber
	nodeRunDB.Status = n.Status
	nodeRunDB.Start = n.Start
	nodeRunDB.Done = n.Done
	nodeRunDB.LastModified = n.LastModified

	if n.TriggersRun != nil {
		s, err := gorpmapping.JSONToNullString(n.TriggersRun)
		if err != nil {
			return nil, sdk.WrapError(err, "makeDBNodeRun> unable to get json from TriggerRun")
		}
		nodeRunDB.TriggersRun = s
	}
	if n.Stages != nil {
		s, err := gorpmapping.JSONToNullString(n.Stages)
		if err != nil {
			return nil, sdk.WrapError(err, "makeDBNodeRun> unable to get json from Stages")
		}
		nodeRunDB.Stages = s
	}
	if n.SourceNodeRuns != nil {
		s, err := gorpmapping.JSONToNullString(n.SourceNodeRuns)
		if err != nil {
			return nil, sdk.WrapError(err, "makeDBNodeRun> unable to get json from SourceNodeRuns")
		}
		nodeRunDB.SourceNodeRuns = s
	}
	if n.HookEvent != nil {
		s, err := gorpmapping.JSONToNullString(n.HookEvent)
		if err != nil {
			return nil, sdk.WrapError(err, "makeDBNodeRun> unable to get json from hook_event")
		}
		nodeRunDB.HookEvent = s
	}
	if n.Manual != nil {
		s, err := gorpmapping.JSONToNullString(n.Manual)
		if err != nil {
			return nil, sdk.WrapError(err, "makeDBNodeRun> unable to get json from manual")
		}
		nodeRunDB.Manual = s
	}
	if n.Payload != nil {
		s, err := gorpmapping.JSONToNullString(n.Payload)
		if err != nil {
			return nil, sdk.WrapError(err, "makeDBNodeRun> unable to get json from payload")
		}
		nodeRunDB.Payload = s
	}
	if n.PipelineParameters != nil {
		s, err := gorpmapping.JSONToNullString(n.PipelineParameters)
		if err != nil {
			return nil, sdk.WrapError(err, "makeDBNodeRun> unable to get json from pipeline_parameters")
		}
		nodeRunDB.PipelineParameters = s
	}
	if n.BuildParameters != nil {
		s, err := gorpmapping.JSONToNullString(n.BuildParameters)
		if err != nil {
			return nil, sdk.WrapError(err, "makeDBNodeRun> unable to get json from build_parameters")
		}
		nodeRunDB.BuildParameters = s
	}
	if n.Tests != nil {
		s, err := gorpmapping.JSONToNullString(n.Tests)
		if err != nil {
			return nil, sdk.WrapError(err, "makeDBNodeRun> unable to get json from tests")
		}
		nodeRunDB.Tests = s
	}
	if n.Commits != nil {
		s, err := gorpmapping.JSONToNullString(n.Commits)
		if err != nil {
			return nil, sdk.WrapError(err, "makeDBNodeRun> unable to get json from commits")
		}
		nodeRunDB.Commits = s
	}

	return nodeRunDB, nil
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
	nodeRunDB, err := makeDBNodeRun(*n)
	if err != nil {
		return err
	}
	if _, err := db.Update(nodeRunDB); err != nil {
		return err
	}
	return nil
}
