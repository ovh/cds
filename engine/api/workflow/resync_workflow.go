package workflow

import (
	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

// Resync a workflow in the given workflow run
func Resync(db gorp.SqlExecutor, store cache.Store, proj *sdk.Project, wr *sdk.WorkflowRun, u *sdk.User) error {
	options := LoadOptions{
		DeepPipeline: true,
		Base64Keys:   true,
	}
	wf, errW := LoadByID(db, store, proj, wr.Workflow.ID, u, options)
	if errW != nil {
		return sdk.WrapError(errW, "Resync> Cannot load workflow")
	}

	if err := resyncNode(wr.Workflow.Root, *wf); err != nil {
		return err
	}

	for i := range wr.Workflow.Joins {
		join := &wr.Workflow.Joins[i]
		for j := range join.Triggers {
			t := &join.Triggers[j]
			if err := resyncNode(&t.WorkflowDestNode, *wf); err != nil {
				return err
			}
		}
	}

	return UpdateWorkflowRun(nil, db, wr)
}

func resyncNode(node *sdk.WorkflowNode, newWorkflow sdk.Workflow) error {
	newNode := newWorkflow.GetNode(node.ID)
	if newNode == nil {
		newNode = newWorkflow.GetNodeByName(node.Name)
	}
	if newNode == nil {
		return sdk.ErrWorkflowNodeNotFound
	}

	node.Name = newNode.Name
	node.Context = newNode.Context
	node.Pipeline = newNode.Pipeline

	for i := range node.Triggers {
		t := &node.Triggers[i]
		if err := resyncNode(&t.WorkflowDestNode, newWorkflow); err != nil {
			return err
		}
	}
	return nil
}

//ResyncWorkflowRunStatus resync the status of workflow if you stop a node run when workflow run is building
func ResyncWorkflowRunStatus(db gorp.SqlExecutor, wr *sdk.WorkflowRun) (*ProcessorReport, error) {
	report := new(ProcessorReport)
	var success, building, failed, stopped, skipped, disabled int
	for _, wnrs := range wr.WorkflowNodeRuns {
		for _, wnr := range wnrs {
			if wr.LastSubNumber == wnr.SubNumber {
				computeRunStatus(wnr.Status, &success, &building, &failed, &stopped, &skipped, &disabled)
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
		newStatus = getRunStatus(success, building, failed, stopped, skipped, disabled)
	}

	if newStatus != wr.Status {
		wr.Status = newStatus
		report.Add(*wr)

		return report, UpdateWorkflowRunStatus(db, wr)
	}

	return report, nil
}

// ResyncNodeRunsWithCommits load commits build in this node run and save it into node run
func ResyncNodeRunsWithCommits(db gorp.SqlExecutor, store cache.Store, proj *sdk.Project, nodeRuns []sdk.WorkflowNodeRun) {
	for _, nodeRun := range nodeRuns {
		if len(nodeRun.Commits) > 0 {
			continue
		}
		go func(nr sdk.WorkflowNodeRun) {
			wr, errL := LoadRunByID(db, nr.WorkflowRunID, LoadRunOptions{})
			if errL != nil {
				log.Error("ResyncNodeRuns> Unable to load workflowRun by id %d : %v", nr.WorkflowRunID, errL)
				return
			}

			n := wr.Workflow.GetNode(nr.WorkflowNodeID)
			if n == nil {
				log.Error("ResyncNodeRuns> Unable to find node by id %d in a workflow run id %d", nr.WorkflowNodeID, nr.WorkflowRunID)
				return
			}

			if n.Context == nil || n.Context.Application == nil {
				return
			}

			commits, curVCSInfos, err := GetNodeRunBuildCommits(db, store, proj, &wr.Workflow, n.Name, wr.Number, &nr, n.Context.Application, n.Context.Environment)
			if err != nil {
				log.Error("ResyncNodeRuns> cannot get build commits on a node run %v", err)
			} else if commits != nil {
				nr.Commits = commits
			}

			if len(commits) > 0 {
				if err := updateNodeRunCommits(db, nr.ID, commits); err != nil {
					log.Error("ResyncNodeRuns> Unable to update node run commits %v", err)
				}
			}

			tagsUpdated := false
			if curVCSInfos.Branch != "" {
				tagsUpdated = wr.Tag(tagGitBranch, curVCSInfos.Branch)
			}
			if curVCSInfos.Hash != "" {
				tagsUpdated = wr.Tag(tagGitHash, curVCSInfos.Hash)
			}
			if curVCSInfos.Remote != "" {
				tagsUpdated = wr.Tag(tagGitRepository, curVCSInfos.Remote)
			}

			if tagsUpdated {
				if err := UpdateWorkflowRunTags(db, wr); err != nil {
					log.Error("ResyncNodeRuns> Unable to update workflow run tags %v", err)
				}
			}
		}(nodeRun)
	}
}
