package event

import (
	"fmt"
	"time"

	"github.com/fatih/structs"
	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/notification"
	"github.com/ovh/cds/sdk"
)

var Cache cache.Store

// Publish sends a event to a queue
//func Publish(event sdk.Event, eventType string) {
func Publish(payload interface{}) {
	event := sdk.Event{
		Timestamp: time.Now(),
		Hostname:  hostname,
		CDSName:   cdsname,
		EventType: fmt.Sprintf("%T", payload),
		Payload:   structs.Map(payload),
	}

	Cache.Enqueue("events", event)
	// send to cache for cds repositories manager
	Cache.Enqueue("events_repositoriesmanager", event)
}

// PublishActionBuild sends a actionBuild event
func PublishActionBuild(pb *sdk.PipelineBuild, pbJob *sdk.PipelineBuildJob) {
	e := sdk.EventJob{
		Version:         pb.Version,
		JobName:         pbJob.Job.Action.Name,
		JobID:           pbJob.Job.PipelineActionID,
		Status:          sdk.StatusFromString(pbJob.Status),
		Queued:          pbJob.Queued.Unix(),
		Start:           pbJob.Start.Unix(),
		Done:            pbJob.Done.Unix(),
		ModelName:       pbJob.Model,
		PipelineName:    pb.Pipeline.Name,
		PipelineType:    pb.Pipeline.Type,
		ProjectKey:      pb.Pipeline.ProjectKey,
		ApplicationName: pb.Application.Name,
		EnvironmentName: pb.Environment.Name,
		BranchName:      pb.Trigger.VCSChangesBranch,
		Hash:            pb.Trigger.VCSChangesHash,
	}

	Publish(e)
}

// PublishPipelineBuild sends a pipelineBuild event
func PublishPipelineBuild(db gorp.SqlExecutor, pb *sdk.PipelineBuild, previous *sdk.PipelineBuild) {
	// get and send all user notifications
	for _, event := range notification.GetUserEvents(db, pb, previous) {
		Publish(event)
	}

	rmn := ""
	rfn := ""
	if pb.Application.VCSServer != "" {
		rmn = pb.Application.VCSServer
		rfn = pb.Application.RepositoryFullname
	}

	e := sdk.EventPipelineBuild{
		Version:     pb.Version,
		BuildNumber: pb.BuildNumber,
		Status:      pb.Status,
		Start:       pb.Start.Unix(),
		Done:        pb.Done.Unix(),
		RepositoryManagerName: rmn,
		RepositoryFullname:    rfn,
		PipelineName:          pb.Pipeline.Name,
		PipelineType:          pb.Pipeline.Type,
		ProjectKey:            pb.Pipeline.ProjectKey,
		ApplicationName:       pb.Application.Name,
		EnvironmentName:       pb.Environment.Name,
		BranchName:            pb.Trigger.VCSChangesBranch,
		Hash:                  pb.Trigger.VCSChangesHash,
	}

	Publish(e)
}

// PublishWorkflowRun publish event on a workflow run
func PublishWorkflowRun(wr sdk.WorkflowRun, projectKey string) {
	e := sdk.EventWorkflowRun{
		ID:           wr.ID,
		Number:       wr.Number,
		Status:       wr.Status,
		Start:        wr.Start.Unix(),
		ProjectKey:   projectKey,
		WorkflowName: wr.Workflow.Name,
		Workflow:     wr.Workflow,
	}
	Publish(e)
}

// PublishWorkflowNodeRun publish event on a workflow node run
func PublishWorkflowNodeRun(nr sdk.WorkflowNodeRun, wr sdk.WorkflowRun, projectKey string) {
	e := sdk.EventWorkflowNodeRun{
		ID:             nr.ID,
		Number:         nr.Number,
		SubNumber:      nr.SubNumber,
		Status:         nr.Status,
		Start:          nr.Start.Unix(),
		ProjectKey:     projectKey,
		Manual:         nr.Manual,
		HookEvent:      nr.HookEvent,
		Payload:        nr.Payload,
		SourceNodeRuns: nr.SourceNodeRuns,
	}

	node := wr.Workflow.GetNode(nr.WorkflowNodeID)
	if node != nil {
		e.PipelineName = node.Pipeline.Name
	}
	if node.Context != nil {
		if node.Context.Application != nil {
			e.ApplicationName = node.Context.Application.Name
			e.RepositoryManagerName = node.Context.Application.VCSServer
		}
		if node.Context.Environment != nil {
			e.EnvironmentName = node.Context.Environment.Name
		}
	}

	if nr.Status != sdk.StatusBuilding.String() && nr.Status != sdk.StatusWaiting.String() {
		e.Done = nr.Done.Unix()
	}
	Publish(e)
}

// EventWorkflowNodeJobRun publish event on a workflow node job run
func PublishWorkflowNodeJobRun(njr sdk.WorkflowNodeJobRun) {
	e := sdk.EventWorkflowNodeJobRun{
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
	Publish(e)
}
