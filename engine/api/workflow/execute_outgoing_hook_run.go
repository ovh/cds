package workflow

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ovh/cds/engine/api/observability"

	"github.com/go-gorp/gorp"
	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func UpdateOutgoingHookRunStatus(ctx context.Context, db gorp.SqlExecutor, store cache.Store, proj *sdk.Project, wr *sdk.WorkflowRun, hookRunID string, callback sdk.WorkflowNodeOutgoingHookRunCallback) (*ProcessorReport, error) {
	ctx, end := observability.Span(ctx, "workflow.UpdateOutgoingHookRunStatus")
	defer end()

	report := new(ProcessorReport)

	//Checking if the hook is still at status waiting or building
	var hookRun = wr.GetOutgoingHookRun(hookRunID)
	if hookRun.Status != sdk.StatusWaiting.String() && hookRun.Status != sdk.StatusBuilding.String() {
		log.Debug("UpdateOutgoingHookRunStatus> hookRun status is %s. aborting", hookRun.Status)
		hookRun = nil
	}

	if hookRun == nil {
		return nil, sdk.ErrNotFound
	}

	hookRun.Status = callback.Status
	hookRun.Callback = &callback

	report.Add(hookRun)
	report1, _, err := processWorkflowRun(ctx, db, store, proj, wr, nil, nil, nil)
	report.Merge(report1, err) //nolint
	if err != nil {
		return nil, err
	}

	if err := UpdateWorkflowRun(ctx, db, wr); err != nil {
		return nil, err
	}

	return report, nil

}

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
		return err
	}

	defer tx.Rollback()

	hookrun := parentWR.GetOutgoingHookRun(wr.RootRun().HookEvent.ParentWorkflow.HookRunID)
	if hookrun == nil {
		return errors.New("unable to find hookrun")
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
		log.Error("workflow.UpdateWorkflowRun> unable to update hook run status run %s/%s#%d: %v",
			parentProj.Key,
			parentWR.Workflow.Name,
			wr.RootRun().HookEvent.ParentWorkflow.Run,
			err)
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	go SendEvent(dbFunc(), parentProj.Key, report)

	return nil
}
