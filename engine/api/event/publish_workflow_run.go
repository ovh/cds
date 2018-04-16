package event

import (
	"fmt"
	"time"

	"github.com/fatih/structs"
	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/notification"
	"github.com/ovh/cds/sdk"
)

// PublishRunWorkflow
func PublishRunWorkflow(payload interface{}, key, workflowName, appName, pipName, envName string, num int64, u *sdk.User) {
	event := sdk.Event{
		Timestamp:       time.Now(),
		Hostname:        hostname,
		CDSName:         cdsname,
		EventType:       fmt.Sprintf("%T", payload),
		Payload:         structs.Map(payload),
		ProjectKey:      key,
		ApplicationName: appName,
		PipelineName:    pipName,
		WorkflowName:    workflowName,
		EnvironmentName: envName,
		WorkflowRunNum:  num,
	}
	if u != nil {
		event.Username = u.Username
		event.UserMail = u.Email
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
		Workflow:      wr.Workflow,
		LastExecution: wr.LastExecution.Unix(),
		LastModified:  wr.LastModified.Unix(),
		Tags:          wr.Tags,
	}
	PublishRunWorkflow(e, projectKey, wr.Workflow.Name, "", "", "", wr.Number, nil)
}

// PublishWorkflowNodeRun publish event on a workflow node run
func PublishWorkflowNodeRun(db gorp.SqlExecutor, nr sdk.WorkflowNodeRun, wr sdk.WorkflowRun, previousWR sdk.WorkflowNodeRun, projectKey string) {
	// get and send all user notifications
	for _, event := range notification.GetUserWorkflowEvents(db, wr, previousWR, nr) {
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
	}

	var pipName string
	node := wr.Workflow.GetNode(nr.WorkflowNodeID)
	if node != nil {
		pipName = node.Pipeline.Name
		e.NodeName = node.Name
	}
	var envName string
	var appName string
	if node.Context != nil {
		if node.Context.Application != nil {
			appName = node.Context.Application.Name
			e.RepositoryManagerName = node.Context.Application.VCSServer
			e.RepositoryFullName = node.Context.Application.RepositoryFullname
		}
		if node.Context.Environment != nil {
			envName = node.Context.Environment.Name
		}
	}

	if nr.Status != sdk.StatusBuilding.String() && nr.Status != sdk.StatusWaiting.String() {
		e.Done = nr.Done.Unix()
	}
	PublishRunWorkflow(e, projectKey, wr.Workflow.Name, appName, pipName, envName, wr.Number, nil)
}

// PublishWorkflowNodeJobRun publish event on a workflow node job run
func PublishWorkflowNodeJobRun(prokectKey string, njr sdk.WorkflowNodeJobRun, wnr sdk.WorkflowNodeRun, wr sdk.WorkflowRun) {
	e := sdk.EventRunWorkflowNodeJob{
		ID:                njr.ID,
		Status:            njr.Status,
		WorkflowNodeRunID: njr.WorkflowNodeRunID,
		Start:             njr.Start.Unix(),
		Model:             njr.Model,
		Queued:            njr.Queued.Unix(),
	}
	if njr.Status != sdk.StatusBuilding.String() && njr.Status != sdk.StatusWaiting.String() {
		e.Done = njr.Done.Unix()
	}

	var pipName string
	node := wr.Workflow.GetNode(wnr.WorkflowNodeID)
	if node != nil {
		pipName = node.Pipeline.Name
	}
	var envName string
	var appName string
	if node.Context != nil {
		if node.Context.Application != nil {
			appName = node.Context.Application.Name
		}
		if node.Context.Environment != nil {
			envName = node.Context.Environment.Name
		}
	}

	PublishRunWorkflow(e, prokectKey, wr.Workflow.Name, appName, pipName, envName, wr.Number, nil)
}
