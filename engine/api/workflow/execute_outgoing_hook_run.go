package workflow

import (
	"context"
	"fmt"
	"time"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
	"github.com/ovh/cds/sdk/telemetry"
)

// UpdateOutgoingHookRunStatus updates the status and callback of a outgoing hook run, and then it reprocess the whole workflow
func UpdateOutgoingHookRunStatus(ctx context.Context, db gorpmapper.SqlExecutorWithTx, store cache.Store, proj sdk.Project, wr *sdk.WorkflowRun, hookRunID string, callback sdk.WorkflowNodeOutgoingHookRunCallback) (*ProcessorReport, error) {
	ctx, end := telemetry.Span(ctx, "workflow.UpdateOutgoingHookRunStatus")
	defer end()

	report := new(ProcessorReport)

	//Checking if the hook is still at status waiting or building
	pendingOutgoingHooks := wr.PendingOutgoingHook()
	nr, ok := pendingOutgoingHooks[hookRunID]
	if !ok {
		return nil, sdk.WrapError(sdk.ErrNotFound, "Unable to find node run")
	}

	// Reload node run with build parameters
	nodeRun, err := LoadNodeRunByID(db, nr.ID, LoadRunOptions{})
	if err != nil {
		return nil, err
	}

	nodeRun.Status = callback.Status
	nodeRun.Callback = &callback

	if sdk.StatusIsTerminated(nodeRun.Status) {
		nodeRun.Done = time.Now()
	}

	if err := UpdateNodeRun(db, nodeRun); err != nil {
		return nil, sdk.WrapError(err, "UpdateOutgoingHookRunStatus> Unable to update callback for outgoing node run")
	}

	report.Add(ctx, nodeRun)

	mapNodes := wr.Workflow.WorkflowData.Maps()
	node := wr.Workflow.WorkflowData.NodeByID(nodeRun.WorkflowNodeID)

loop:
	for i := range wr.WorkflowNodeRuns {
		nrs := wr.WorkflowNodeRuns[i]
		for j := range nrs {
			nr := nrs[j]
			if nr.ID == nodeRun.ID {
				nrs[j] = *nodeRun
				break loop
			}
		}
	}

	report1, _, err := processNodeOutGoingHook(ctx, db, store, proj, wr, mapNodes, nil, node, int(nodeRun.SubNumber), nil)
	report.Merge(ctx, report1)
	if err != nil {
		return nil, sdk.WrapError(err, "unable to processNodeOutGoingHook")
	}

	oldStatus := wr.Status
	r1, err := computeAndUpdateWorkflowRunStatus(ctx, db, wr)
	if err != nil {
		return report, sdk.WrapError(err, "unable to compute workflow run status")
	}
	report.Merge(ctx, r1)
	if wr.Status != oldStatus {
		report.Add(ctx, wr)
	}
	return report, nil
}

// UpdateParentWorkflowRun updates the workflow which triggered the current workflow
func UpdateParentWorkflowRun(ctx context.Context, dbFunc func() *gorp.DbMap, store cache.Store, wr *sdk.WorkflowRun, parentProj sdk.Project, parentWR *sdk.WorkflowRun) (*ProcessorReport, error) {
	_, end := telemetry.Span(ctx, "workflow.UpdateParentWorkflowRun")
	defer end()

	// If the root node has been triggered by a parent workflow we have to update the parent workflow
	// and the outgoing hook callback

	if !sdk.StatusIsTerminated(wr.Status) {
		return nil, nil
	}

	if !wr.HasParentWorkflow() {
		return nil, nil
	}

	tx, err := dbFunc().Begin()
	if err != nil {
		return nil, sdk.WrapError(err, "Unable to start transaction")
	}

	defer tx.Rollback() //nolint

	hookrun := parentWR.GetOutgoingHookRun(wr.RootRun().HookEvent.ParentWorkflow.HookRunID)
	if hookrun == nil {
		return nil, sdk.WrapError(sdk.ErrNotFound, "unable to find hookrun")
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
		log.Error(ctx, "workflow.UpdateParentWorkflowRun> unable to update hook run status run %s/%s#%d: %v",
			parentProj.Key,
			parentWR.Workflow.Name,
			wr.RootRun().HookEvent.ParentWorkflow.Run,
			err)
		return nil, sdk.WrapError(err, "Unable to update outgoing hook run status")
	}

	if err := tx.Commit(); err != nil {
		return nil, sdk.WrapError(err, "Unable to commit transaction")
	}

	return report, nil
}
