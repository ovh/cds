package api

import (
	"context"

	"github.com/rockbears/log"

	"github.com/ovh/cds/engine/api/event"
	"github.com/ovh/cds/engine/api/notification"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/sdk"
)

// WorkflowSendEvent Send event on workflow run
func (api *API) WorkflowSendEvent(ctx context.Context, proj sdk.Project, report *workflow.ProcessorReport) {
	if report == nil {
		return
	}
	for _, wr := range report.Workflows() {
		event.PublishWorkflowRun(ctx, wr, proj.Key)
	}
	for _, wnr := range report.Nodes() {
		wr, err := workflow.LoadRunByID(ctx, api.mustDB(), wnr.WorkflowRunID, workflow.LoadRunOptions{
			DisableDetailledNodeRun: true,
		})
		if err != nil {
			log.Warn(ctx, "workflowSendEvent> Cannot load workflow run %d: %s", wnr.WorkflowRunID, err)
			continue
		}

		var previousNodeRun *sdk.WorkflowNodeRun
		if wnr.SubNumber > 0 {
			previousNodeRun = &wnr
		} else {
			previousNodeRun, err = workflow.PreviousNodeRun(api.mustDB(), wnr, wnr.WorkflowNodeName, wr.WorkflowID)
			if err != nil {
				log.Warn(ctx, "workflowSendEvent> Cannot load previous node run: %v", err)
			}
		}

		nr, err := workflow.LoadNodeRunByID(ctx, api.mustDB(), wnr.ID, workflow.LoadRunOptions{
			DisableDetailledNodeRun: false, // load build parameters, used in notif interpolate below
		})
		if err != nil {
			log.Warn(ctx, "workflowSendEvent > Cannot load workflow node run: %v", err)
			continue
		}

		workDB, err := workflow.LoadWorkflowFromWorkflowRunID(api.mustDB(), wr.ID)
		if err != nil {
			log.Warn(ctx, "WorkflowSendEvent> Unable to load workflow for event: %v", err)
			continue
		}
		eventsNotif := notification.GetUserWorkflowEvents(ctx, api.mustDB(), api.Cache, wr.Workflow.ProjectID, wr.Workflow.ProjectKey, workDB.Name, wr.Workflow.Notifications, previousNodeRun, *nr)
		event.PublishWorkflowNodeRun(ctx, *nr, wr.Workflow, eventsNotif)
		e := &workflow.VCSEventMessenger{}
		if err := e.SendVCSEvent(ctx, api.mustDB(), api.Cache, proj, *wr, wnr); err != nil {
			log.Warn(ctx, "WorkflowSendEvent> Cannot send vcs notification")
		}
	}

	for _, jobrun := range report.Jobs() {
		noderun, err := workflow.LoadNodeRunByID(ctx, api.mustDB(), jobrun.WorkflowNodeRunID, workflow.LoadRunOptions{})
		if err != nil {
			log.Warn(ctx, "workflowSendEvent> Cannot load workflow node run %d: %s", jobrun.WorkflowNodeRunID, err)
			continue
		}
		wr, err := workflow.LoadRunByID(ctx, api.mustDB(), noderun.WorkflowRunID, workflow.LoadRunOptions{
			WithLightTests: true,
		})
		if err != nil {
			log.Warn(ctx, "workflowSendEvent> Cannot load workflow run %d: %s", noderun.WorkflowRunID, err)
			continue
		}
		event.PublishWorkflowNodeJobRun(ctx, proj.Key, *wr, jobrun)
	}
}
