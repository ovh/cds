package event

import (
	"fmt"
	"time"

	"github.com/fatih/structs"

	"github.com/ovh/cds/sdk"
)

// PublishApplicationEvent publish application event
func publishApplicationEvent(payload interface{}, key, appName string, u *sdk.User) {
	event := sdk.Event{
		Timestamp:       time.Now(),
		Hostname:        hostname,
		CDSName:         cdsname,
		EventType:       fmt.Sprintf("%T", payload),
		Payload:         structs.Map(payload),
		ProjectKey:      key,
		ApplicationName: appName,
	}
	if u != nil {
		event.Username = u.Username
		event.UserMail = u.Email
	}
	publishEvent(event)
}

// PublishAddApplication publishes an event for the creation of the given application
func PublishAddApplication(projKey string, app sdk.Application, u *sdk.User) {
	e := sdk.EventApplicationAdd{
		Application: app,
	}
	publishApplicationEvent(e, projKey, app.Name, u)
}

// PublishUpdateApplication publishes an event for the update of the given application
func PublishUpdateApplication(projKey string, app sdk.Application, oldApp sdk.Application, u *sdk.User) {
	e := sdk.EventApplicationUpdate{
		NewMetadata:           app.Metadata,
		NewRepositoryStrategy: app.RepositoryStrategy,
		NewName:               app.Name,
		OldMetadata:           oldApp.Metadata,
		OldRepositoryStrategy: oldApp.RepositoryStrategy,
		OldName:               oldApp.Name,
	}
	publishApplicationEvent(e, projKey, app.Name, u)
}

// PublishDeleteApplication publishes an event for the deletion of the given application
func PublishDeleteApplication(projKey string, app sdk.Application, u *sdk.User) {
	e := sdk.EventApplicationDelete{}
	publishApplicationEvent(e, projKey, app.Name, u)
}

// PublishAddVariableApplication publishes an event when adding a new variable
func PublishAddVariableApplication(projKey string, app sdk.Application, v sdk.Variable, u *sdk.User) {
	if sdk.NeedPlaceholder(v.Type) {
		v.Value = sdk.PasswordPlaceholder
	}
	e := sdk.EventApplicationVariableAdd{
		Variable: v,
	}
	publishApplicationEvent(e, projKey, app.Name, u)
}

// PublishUpdateVariableApplication publishes an event when updating a variable
func PublishUpdateVariableApplication(projKey string, app sdk.Application, v sdk.Variable, vOld sdk.Variable, u *sdk.User) {
	e := sdk.EventApplicationVariableUpdate{
		OldVariable: vOld,
		NewVariable: v,
	}
	publishApplicationEvent(e, projKey, app.Name, u)
}

// PublishDeleteVariableApplication publishes an event when deleting a new variable
func PublishDeleteVariableApplication(projKey string, app sdk.Application, v sdk.Variable, u *sdk.User) {
	e := sdk.EventApplicationVariableDelete{
		Variable: v,
	}
	publishApplicationEvent(e, projKey, app.Name, u)
}

func PublishApplicationPermissionAdd(projKey string, app sdk.Application, gp sdk.GroupPermission, u *sdk.User) {
	e := sdk.EventApplicationPermissionAdd{
		Permission: gp,
	}
	publishApplicationEvent(e, projKey, app.Name, u)
}

func PublishApplicationPermissionUpdate(projKey string, app sdk.Application, gp sdk.GroupPermission, gpOld sdk.GroupPermission, u *sdk.User) {
	e := sdk.EventApplicationPermissionUpdate{
		NewPermission: gp,
		OldPermission: gpOld,
	}
	publishApplicationEvent(e, projKey, app.Name, u)
}

func PublishApplicationPermissionDelete(projKey string, app sdk.Application, gp sdk.GroupPermission, u *sdk.User) {
	e := sdk.EventApplicationPermissionDelete{
		Permission: gp,
	}
	publishApplicationEvent(e, projKey, app.Name, u)
}

func PublishApplicationKeyAdd(projKey string, app sdk.Application, k sdk.ApplicationKey, u *sdk.User) {
	e := sdk.EventApplicationKeyAdd{
		Key: k,
	}
	publishApplicationEvent(e, projKey, app.Name, u)
}

func PublishApplicationKeyDelete(projKey string, app sdk.Application, k sdk.ApplicationKey, u *sdk.User) {
	e := sdk.EventApplicationKeyDelete{
		Key: k,
	}
	publishApplicationEvent(e, projKey, app.Name, u)
}
