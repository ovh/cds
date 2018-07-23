package event

import (
	"fmt"
	"time"

	"github.com/fatih/structs"
	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/notification"
	"github.com/ovh/cds/sdk"
)

func publishRunWorkflow(payload interface{}, key, workflowName, appName, pipName, envName string, num int64, sub int64, status string, u *sdk.User) {
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
		LastExecution: wr.LastExecution.Unix(),
		LastModified:  wr.LastModified.Unix(),
		Tags:          wr.Tags,
	}
	publishRunWorkflow(e, projectKey, wr.Workflow.Name, "", "", "", wr.Number, wr.LastSubNumber, wr.Status, nil)
}

// PublishWorkflowNodeRun publish event on a workflow node run
func PublishWorkflowNodeRun(db gorp.SqlExecutor, nr sdk.WorkflowNodeRun, w sdk.Workflow, previousWR *sdk.WorkflowNodeRun) {

	// get and send all user notifications
	if previousWR != nil {
		for _, event := range notification.GetUserWorkflowEvents(db, w, *previousWR, nr) {
			Publish(event, nil)
		}
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
	}

	for i := range nr.Stages {
		e.StagesSummary[i] = nr.Stages[i].ToSummary()
	}

	var pipName string
	node := w.GetNode(nr.WorkflowNodeID)
	if node != nil {
		pipName = w.Pipelines[node.PipelineID].Name
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
	if sdk.StatusIsTerminated(nr.Status) {
		e.Done = nr.Done.Unix()
	}
	publishRunWorkflow(e, w.ProjectKey, w.Name, appName, pipName, envName, nr.Number, nr.SubNumber, nr.Status, nil)
}
