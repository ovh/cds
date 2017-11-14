package workflow

import (
	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/engine/api/cache"
)

// Resync a workflow in the given workflow run
func Resync(db gorp.SqlExecutor, store cache.Store, wr *sdk.WorkflowRun, u *sdk.User) error {
	wf, errW := LoadByID(db, store, wr.Workflow.ID, u)
	if errW != nil {
		return sdk.WrapError(errW, "Resync> Cannot load workflow")
	}
	wr.Workflow = *wf
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
func ResyncWorkflowRunStatus(db gorp.SqlExecutor, wr *sdk.WorkflowRun, chEvent chan<- interface{}) error {
	var success, building, failed, stopped int
	for _, wnrs := range wr.WorkflowNodeRuns {
		for _, wnr := range wnrs {
			if wr.LastSubNumber == wnr.SubNumber {
				computeNodesRunStatus(wnr.Status, &success, &building, &failed, &stopped)
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
		chEvent <- wr
		return UpdateWorkflowRunStatus(db, wr.ID, newStatus)
	}

	return nil
}
