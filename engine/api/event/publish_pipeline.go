package event

import (
	"context"
	"fmt"
	"time"

	"github.com/fatih/structs"

	"github.com/ovh/cds/sdk"
)

// PublishPipelineEvent publish pipeline event
func publishPipelineEvent(ctx context.Context, payload interface{}, key string, pipName string, u sdk.Identifiable) {
	event := sdk.Event{
		Timestamp:    time.Now(),
		Hostname:     hostname,
		CDSName:      cdsname,
		EventType:    fmt.Sprintf("%T", payload),
		Payload:      structs.Map(payload),
		ProjectKey:   key,
		PipelineName: pipName,
	}
	if u != nil {
		event.UserMail = u.GetEmail()
		event.Username = u.GetUsername()
	}
	publishEvent(ctx, event)
}

// PublishPipelineAdd publishes an event for the creation of the given pipeline
func PublishPipelineAdd(ctx context.Context, key string, pip sdk.Pipeline, u sdk.Identifiable) {
	e := sdk.EventPipelineAdd{
		Pipeline: pip,
	}
	publishPipelineEvent(ctx, e, key, pip.Name, u)
}

// PublishPipelineUpdate publishes an event for the modification of the pipeline
func PublishPipelineUpdate(ctx context.Context, key string, newName string, oldName string, u sdk.Identifiable) {
	e := sdk.EventPipelineUpdate{
		NewName: newName,
		OldName: oldName,
	}
	publishPipelineEvent(ctx, e, key, newName, u)
}

// PublishPipelineDelete publishes an event for the deletion of the pipeline
func PublishPipelineDelete(ctx context.Context, key string, pip sdk.Pipeline, u sdk.Identifiable) {
	e := sdk.EventPipelineDelete{}
	publishPipelineEvent(ctx, e, key, pip.Name, u)
}

// PublishPipelineParameterAdd publishes an event on adding a pipeline parameter
func PublishPipelineParameterAdd(ctx context.Context, key string, pipName string, p sdk.Parameter, u sdk.Identifiable) {
	e := sdk.EventPipelineParameterAdd{
		Parameter: p,
	}
	publishPipelineEvent(ctx, e, key, pipName, u)
}

// PublishPipelineParameterUpdate publishes an event on editing a pipeline parameter
func PublishPipelineParameterUpdate(ctx context.Context, key string, pipName string, oldP sdk.Parameter, p sdk.Parameter, u sdk.Identifiable) {
	e := sdk.EventPipelineParameterUpdate{
		OldParameter: oldP,
		NewParameter: p,
	}
	publishPipelineEvent(ctx, e, key, pipName, u)
}

// PublishPipelineParameterDelete publishes an event on deleting a pipeline parameter
func PublishPipelineParameterDelete(ctx context.Context, key string, pipName string, p sdk.Parameter, u sdk.Identifiable) {
	e := sdk.EventPipelineParameterDelete{
		Parameter: p,
	}
	publishPipelineEvent(ctx, e, key, pipName, u)
}

// PublishPipelineStageAdd publishes an event on adding a stage
func PublishPipelineStageAdd(ctx context.Context, key string, pipName string, s sdk.Stage, u sdk.Identifiable) {
	e := sdk.EventPipelineStageAdd{
		Name:         s.Name,
		BuildOrder:   s.BuildOrder,
		Enabled:      s.Enabled,
		Prerequisite: s.Prerequisites,
	}

	publishPipelineEvent(ctx, e, key, pipName, u)
}

// PublishPipelineStageMove publishes an event on moving a stage
func PublishPipelineStageMove(ctx context.Context, key string, pipName string, s sdk.Stage, oldBuildOrder int, u sdk.Identifiable) {
	e := sdk.EventPipelineStageMove{
		StageName:          s.Name,
		StageID:            s.ID,
		NewStageBuildOrder: s.BuildOrder,
		OldStageBuildOrder: oldBuildOrder,
	}
	publishPipelineEvent(ctx, e, key, pipName, u)
}

// PublishPipelineStageUpdate publishes an event on updating a stage
func PublishPipelineStageUpdate(ctx context.Context, key string, pipName string, oldStage sdk.Stage, newStage sdk.Stage, u sdk.Identifiable) {
	e := sdk.EventPipelineStageUpdate{
		OldName:         oldStage.Name,
		OldBuildOrder:   oldStage.BuildOrder,
		OldEnabled:      oldStage.Enabled,
		OldPrerequisite: oldStage.Prerequisites,
		NewName:         newStage.Name,
		NewBuildOrder:   newStage.BuildOrder,
		NewEnabled:      newStage.Enabled,
		NewPrerequisite: newStage.Prerequisites,
	}
	publishPipelineEvent(ctx, e, key, pipName, u)
}

// PublishPipelineStageDelete publishes an event on deleting a stage
func PublishPipelineStageDelete(ctx context.Context, key string, pipName string, s sdk.Stage, u sdk.Identifiable) {
	e := sdk.EventPipelineStageDelete{
		ID:         s.ID,
		Name:       s.Name,
		BuildOrder: s.BuildOrder,
	}
	publishPipelineEvent(ctx, e, key, pipName, u)
}

// PublishPipelineJobAdd publishes an event on adding a job
func PublishPipelineJobAdd(ctx context.Context, key string, pipName string, s sdk.Stage, j sdk.Job, u sdk.Identifiable) {
	e := sdk.EventPipelineJobAdd{
		StageID:         s.ID,
		StageName:       s.Name,
		StageBuildOrder: s.BuildOrder,
		Job:             j,
	}
	publishPipelineEvent(ctx, e, key, pipName, u)
}

// PublishPipelineJobUpdate publishes an event on updating a job
func PublishPipelineJobUpdate(ctx context.Context, key string, pipName string, s sdk.Stage, oldJob sdk.Job, newJob sdk.Job, u sdk.Identifiable) {
	e := sdk.EventPipelineJobUpdate{
		StageID:         s.ID,
		StageName:       s.Name,
		StageBuildOrder: s.BuildOrder,
		OldJob:          oldJob,
		NewJob:          newJob,
	}
	publishPipelineEvent(ctx, e, key, pipName, u)
}

// PublishPipelineJobDelete publishes an event on deleting a job
func PublishPipelineJobDelete(ctx context.Context, key string, pipName string, s sdk.Stage, j sdk.Job, u sdk.Identifiable) {
	e := sdk.EventPipelineJobDelete{
		StageID:         s.ID,
		StageName:       s.Name,
		StageBuildOrder: s.BuildOrder,
		JobName:         j.Action.Name,
	}
	publishPipelineEvent(ctx, e, key, pipName, u)
}
