package event

import (
	"fmt"
	"time"

	"github.com/fatih/structs"
	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/notification"
	"github.com/ovh/cds/sdk"
)

func publishRunWorkflow(payload interface{}, key, workflowName, appName, pipName, envName string, num int64, sub int64, status string) {
	event := sdk.Event{
		Timestamp:         time.Now(),
		Hostname:          hostname,
		CDSName:           cdsname,
		EventType:         fmt.Sprintf("%T", payload),
		Payload:           structs.Map(payload),
		ProjectKey:        key,
		ApplicationName:   appName,
		PipelineName:      pipName,
		WorkflowName:      workflowName,
		EnvironmentName:   envName,
		WorkflowRunNum:    num,
		WorkflowRunNumSub: sub,
		Status:            status,
	}
	publishEvent(event)
}

// PublishWorkflowRun publish event on a workflow run
func PublishWorkflowRun(wr sdk.WorkflowRun, projectKey string) {
	e := sdk.EventRunWorkflow{
		ID:            wr.ID,
		Number:        wr.Number,
		Status:        wr.Status,
		Start:         wr.Start.Unix(),
		LastExecution: wr.LastExecution.Unix(),
		LastModified:  wr.LastModified.Unix(),
		Tags:          wr.Tags,
	}
	publishRunWorkflow(e, projectKey, wr.Workflow.Name, "", "", "", wr.Number, wr.LastSubNumber, wr.Status)
}

// PublishWorkflowNodeRun publish event on a workflow node run
func PublishWorkflowNodeRun(db gorp.SqlExecutor, nr sdk.WorkflowNodeRun, w sdk.Workflow, previousWR *sdk.WorkflowNodeRun) {
	// get and send all user notifications
	for _, event := range notification.GetUserWorkflowEvents(db, w, previousWR, nr) {
		Publish(event, nil)
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
	n := w.GetNode(nr.WorkflowNodeID)
	if n == nil {
		// check on workflow data
		wnode := w.WorkflowData.NodeByID(nr.WorkflowNodeID)
		if wnode == nil {
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
	} else {
		nodeName = n.Name
		pipName = w.Pipelines[n.PipelineID].Name
		if n.Context != nil && n.Context.Application != nil {
			app = *n.Context.Application
		}
		if n.Context != nil && n.Context.Environment != nil {
			env = *n.Context.Environment
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
	publishRunWorkflow(e, w.ProjectKey, w.Name, appName, pipName, envName, nr.Number, nr.SubNumber, nr.Status)
}
