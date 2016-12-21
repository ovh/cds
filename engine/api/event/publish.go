package event

import (
	"fmt"
	"time"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/database"
	"github.com/ovh/cds/engine/api/notification"
	"github.com/ovh/cds/engine/log"
	"github.com/ovh/cds/sdk"

	"github.com/fatih/structs"
)

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

	log.Debug("Publish> new event %+v", event)
	cache.Enqueue("events", event)
}

// PublishActionBuild sends a actionBuild event
func PublishActionBuild(pb *sdk.PipelineBuild, ab *sdk.ActionBuild) {
	e := sdk.EventJob{
		Version:         pb.Version,
		JobName:         ab.ActionName,
		Status:          ab.Status,
		Queued:          ab.Queued,
		Start:           ab.Start,
		Done:            ab.Done,
		Model:           ab.Model,
		PipelineName:    pb.Pipeline.Name,
		PipelineType:    pb.Pipeline.Type,
		ProjectKey:      pb.Pipeline.ProjectKey,
		ApplicationName: pb.Application.Name,
		EnvironmentName: pb.Environment.Name,
		BranchName:      getBranch(pb),
	}

	Publish(e)
}

// PublishPipelineBuild sends a pipelineBuild event
func PublishPipelineBuild(db database.QueryExecuter, pb *sdk.PipelineBuild, previous *sdk.PipelineBuild) {
	// get and send all user notifications
	for _, event := range notification.GetUserEvents(db, pb, previous) {
		Publish(event)
	}

	e := sdk.EventPipelineBuild{
		Version:         pb.Version,
		Status:          pb.Status,
		Start:           pb.Start,
		Done:            pb.Done,
		PipelineName:    pb.Pipeline.Name,
		PipelineType:    pb.Pipeline.Type,
		ProjectKey:      pb.Pipeline.ProjectKey,
		ApplicationName: pb.Application.Name,
		EnvironmentName: pb.Environment.Name,
		BranchName:      getBranch(pb),
	}

	Publish(e)
}

func getBranch(pb *sdk.PipelineBuild) string {
	branch := ""
	for _, param := range pb.Parameters {
		if param.Name == ".git.branch" {
			branch = param.Value
			break
		}
	}
	return branch
}
