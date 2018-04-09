package event

import (
	"fmt"
	"time"

	"github.com/fatih/structs"

	"github.com/ovh/cds/sdk"
)

// PublishEnvironmentEvent publish Environment event
func publishEnvironmentEvent(payload interface{}, key, envName string, u *sdk.User) {
	event := sdk.Event{
		Timestamp:       time.Now(),
		Hostname:        hostname,
		CDSName:         cdsname,
		EventType:       fmt.Sprintf("%T", payload),
		Payload:         structs.Map(payload),
		ProjectKey:      key,
		EnvironmentName: envName,
	}
	if u != nil {
		event.Username = u.Username
		event.UserMail = u.Email
	}
	publishEvent(event)
}

// PublishEnvironmentAdd publishes an event for the creation of the given environment
func PublishEnvironmentAdd(projKey string, env sdk.Environment, u *sdk.User) {
	e := sdk.EventEnvironmentAdd{
		env,
	}
	publishEnvironmentEvent(e, projKey, env.Name, u)
}

// PublishEnvironmentUpdate publishes an event for the update of the given Environment
func PublishEnvironmentUpdate(projKey string, env sdk.Environment, oldenv sdk.Environment, u *sdk.User) {
	e := sdk.EventEnvironmentUpdate{
		NewName: env.Name,
		OldName: oldenv.Name,
	}
	publishEnvironmentEvent(e, projKey, env.Name, u)
}

// PublishEnvironmentDelete publishes an event for the deletion of the given Environment
func PublishEnvironmentDelete(projKey string, env sdk.Environment, u *sdk.User) {
	e := sdk.EventEnvironmentDelete{}
	publishEnvironmentEvent(e, projKey, env.Name, u)
}

// PublishEnvironmentVariableAdd publishes an event when adding a new variable
func PublishEnvironmentVariableAdd(projKey string, env sdk.Environment, v sdk.Variable, u *sdk.User) {
	if sdk.NeedPlaceholder(v.Type) {
		v.Value = sdk.PasswordPlaceholder
	}
	e := sdk.EventEnvironmentVariableAdd{
		Variable: v,
	}
	publishEnvironmentEvent(e, projKey, env.Name, u)
}

// PublishEnvironmentVariableUpdate publishes an event when updating a variable
func PublishEnvironmentVariableUpdate(projKey string, env sdk.Environment, v sdk.Variable, vOld sdk.Variable, u *sdk.User) {
	e := sdk.EventEnvironmentVariableUpdate{
		OldVariable: vOld,
		NewVariable: v,
	}
	publishEnvironmentEvent(e, projKey, env.Name, u)
}

// PublishEnvironmentVariableDelete publishes an event when deleting a new variable
func PublishEnvironmentVariableDelete(projKey string, env sdk.Environment, v sdk.Variable, u *sdk.User) {
	e := sdk.EventEnvironmentVariableDelete{
		Variable: v,
	}
	publishEnvironmentEvent(e, projKey, env.Name, u)
}

// PublishEnvironmentPermissionAdd publishes an event when adding a permission on the given environment
func PublishEnvironmentPermissionAdd(projKey string, env sdk.Environment, gp sdk.GroupPermission, u *sdk.User) {
	e := sdk.EventEnvironmentPermissionAdd{
		Permission: gp,
	}
	publishEnvironmentEvent(e, projKey, env.Name, u)
}

// PublishEnvironmentPermissionUpdate publishes an event when updating a permission on the given environment
func PublishEnvironmentPermissionUpdate(projKey string, env sdk.Environment, gp sdk.GroupPermission, gpOld sdk.GroupPermission, u *sdk.User) {
	e := sdk.EventEnvironmentPermissionUpdate{
		NewPermission: gp,
		OldPermission: gpOld,
	}
	publishEnvironmentEvent(e, projKey, env.Name, u)
}

// PublishEnvironmentPermissionDelete publishes an event when deleting a permission on the given environment
func PublishEnvironmentPermissionDelete(projKey string, env sdk.Environment, gp sdk.GroupPermission, u *sdk.User) {
	e := sdk.EventEnvironmentPermissionDelete{
		Permission: gp,
	}
	publishEnvironmentEvent(e, projKey, env.Name, u)
}

// PublishEnvironmentKeyAdd publishes an event when adding a key on the given environment
func PublishEnvironmentKeyAdd(projKey string, env sdk.Environment, k sdk.EnvironmentKey, u *sdk.User) {
	e := sdk.EventEnvironmentKeyAdd{
		Key: k,
	}
	publishEnvironmentEvent(e, projKey, env.Name, u)
}

// PublishEnvironmentKeyDelete publishes an event when deleting a key on the given environment
func PublishEnvironmentKeyDelete(projKey string, env sdk.Environment, k sdk.EnvironmentKey, u *sdk.User) {
	e := sdk.EventEnvironmentKeyDelete{
		Key: k,
	}
	publishEnvironmentEvent(e, projKey, env.Name, u)
}
