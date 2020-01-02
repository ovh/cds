package event

import (
	"context"
	"fmt"
	"time"

	"github.com/fatih/structs"

	"github.com/ovh/cds/sdk"
)

// PublishWarningEvent publish application event
func PublishWarningEvent(ctx context.Context, payload interface{}, key, appName, pipName, envName, workflowName string, u sdk.Identifiable) {
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
		event.Username = u.GetUsername()
		event.UserMail = u.GetEmail()
	}
	publishEvent(ctx, event)
}

// PublishAddWarning publishes an event for the creation of the given warning
func PublishAddWarning(ctx context.Context, w sdk.Warning) {
	e := sdk.EventWarningAdd{
		Warning: w,
	}
	PublishWarningEvent(ctx, e, w.Key, w.AppName, w.PipName, w.EnvName, w.WorkflowName, nil)
}

// PublishUpdateWarning publishes an event for the edition of the given warning
func PublishUpdateWarning(ctx context.Context, w sdk.Warning, u sdk.Identifiable) {
	e := sdk.EventWarningUpdate{
		Warning: w,
	}
	PublishWarningEvent(ctx, e, w.Key, w.AppName, w.PipName, w.EnvName, w.WorkflowName, u)
}

// PublishDeleteWarning publishes an event for the deletion of the given warning
func PublishDeleteWarning(ctx context.Context, t string, element string, projectKey, appName, pipName, envName, workflowName string) {
	e := sdk.EventWarningDelete{
		Type:    t,
		Element: element,
	}
	PublishWarningEvent(ctx, e, projectKey, appName, pipName, envName, workflowName, nil)
}
