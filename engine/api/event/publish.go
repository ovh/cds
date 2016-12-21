package event

import (
	"fmt"
	"time"

	"github.com/fatih/structs"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/database"
	"github.com/ovh/cds/engine/api/notification"
	"github.com/ovh/cds/engine/log"
	"github.com/ovh/cds/sdk"
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
		Queued:          ab.Queued.Unix(),
		Start:           ab.Start.Unix(),
		Done:            ab.Done.Unix(),
		Model:           ab.Model,
		PipelineName:    pb.Pipeline.Name,
		PipelineType:    pb.Pipeline.Type,
		ProjectKey:      pb.Pipeline.ProjectKey,
		ApplicationName: pb.Application.Name,
		EnvironmentName: pb.Environment.Name,
		BranchName:      getParamValue(".git.hash", pb),
		Hash:            getParamValue(".git.hash", pb),
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
		BuildNumber:     pb.BuildNumber,
		Status:          pb.Status,
		Start:           pb.Start.Unix(),
		Done:            pb.Done.Unix(),
		PipelineName:    pb.Pipeline.Name,
		PipelineType:    pb.Pipeline.Type,
		ProjectKey:      pb.Pipeline.ProjectKey,
		ApplicationName: pb.Application.Name,
		EnvironmentName: pb.Environment.Name,
		BranchName:      getParamValue(".git.branch", pb),
		Hash:            getParamValue(".git.hash", pb),
	}

	Publish(e)
}

func getParamValue(paramName string, pb *sdk.PipelineBuild) string {
	branch := ""
	for _, param := range pb.Parameters {
		if param.Name == paramName {
			branch = param.Value
			break
		}
	}
	return branch
}
