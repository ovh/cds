package event

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/ovh/cds/sdk"
)

// PublishApplicationEvent publish application event
func publishApplicationEvent(ctx context.Context, payload interface{}, key, appName string, u sdk.Identifiable) {
	bts, _ := json.Marshal(payload)
	event := sdk.Event{
		Timestamp:       time.Now(),
		Hostname:        hostname,
		CDSName:         cdsname,
		EventType:       fmt.Sprintf("%T", payload),
		Payload:         bts,
		ProjectKey:      key,
		ApplicationName: appName,
	}
	if u != nil {
		event.Username = u.GetUsername()
		event.UserMail = u.GetEmail()
	}
	_ = publishEvent(ctx, event)
}

// PublishAddApplication publishes an event for the creation of the given application
func PublishAddApplication(ctx context.Context, projKey string, app sdk.Application, u sdk.Identifiable) {
	e := sdk.EventApplicationAdd{
		Application: app,
	}
	publishApplicationEvent(ctx, e, projKey, app.Name, u)
}

// PublishUpdateApplication publishes an event for the update of the given application
func PublishUpdateApplication(ctx context.Context, projKey string, app sdk.Application, oldApp sdk.Application, u sdk.Identifiable) {
	e := sdk.EventApplicationUpdate{
		NewMetadata:           app.Metadata,
		NewRepositoryStrategy: app.RepositoryStrategy,
		NewName:               app.Name,
		OldMetadata:           oldApp.Metadata,
		OldRepositoryStrategy: oldApp.RepositoryStrategy,
		OldName:               oldApp.Name,
	}
	publishApplicationEvent(ctx, e, projKey, app.Name, u)
}

// PublishDeleteApplication publishes an event for the deletion of the given application
func PublishDeleteApplication(ctx context.Context, projKey string, app sdk.Application, u sdk.Identifiable) {
	e := sdk.EventApplicationDelete{}
	publishApplicationEvent(ctx, e, projKey, app.Name, u)
}

// PublishAddVariableApplication publishes an event when adding a new variable
func PublishAddVariableApplication(ctx context.Context, projKey string, app sdk.Application, v sdk.ApplicationVariable, u sdk.Identifiable) {
	if sdk.NeedPlaceholder(v.Type) {
		v.Value = sdk.PasswordPlaceholder
	}
	e := sdk.EventApplicationVariableAdd{
		Variable: v,
	}
	publishApplicationEvent(ctx, e, projKey, app.Name, u)
}

// PublishUpdateVariableApplication publishes an event when updating a variable
func PublishUpdateVariableApplication(ctx context.Context, projKey string, app sdk.Application, v sdk.ApplicationVariable, vOld sdk.ApplicationVariable, u sdk.Identifiable) {
	e := sdk.EventApplicationVariableUpdate{
		OldVariable: vOld,
		NewVariable: v,
	}
	publishApplicationEvent(ctx, e, projKey, app.Name, u)
}

// PublishDeleteVariableApplication publishes an event when deleting a new variable
func PublishDeleteVariableApplication(ctx context.Context, projKey string, app sdk.Application, v sdk.ApplicationVariable, u sdk.Identifiable) {
	e := sdk.EventApplicationVariableDelete{
		Variable: v,
	}
	publishApplicationEvent(ctx, e, projKey, app.Name, u)
}

func PublishApplicationKeyAdd(ctx context.Context, projKey string, app sdk.Application, k sdk.ApplicationKey, u sdk.Identifiable) {
	e := sdk.EventApplicationKeyAdd{
		Key: k,
	}
	publishApplicationEvent(ctx, e, projKey, app.Name, u)
}

func PublishApplicationKeyDelete(ctx context.Context, projKey string, app sdk.Application, k sdk.ApplicationKey, u sdk.Identifiable) {
	e := sdk.EventApplicationKeyDelete{
		Key: k,
	}
	publishApplicationEvent(ctx, e, projKey, app.Name, u)
}

// PublishApplicationRepositoryAdd publishes an envet when adding a repository to an application
func PublishApplicationRepositoryAdd(ctx context.Context, projKey string, app sdk.Application, u sdk.Identifiable) {
	e := sdk.EventApplicationRepositoryAdd{
		VCSServer:  app.VCSServer,
		Repository: app.RepositoryFullname,
	}
	publishApplicationEvent(ctx, e, projKey, app.Name, u)
}

// PublishApplicationRepositoryDelete publishes an envet when deleting a repository from an application
func PublishApplicationRepositoryDelete(ctx context.Context, projKey string, appName string, vcsServer string, repository string, u sdk.Identifiable) {
	e := sdk.EventApplicationRepositoryDelete{
		VCSServer:  vcsServer,
		Repository: repository,
	}
	publishApplicationEvent(ctx, e, projKey, appName, u)
}
