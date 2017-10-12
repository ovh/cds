package workflow

import (
	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/sdk"
)

// ResyncPipeline resync all pipelines with DB for the given workflow run
func ResyncPipeline(db gorp.SqlExecutor, wr *sdk.WorkflowRun) error {
	// Resync from root node
	if err := resyncNode(db, wr.Workflow.Root); err != nil {
		return sdk.WrapError(err, "resyncPipeline")
	}

	// Resync from join
	for i := range wr.Workflow.Joins {
		wj := &wr.Workflow.Joins[i]
		for j := range wj.Triggers {
			t := &wj.Triggers[j]
			if err := resyncNode(db, &t.WorkflowDestNode); err != nil {
				return sdk.WrapError(err, "resyncPipeline> Cannot resync node %s", t.WorkflowDestNode.Name)
			}
		}
	}

	return updateWorkflowRun(db, wr)
}

func resyncNode(db gorp.SqlExecutor, n *sdk.WorkflowNode) error {
	pip, errP := pipeline.LoadPipelineByID(db, n.Pipeline.ID, true)
	if errP != nil {
		return sdk.WrapError(errP, "resyncNode> Cannot load pipeline %s", n.Pipeline.Name)
	}
	n.Pipeline = *pip
	for i := range n.Triggers {
		t := &n.Triggers[i]
		if errR := resyncNode(db, &t.WorkflowDestNode); errR != nil {
			return sdk.WrapError(errR, "resyncNode> Cannot resync node %s", n.Name)
		}
	}
	return nil
}

//ResyncWorkflowRunStatus resync the status of workflow if you stop a node run when workflow run is building
func ResyncWorkflowRunStatus(db gorp.SqlExecutor, wr *sdk.WorkflowRun) error {
	var success, building, failed, stopped int
	for _, wnrs := range wr.WorkflowNodeRuns {
		for _, wnr := range wnrs {
			if wr.LastSubNumber == wnr.SubNumber {
				updateNodesRunStatus(wnr.Status, &success, &building, &failed, &stopped)
			}
		}
	}

	var isInError bool
	var newStatus string
	for _, info := range wr.Infos {
		if info.IsError {
			isInError = true
			break
		}
	}

	if !isInError {
		newStatus = getWorkflowRunStatus(success, building, failed, stopped)
	}

	if newStatus != wr.Status {
		wr.Status = newStatus
		return UpdateWorkflowRunStatus(db, wr.ID, newStatus)
	}

	return nil
}
