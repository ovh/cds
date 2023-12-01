package api

import (
	"context"
	"fmt"

	"github.com/rockbears/log"

	"github.com/ovh/cds/engine/api/event"
	"github.com/ovh/cds/engine/api/notification"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/sdk"
	cdslog "github.com/ovh/cds/sdk/log"
)

// WorkflowSendEvent Send event on workflow run
func (api *API) WorkflowSendEvent(ctx context.Context, proj sdk.Project, report *workflow.ProcessorReport) {
	db := api.mustDB()
	if db == nil {
		return
	}

	if report == nil {
		return
	}
	for _, wr := range report.Workflows() {
		event.PublishWorkflowRun(ctx, wr, proj.Key)
	}
	for _, wnr := range report.Nodes() {
		wr, err := workflow.LoadRunByID(ctx, db, wnr.WorkflowRunID, workflow.LoadRunOptions{
			DisableDetailledNodeRun: true,
		})
		if err != nil {
			ctx := sdk.ContextWithStacktrace(ctx, err)
			log.Warn(ctx, "workflowSendEvent> Cannot load workflow run %d: %s", wnr.WorkflowRunID, err)
			continue
		}

		var previousNodeRun *sdk.WorkflowNodeRun
		if wnr.SubNumber > 0 {
			previousNodeRun = &wnr
		} else {
			previousNodeRun, err = workflow.PreviousNodeRun(db, wnr, wnr.WorkflowNodeName, wr.WorkflowID)
			if err != nil {
				ctx := sdk.ContextWithStacktrace(ctx, err)
				log.Warn(ctx, "workflowSendEvent> Cannot load previous node run: %v", err)
			}
		}

		nr, err := workflow.LoadNodeRunByID(ctx, db, wnr.ID, workflow.LoadRunOptions{
			DisableDetailledNodeRun: false, // load build parameters, used in notif interpolate below
		})
		if err != nil {
			ctx := sdk.ContextWithStacktrace(ctx, err)
			log.Warn(ctx, "workflowSendEvent > Cannot load workflow node run: %v", err)
			continue
		}
		ctx := context.WithValue(ctx, cdslog.NodeRunID, nr.ID)

		workDB, err := workflow.LoadWorkflowFromWorkflowRunID(db, wr.ID)
		if err != nil {
			ctx := sdk.ContextWithStacktrace(ctx, err)
			log.Warn(ctx, "WorkflowSendEvent> Unable to load workflow for event: %v", err)
			continue
		}
		eventsNotif := notification.GetUserWorkflowEvents(ctx, db, api.Cache, wr.Workflow.ProjectID, wr.Workflow.ProjectKey, workDB.Name, wr.Workflow.Notifications, previousNodeRun, *nr)
		event.PublishWorkflowNodeRun(ctx, *nr, wr.Workflow, eventsNotif)
		e := &workflow.VCSEventMessenger{}
		if err := e.SendVCSEvent(ctx, db, api.Cache, proj, *wr, wnr); err != nil {
			ctx := sdk.ContextWithStacktrace(ctx, err)
			log.Warn(ctx, "WorkflowSendEvent> Cannot send vcs notification err:%v", err)
		}
	}

	for _, jobrun := range report.Jobs() {
		ctx := context.WithValue(ctx, cdslog.PermJobID, jobrun.ID)

		noderun, err := workflow.LoadNodeRunByID(ctx, db, jobrun.WorkflowNodeRunID, workflow.LoadRunOptions{})
		if err != nil {
			ctx := sdk.ContextWithStacktrace(ctx, err)
			log.Warn(ctx, "workflowSendEvent> Cannot load workflow node run %d: %s", jobrun.WorkflowNodeRunID, err)
			continue
		}
		wr, err := workflow.LoadRunByID(ctx, db, noderun.WorkflowRunID, workflow.LoadRunOptions{
			WithLightTests: true,
		})
		if err != nil {
			ctx := sdk.ContextWithStacktrace(ctx, err)
			log.Warn(ctx, "workflowSendEvent> Cannot load workflow run %d: %s", noderun.WorkflowRunID, err)
			continue
		}
		event.PublishWorkflowNodeJobRun(ctx, proj.Key, *wr, jobrun)

		var ejs = NewEventJobSummary(*wr, *noderun, jobrun)
		event.PublishEventJobSummary(ctx, ejs, wr.Workflow.Integrations)
	}
}

func NewEventJobSummary(wr sdk.WorkflowRun, noderun sdk.WorkflowNodeRun, jobrun sdk.WorkflowNodeJobRun) sdk.EventJobSummary {
	var ejs = sdk.EventJobSummary{
		ID:                   jobrun.ID,
		ProjectKey:           wr.Workflow.ProjectKey,
		Workflow:             wr.Workflow.Name,
		WorkflowRunNumber:    int(noderun.Number),
		WorkflowRunSubNumber: int(noderun.SubNumber),
		Created:              &jobrun.Queued,
		CreatedHour:          jobrun.Queued.Hour(),
		Pipeline:             noderun.WorkflowNodeName,
		Job:                  jobrun.Job.Action.Name,
		GitVCS:               noderun.VCSServer,
		GitRepo:              noderun.VCSRepository,
		GitBranch:            noderun.VCSBranch,
		GitTag:               noderun.VCSTag,
		GitCommit:            noderun.VCSHash,
	}

	node := wr.Workflow.WorkflowData.NodeByID(noderun.WorkflowNodeID)
	if node != nil && node.Context != nil {
		ejs.PipelineName = node.Context.PipelineName
	}

	if wr.Version != nil {
		ejs.WorkflowRunVersion = *wr.Version
	} else {
		ejs.WorkflowRunVersion = fmt.Sprintf("%d", wr.Number)
	}

	if !jobrun.Start.IsZero() {
		ejs.Started = &jobrun.Start
		ejs.InQueueDuration = int(jobrun.Start.UnixMilli() - jobrun.Queued.UnixMilli())
		ejs.WorkerModel = jobrun.Model
		ejs.WorkerModelType = jobrun.ModelType
		ejs.Worker = jobrun.WorkerName
		ejs.Hatchery = jobrun.HatcheryName
		if jobrun.Region != nil {
			ejs.Region = *jobrun.Region
		}
		if noderun.HookEvent != nil {
			ejs.Hook = noderun.HookEvent.WorkflowNodeHookUUID
		}
	}

	if !jobrun.Done.IsZero() {
		ejs.Ended = &jobrun.Done
		ejs.TotalDuration = int(jobrun.Done.UnixMilli() - jobrun.Queued.UnixMilli())
		ejs.BuildDuration = int(jobrun.Done.UnixMilli() - jobrun.Start.UnixMilli())
		ejs.FinalStatus = jobrun.Status
	}

	return ejs
}
