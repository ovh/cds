package workflow

import (
	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/sdk"
)

// Resync a workflow in the given workflow run
func Resync(db gorp.SqlExecutor, store cache.Store, wr *sdk.WorkflowRun, u *sdk.User) error {
	wf, errW := LoadByID(db, store, wr.Workflow.ID, u)
	if errW != nil {
		return sdk.WrapError(errW, "Resync> Cannot load workflow")
	}

	resyncNode(wr.Workflow.Root, *wf)

	for i := range wr.Workflow.Joins {
		join := &wr.Workflow.Joins[i]
		for j := range join.Triggers {
			t := &join.Triggers[j]
			resyncNode(&t.WorkflowDestNode, *wf)
		}
	}

	return updateWorkflowRun(db, wr)
}

func resyncNode(node *sdk.WorkflowNode, newWorkflow sdk.Workflow) {
	newNode := newWorkflow.GetNode(node.ID)
	if newNode == nil {
		newNode = newWorkflow.GetNodeByName(node.Name)
	}
	if newNode != nil {
		node.Name = newNode.Name
		node.Context = newNode.Context
		node.Pipeline = newNode.Pipeline
	}

	for i := range node.Triggers {
		t := &node.Triggers[i]
		resyncNode(&t.WorkflowDestNode, newWorkflow)
	}

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
