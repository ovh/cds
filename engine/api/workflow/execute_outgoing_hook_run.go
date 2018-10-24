package workflow

import (
	"context"
	"fmt"
	"time"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/observability"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

// UpdateOutgoingHookRunStatus updates the status and callback of a outgoing hook run, and then it reprocess the whole workflow
func UpdateOutgoingHookRunStatus(ctx context.Context, db gorp.SqlExecutor, store cache.Store, proj *sdk.Project, wr *sdk.WorkflowRun, hookRunID string, callback sdk.WorkflowNodeOutgoingHookRunCallback) (*ProcessorReport, error) {
	ctx, end := observability.Span(ctx, "workflow.UpdateOutgoingHookRunStatus")
	defer end()

	report := new(ProcessorReport)

	//Checking if the hook is still at status waiting or building
	pendingOutgoingHooks := wr.PendingOutgoingHook()
	nodeRun, ok := pendingOutgoingHooks[hookRunID]
	if !ok {
		return nil, sdk.WrapError(sdk.ErrNotFound, "Unable to find node run")
	}

	nodeRun.Status = callback.Status
	nodeRun.Callback = &callback

	if sdk.StatusIsTerminated(nodeRun.Status) {
		nodeRun.Done = time.Now()
	}

	if err := UpdateNodeRun(db, nodeRun); err != nil {
		return nil, sdk.WrapError(err, "UpdateOutgoingHookRunStatus> Unable to update callback for outgoing node run")
	}

	report.Add(nodeRun)

	if wr.Version < 2 {
		report1, _, err := processWorkflowRun(ctx, db, store, proj, wr, nil, nil, nil)
		report.Merge(report1, err) //nolint
		if err != nil {
			return nil, sdk.WrapError(err, "Unable to process workflow run")
		}
	} else {
		mapNodes := wr.Workflow.WorkflowData.Maps()
		node := wr.Workflow.WorkflowData.NodeByID(nodeRun.WorkflowNodeID)

		report1, err := processNodeOutGoingHook(ctx, db, store, proj, wr, mapNodes, nil, node, int(nodeRun.SubNumber))
		report.Merge(report1, err) //nolint
		if err != nil {
			return nil, sdk.WrapError(err, "Unable to processNodeOutGoingHook")
		}
	}

	r1, err := computeAndUpdateWorkflowRunStatus(ctx, db, wr)
	if err != nil {
		return report, sdk.WrapError(err, "processNodeOutGoingHook> Unable to compute workflow run status")
	}
	report.Merge(r1, nil) // nolint
	return report, nil

}

// UpdateParentWorkflowRun updates the workflow which triggered the current workflow
func UpdateParentWorkflowRun(ctx context.Context, dbFunc func() *gorp.DbMap, store cache.Store, wr *sdk.WorkflowRun, parentProj *sdk.Project, parentWR *sdk.WorkflowRun) error {
	_, end := observability.Span(ctx, "workflow.UpdateParentWorkflowRun")
	defer end()

	// If the root node has been triggered by a parent workflow we have to update the parent workflow
	// and the outgoing hook callback

	if !sdk.StatusIsTerminated(wr.Status) {
		return nil
	}

	if !wr.HasParentWorkflow() {
		return nil
	}

	tx, err := dbFunc().Begin()
	if err != nil {
		return sdk.WrapError(err, "Unable to start transaction")
	}

	defer tx.Rollback() //nolint

	hookrun := parentWR.GetOutgoingHookRun(wr.RootRun().HookEvent.ParentWorkflow.HookRunID)
	if hookrun == nil {
		return sdk.WrapError(sdk.ErrNotFound, "unable to find hookrun")
	}

	if hookrun.Callback == nil {
		hookrun.Callback = new(sdk.WorkflowNodeOutgoingHookRunCallback)
		hookrun.Callback.Start = time.Now()
	}

	hookrun.Callback.Done = time.Now()
	hookrun.Callback.Log += fmt.Sprintf("\nWorkflow finished with status %s", wr.Status)
	hookrun.Callback.Status = wr.Status
	hookrun.Callback.WorkflowRunNumber = &wr.Number

	report, err := UpdateOutgoingHookRunStatus(ctx, tx, store, parentProj, parentWR, wr.RootRun().HookEvent.ParentWorkflow.HookRunID, *hookrun.Callback)
	if err != nil {
		log.Error("workflow.UpdateParentWorkflowRun> unable to update hook run status run %s/%s#%d: %v",
			parentProj.Key,
			parentWR.Workflow.Name,
			wr.RootRun().HookEvent.ParentWorkflow.Run,
			err)
		return sdk.WrapError(err, "Unable to update outgoing hook run status")
	}

	if err := tx.Commit(); err != nil {
		return sdk.WrapError(err, "Unable to commit transaction")
	}

	go SendEvent(dbFunc(), parentProj.Key, report)

	return nil
}
