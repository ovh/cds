package event

import (
	"fmt"
	"time"

	"github.com/fatih/structs"

	"github.com/ovh/cds/sdk"
)

// PublishWarningEvent publish application event
func PublishWarningEvent(payload interface{}, key, appName, pipName, envName, workflowName string, u *sdk.User) {
	event := sdk.Event{
		Timestamp:       time.Now(),
		Hostname:        hostname,
		CDSName:         cdsname,
		EventType:       fmt.Sprintf("%T", payload),
		Payload:         structs.Map(payload),
		ProjectKey:      key,
		ApplicationName: appName,
		PipelineName:    pipName,
		EnvironmentName: envName,
		WorkflowName:    workflowName,
	}
	if u != nil {
		event.Username = u.Username
		event.UserMail = u.Email
	}
	publishEvent(event)
}

// PublishAddWarning publishes an event for the creation of the given warning
func PublishAddWarning(w sdk.Warning) {
	e := sdk.EventWarningAdd{
		Warning: w,
	}
	PublishWarningEvent(e, w.Key, w.AppName, w.PipName, w.EnvName, w.WorkflowName, nil)
}

// PublishUpdateWarning publishes an event for the edition of the given warning
func PublishUpdateWarning(w sdk.Warning, u *sdk.User) {
	e := sdk.EventWarningUpdate{
		Warning: w,
	}
	PublishWarningEvent(e, w.Key, w.AppName, w.PipName, w.EnvName, w.WorkflowName, u)
}

// PublishDeleteWarning publishes an event for the deletion of the given warning
func PublishDeleteWarning(t string, element string, projectKey, appName, pipName, envName, workflowName string) {
	e := sdk.EventWarningDelete{
		Type:    t,
		Element: element,
	}
	PublishWarningEvent(e, projectKey, appName, pipName, envName, workflowName, nil)
}
