package event

import (
	"fmt"
	"time"

	"github.com/fatih/structs"

	"github.com/ovh/cds/sdk"
)

// PublishPipelineEvent publish pipeline event
func publishPipelineEvent(payload interface{}, key string, pipName string, u *sdk.User) {
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
		event.UserMail = u.Email
		event.Username = u.Username
	}
	publishEvent(event)
}

// PublishPipelineAdd publishes an event for the creation of the given pipeline
func PublishPipelineAdd(key string, pip sdk.Pipeline, u *sdk.User) {
	e := sdk.EventPipelineAdd{
		Pipeline: pip,
	}
	publishPipelineEvent(e, key, pip.Name, u)
}

// PublishPipelineUpdate publishes an event for the modification of the pipeline
func PublishPipelineUpdate(key string, newName string, oldName string, u *sdk.User) {
	e := sdk.EventPipelineUpdate{
		NewName: newName,
		OldName: oldName,
	}
	publishPipelineEvent(e, key, newName, u)
}

// PublishPipelineDelete publishes an event for the deletion of the pipeline
func PublishPipelineDelete(key string, pip sdk.Pipeline, u *sdk.User) {
	e := sdk.EventPipelineDelete{}
	publishPipelineEvent(e, key, pip.Name, u)
}

// PublishPipelinePermissionAdd publishes an event for pipeline permission adding
func PublishPipelinePermissionAdd(key string, pipName string, gp sdk.GroupPermission, u *sdk.User) {
	e := sdk.EventPipelinePermissionAdd{
		gp,
	}
	publishPipelineEvent(e, key, pipName, u)
}

// PublishPipelinePermissionUpdate publishes an event for pipeline permission update
func PublishPipelinePermissionUpdate(key string, pipName string, oldGp sdk.GroupPermission, gp sdk.GroupPermission, u *sdk.User) {
	e := sdk.EventPipelinePermissionUpdate{
		OldPermission: oldGp,
		NewPermission: gp,
	}
	publishPipelineEvent(e, key, pipName, u)
}

// PublishPipelinePermissionDelete publishes an event for pipeline permission deletion
func PublishPipelinePermissionDelete(key string, pipName string, gp sdk.GroupPermission, u *sdk.User) {
	e := sdk.EventPipelinePermissionDelete{
		Permission: gp,
	}
	publishPipelineEvent(e, key, pipName, u)
}

// PublishPipelineParameterAdd publishes an event on adding a pipeline parameter
func PublishPipelineParameterAdd(key string, pipName string, p sdk.Parameter, u *sdk.User) {
	e := sdk.EventPipelineParameterAdd{
		Parameter: p,
	}
	publishPipelineEvent(e, key, pipName, u)
}

// PublishPipelineParameterUpdate publishes an event on editing a pipeline parameter
func PublishPipelineParameterUpdate(key string, pipName string, oldP sdk.Parameter, p sdk.Parameter, u *sdk.User) {
	e := sdk.EventPipelineParameterUpdate{
		OldParameter: oldP,
		NewParameter: p,
	}
	publishPipelineEvent(e, key, pipName, u)
}

// PublishPipelineParameterDelete publishes an event on deleting a pipeline parameter
func PublishPipelineParameterDelete(key string, pipName string, p sdk.Parameter, u *sdk.User) {
	e := sdk.EventPipelineParameterDelete{
		Parameter: p,
	}
	publishPipelineEvent(e, key, pipName, u)
}

// PublishPipelineStageAdd publishes an event on adding a stage
func PublishPipelineStageAdd(key string, pipName string, s sdk.Stage, u *sdk.User) {
	e := sdk.EventPipelineStageAdd{
		Name:         s.Name,
		BuildOrder:   s.BuildOrder,
		Enabled:      s.Enabled,
		Prerequisite: s.Prerequisites,
	}

	publishPipelineEvent(e, key, pipName, u)
}
