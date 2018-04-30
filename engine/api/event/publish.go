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

func publishEvent(e sdk.Event) {

	Cache.Enqueue("events", e)
	// send to cache for cds repositories manager
	Cache.Enqueue("events_repositoriesmanager", e)
}

// Publish sends a event to a queue
//func Publish(event sdk.Event, eventType string) {
func Publish(payload interface{}, u *sdk.User) {
	p := structs.Map(payload)
	var projectKey, applicationName, pipelineName, environmentName, workflowName string
	if v, ok := p["ProjectKey"]; ok {
		projectKey = v.(string)
	}
	if v, ok := p["ApplicationName"]; ok {
		applicationName = v.(string)
	}
	if v, ok := p["PipelineName"]; ok {
		pipelineName = v.(string)
	}
	if v, ok := p["EnvironmentName"]; ok {
		environmentName = v.(string)
	}
	if v, ok := p["WorkflowName"]; ok {
		workflowName = v.(string)
	}

	event := sdk.Event{
		Timestamp:       time.Now(),
		Hostname:        hostname,
		CDSName:         cdsname,
		EventType:       fmt.Sprintf("%T", payload),
		Payload:         p,
		ProjectKey:      projectKey,
		ApplicationName: applicationName,
		PipelineName:    pipelineName,
		EnvironmentName: environmentName,
		WorkflowName:    workflowName,
	}
	if u != nil {
		event.Username = u.Username
		event.UserMail = u.Email
	}
	publishEvent(event)
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

	Publish(e, nil)
}

// PublishPipelineBuild sends a pipelineBuild event
func PublishPipelineBuild(db gorp.SqlExecutor, pb *sdk.PipelineBuild, previous *sdk.PipelineBuild) {
	// get and send all user notifications
	for _, event := range notification.GetUserEvents(db, pb, previous) {
		Publish(event, nil)
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

	Publish(e, nil)
}
