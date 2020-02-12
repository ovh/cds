package event

import (
	"context"
	"fmt"
	"time"

	"github.com/fatih/structs"
	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/notification"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func publishRunWorkflow(ctx context.Context, payload interface{}, key, workflowName, appName, pipName, envName string, num int64, sub int64, status string, tags []sdk.WorkflowRunTag, eventIntegrations []sdk.ProjectIntegration) {
	eventIntegrationsID := make([]int64, len(eventIntegrations))
	for i, eventIntegration := range eventIntegrations {
		eventIntegrationsID[i] = eventIntegration.ID
	}

	event := sdk.Event{
		Timestamp:           time.Now(),
		Hostname:            hostname,
		CDSName:             cdsname,
		EventType:           fmt.Sprintf("%T", payload),
		Payload:             structs.Map(payload),
		ProjectKey:          key,
		ApplicationName:     appName,
		PipelineName:        pipName,
		WorkflowName:        workflowName,
		EnvironmentName:     envName,
		WorkflowRunNum:      num,
		WorkflowRunNumSub:   sub,
		Status:              status,
		Tags:                tags,
		EventIntegrationsID: eventIntegrationsID,
	}
	publishEvent(ctx, event)
}

// PublishWorkflowRun publish event on a workflow run
func PublishWorkflowRun(ctx context.Context, wr sdk.WorkflowRun, projectKey string) {
	e := sdk.EventRunWorkflow{
		ID:               wr.ID,
		Number:           wr.Number,
		Status:           wr.Status,
		Start:            wr.Start.Unix(),
		LastExecution:    wr.LastExecution.Unix(),
		LastModified:     wr.LastModified.Unix(),
		LastModifiedNano: wr.LastModified.UnixNano(),
		Tags:             wr.Tags,
	}
	publishRunWorkflow(ctx, e, projectKey, wr.Workflow.Name, "", "", "", wr.Number, wr.LastSubNumber, wr.Status, wr.Tags, wr.Workflow.EventIntegrations)
}

// PublishWorkflowNodeRun publish event on a workflow node run
func PublishWorkflowNodeRun(ctx context.Context, db gorp.SqlExecutor, store cache.Store, nr sdk.WorkflowNodeRun, w sdk.Workflow, previousWR *sdk.WorkflowNodeRun) {
	// get and send all user notifications
	for _, event := range notification.GetUserWorkflowEvents(ctx, db, store, w, previousWR, nr) {
		Publish(ctx, event, nil)
	}

	e := sdk.EventRunWorkflowNode{
		ID:             nr.ID,
		Number:         nr.Number,
		SubNumber:      nr.SubNumber,
		Status:         nr.Status,
		Start:          nr.Start.Unix(),
		Manual:         nr.Manual,
		HookEvent:      nr.HookEvent,
		Payload:        nr.Payload,
		SourceNodeRuns: nr.SourceNodeRuns,
		Hash:           nr.VCSHash,
		BranchName:     nr.VCSBranch,
		NodeID:         nr.WorkflowNodeID,
		RunID:          nr.WorkflowRunID,
		StagesSummary:  make([]sdk.StageSummary, len(nr.Stages)),
		HookUUID:       nr.UUID,
	}

	if nr.Callback != nil {
		e.HookLog = nr.Callback.Log
	}

	for i := range nr.Stages {
		e.StagesSummary[i] = nr.Stages[i].ToSummary()
	}

	var pipName string
	var nodeName string
	var app sdk.Application
	var env sdk.Environment

	// check on workflow data
	wnode := w.WorkflowData.NodeByID(nr.WorkflowNodeID)
	if wnode == nil {
		log.Warning(ctx, "PublishWorkflowNodeRun> Unable to publish event on node %d", nr.WorkflowNodeID)
		return
	}
	nodeName = wnode.Name
	if wnode.Context != nil && wnode.Context.PipelineID != 0 {
		pipName = w.Pipelines[wnode.Context.PipelineID].Name
	}

	if wnode.Context != nil && wnode.Context.ApplicationID != 0 {
		app = w.Applications[wnode.Context.ApplicationID]
	}
	if wnode.Context != nil && wnode.Context.EnvironmentID != 0 {
		env = w.Environments[wnode.Context.EnvironmentID]
	}
	e.NodeType = wnode.Type

	// Try to get gerrit variable
	var project, changeID, branch, revision, url string
	projectParam := sdk.ParameterFind(nr.BuildParameters, "git.repository")
	if projectParam != nil {
		project = projectParam.Value
	}
	changeIDParam := sdk.ParameterFind(nr.BuildParameters, "gerrit.change.id")
	if changeIDParam != nil {
		changeID = changeIDParam.Value
	}
	branchParam := sdk.ParameterFind(nr.BuildParameters, "gerrit.change.branch")
	if branchParam != nil {
		branch = branchParam.Value
	}
	revisionParams := sdk.ParameterFind(nr.BuildParameters, "git.hash")
	if revisionParams != nil {
		revision = revisionParams.Value
	}
	urlParams := sdk.ParameterFind(nr.BuildParameters, "cds.ui.pipeline.run")
	if urlParams != nil {
		url = urlParams.Value
	}
	if changeID != "" && project != "" && branch != "" && revision != "" {
		e.GerritChange = &sdk.GerritChangeEvent{
			ID:         changeID,
			DestBranch: branch,
			Project:    project,
			Revision:   revision,
			URL:        url,
		}
	}

	e.NodeName = nodeName
	var envName string
	var appName string
	if app.ID != 0 {
		appName = app.Name
		e.RepositoryManagerName = app.VCSServer
		e.RepositoryFullName = app.RepositoryFullname
	}
	if env.ID != 0 {
		envName = env.Name
	}
	if sdk.StatusIsTerminated(nr.Status) {
		e.Done = nr.Done.Unix()
	}
	publishRunWorkflow(ctx, e, w.ProjectKey, w.Name, appName, pipName, envName, nr.Number, nr.SubNumber, nr.Status, nil, w.EventIntegrations)
}

// PublishWorkflowNodeJobRun publish a WorkflowNodeJobRun
func PublishWorkflowNodeJobRun(ctx context.Context, db gorp.SqlExecutor, pkey string, wr sdk.WorkflowRun, jr sdk.WorkflowNodeJobRun) {
	e := sdk.EventRunWorkflowJob{
		ID:     jr.ID,
		Status: jr.Status,
		Start:  jr.Start.Unix(),
	}

	if sdk.StatusIsTerminated(jr.Status) {
		e.Done = jr.Done.Unix()
	}
	publishRunWorkflow(ctx, e, pkey, wr.Workflow.Name, "", "", "", 0, 0, jr.Status, nil, wr.Workflow.EventIntegrations)
}
