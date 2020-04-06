package api

import (
	"context"

	"github.com/ovh/cds/engine/api/notification"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/event"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

// WorkflowSendEvent Send event on workflow run
func WorkflowSendEvent(ctx context.Context, db gorp.SqlExecutor, store cache.Store, proj sdk.Project, report *workflow.ProcessorReport) {
	if report == nil {
		return
	}
	for _, wr := range report.Workflows() {
		event.PublishWorkflowRun(ctx, wr, proj.Key)
	}
	for _, wnr := range report.Nodes() {
		wr, errWR := workflow.LoadRunByID(db, wnr.WorkflowRunID, workflow.LoadRunOptions{
			DisableDetailledNodeRun: true,
		})
		if errWR != nil {
			log.Warning(ctx, "workflowSendEvent> Cannot load workflow run %d: %s", wnr.WorkflowRunID, errWR)
			continue
		}

		var previousNodeRun sdk.WorkflowNodeRun
		if wnr.SubNumber > 0 {
			previousNodeRun = wnr
		} else {
			var errN error
			previousNodeRun, errN = workflow.PreviousNodeRun(db, wnr, wnr.WorkflowNodeName, wr.WorkflowID)
			if errN != nil {
				log.Warning(ctx, "workflowSendEvent> Cannot load previous node run: %v", errN)
			}
		}

		nr, err := workflow.LoadNodeRunByID(db, wnr.ID, workflow.LoadRunOptions{
			DisableDetailledNodeRun: false, // load build parameters, used in notif interpolate below
		})
		if err != nil {
			log.Warning(ctx, "workflowSendEvent > Cannot load workflow node run: %v", err)
			continue
		}

		event.PublishWorkflowNodeRun(ctx, *nr, wr.Workflow, notification.GetUserWorkflowEvents(ctx, db, store, wr.Workflow, &previousNodeRun, *nr))
		e := &workflow.VCSEventMessenger{}
		if err := e.SendVCSEvent(ctx, db, store, proj, *wr, wnr); err != nil {
			log.Warning(ctx, "WorkflowSendEvent> Cannot send vcs notification")
		}
	}

	for _, jobrun := range report.Jobs() {
		noderun, err := workflow.LoadNodeRunByID(db, jobrun.WorkflowNodeRunID, workflow.LoadRunOptions{})
		if err != nil {
			log.Warning(ctx, "workflowSendEvent> Cannot load workflow node run %d: %s", jobrun.WorkflowNodeRunID, err)
			continue
		}
		wr, errWR := workflow.LoadRunByID(db, noderun.WorkflowRunID, workflow.LoadRunOptions{
			WithLightTests: true,
		})
		if errWR != nil {
			log.Warning(ctx, "workflowSendEvent> Cannot load workflow run %d: %s", noderun.WorkflowRunID, errWR)
			continue
		}
		event.PublishWorkflowNodeJobRun(ctx, db, proj.Key, *wr, jobrun)
	}
}
