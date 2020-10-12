package event

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

type publishWorkflowRunData struct {
	projectKey        string
	workflowName      string
	applicationName   string
	pipelineName      string
	environmentName   string
	workflowRunNum    int64
	workflowRunSubNum int64
	status            string
	workflowRunTags   []sdk.WorkflowRunTag
	eventIntegrations []sdk.ProjectIntegration
	workflowNodeRunID int64
}

func publishRunWorkflow(ctx context.Context, payload interface{}, data publishWorkflowRunData) {
	eventIntegrationsID := make([]int64, len(data.eventIntegrations))
	for i, eventIntegration := range data.eventIntegrations {
		eventIntegrationsID[i] = eventIntegration.ID
	}

	bts, _ := json.Marshal(payload)
	event := sdk.Event{
		Timestamp:           time.Now(),
		Hostname:            hostname,
		CDSName:             cdsname,
		EventType:           fmt.Sprintf("%T", payload),
		Payload:             bts,
		ProjectKey:          data.projectKey,
		ApplicationName:     data.applicationName,
		PipelineName:        data.pipelineName,
		WorkflowName:        data.workflowName,
		EnvironmentName:     data.environmentName,
		WorkflowRunNum:      data.workflowRunNum,
		WorkflowRunNumSub:   data.workflowRunSubNum,
		WorkflowNodeRunID:   data.workflowNodeRunID,
		Status:              data.status,
		Tags:                data.workflowRunTags,
		EventIntegrationsID: eventIntegrationsID,
	}
	_ = publishEvent(ctx, event)
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
		ToDelete:         wr.ToDelete,
	}
	data := publishWorkflowRunData{
		projectKey:        projectKey,
		workflowName:      wr.Workflow.Name,
		workflowRunNum:    wr.Number,
		workflowRunSubNum: wr.LastSubNumber,
		status:            wr.Status,
		workflowRunTags:   wr.Tags,
		eventIntegrations: wr.Workflow.EventIntegrations,
	}
	publishRunWorkflow(ctx, e, data)
}

// PublishWorkflowNodeRun publish event on a workflow node run
func PublishWorkflowNodeRun(ctx context.Context, nr sdk.WorkflowNodeRun, w sdk.Workflow, userWorkflowEvent []sdk.EventNotif) {
	// get and send all user notifications
	for _, event := range userWorkflowEvent {
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
	data := publishWorkflowRunData{
		projectKey:        w.ProjectKey,
		workflowName:      w.Name,
		applicationName:   appName,
		pipelineName:      pipName,
		environmentName:   envName,
		workflowRunNum:    nr.Number,
		workflowRunSubNum: nr.SubNumber,
		status:            nr.Status,
		eventIntegrations: w.EventIntegrations,
		workflowNodeRunID: nr.ID,
	}
	publishRunWorkflow(ctx, e, data)
}

// PublishWorkflowNodeJobRun publish a WorkflowNodeJobRun
func PublishWorkflowNodeJobRun(ctx context.Context, pkey string, wr sdk.WorkflowRun, jr sdk.WorkflowNodeJobRun) {
	e := sdk.EventRunWorkflowJob{
		ID:     jr.ID,
		Status: jr.Status,
		Start:  jr.Start.Unix(),
	}

	if sdk.StatusIsTerminated(jr.Status) {
		e.Done = jr.Done.Unix()
	}
	data := publishWorkflowRunData{
		projectKey:        pkey,
		workflowName:      wr.Workflow.Name,
		workflowRunNum:    wr.Number,
		workflowRunSubNum: wr.LastSubNumber,
		status:            jr.Status,
		workflowRunTags:   wr.Tags,
		eventIntegrations: wr.Workflow.EventIntegrations,
		workflowNodeRunID: jr.WorkflowNodeRunID,
	}
	publishRunWorkflow(ctx, e, data)
}
