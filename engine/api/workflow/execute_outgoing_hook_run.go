package workflow

import (
	"context"

	"github.com/ovh/cds/engine/api/observability"

	"github.com/go-gorp/gorp"
	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func UpdateOutgoingHookRunStatus(ctx context.Context, dbFunc func() *gorp.DbMap, db gorp.SqlExecutor, store cache.Store, proj *sdk.Project, wr *sdk.WorkflowRun, hookRunID string, callback sdk.WorkflowNodeOutgoingHookRunCallback) (*ProcessorReport, error) {
	ctx, end := observability.Span(ctx, "workflow.UpdateOutgoingHookRunStatus")
	defer end()

	report := new(ProcessorReport)

	//Checking if the hook is still at status waiting
	var hookRun *sdk.WorkflowNodeOutgoingHookRun
loop:
	for i := range wr.WorkflowNodeOutgoingHookRuns {
		hookRuns := wr.WorkflowNodeOutgoingHookRuns[i]
		for j := range hookRuns {
			hr := &hookRuns[j]
			log.Debug("UpdateOutgoingHookRunStatus> checking %s", hr.HookRunID)
			if hr.HookRunID == hookRunID && hr.Status == sdk.StatusWaiting.String() {
				hookRun = hr
				break loop
			}
		}
	}

	if hookRun == nil {
		return nil, sdk.ErrNotFound
	}

	hookRun.Status = callback.Status
	hookRun.Callback = &callback

	report.Add(hookRun)
	log.Debug("UpdateOutgoingHookRunStatus> hook run updated: %v", hookRun)

	report1, _, err := processWorkflowRun(ctx, db, store, proj, wr, nil, nil, nil)
	report.Merge(report1, err)
	if err != nil {
		return nil, err
	}

	if err := UpdateWorkflowRun(ctx, db, wr); err != nil {
		return nil, err
	}

	return report, nil

}
