package api

import (
	"context"

	"github.com/go-gorp/gorp"
	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/event"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

// WorkflowSendEvent Send event on workflow run
func WorkflowSendEvent(ctx context.Context, db gorp.SqlExecutor, store cache.Store, key string, report *workflow.ProcessorReport) {
	if report == nil {
		return
	}
	for _, wr := range report.Workflows() {
		event.PublishWorkflowRun(ctx, wr, key)
	}
	for _, wnr := range report.Nodes() {
		wr, errWR := workflow.LoadRunByID(db, wnr.WorkflowRunID, workflow.LoadRunOptions{
			WithLightTests: true,
		})
		if errWR != nil {
			log.Warning(ctx, "WorkflowSendEvent> Cannot load workflow run %d: %s", wnr.WorkflowRunID, errWR)
			continue
		}

		var previousNodeRun sdk.WorkflowNodeRun
		if wnr.SubNumber > 0 {
			previousNodeRun = wnr
		} else {
			var errN error
			previousNodeRun, errN = workflow.PreviousNodeRun(db, wnr, wnr.WorkflowNodeName, wr.WorkflowID)
			if errN != nil {
				log.Warning(ctx, "WorkflowSendEvent> Cannot load previous node run: %s", errN)
			}
		}

		event.PublishWorkflowNodeRun(ctx, db, store, wnr, wr.Workflow, &previousNodeRun)
	}

	for _, jobrun := range report.Jobs() {
		noderun, err := workflow.LoadNodeRunByID(db, jobrun.WorkflowNodeRunID, workflow.LoadRunOptions{})
		if err != nil {
			log.Warning(ctx, "WorkflowSendEvent> Cannot load workflow node run %d: %s", jobrun.WorkflowNodeRunID, err)
			continue
		}
		wr, errWR := workflow.LoadRunByID(db, noderun.WorkflowRunID, workflow.LoadRunOptions{
			WithLightTests: true,
		})
		if errWR != nil {
			log.Warning(ctx, "WorkflowSendEvent> Cannot load workflow run %d: %s", noderun.WorkflowRunID, errWR)
			continue
		}
		event.PublishWorkflowNodeJobRun(ctx, db, key, *wr, jobrun)
	}
}
