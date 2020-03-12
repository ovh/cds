package event

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/ovh/cds/sdk"
)

// PublishEnvironmentEvent publish Environment event
func publishEnvironmentEvent(ctx context.Context, payload interface{}, key, envName string, u sdk.Identifiable) {
	bts, _ := json.Marshal(payload)
	event := sdk.Event{
		Timestamp:       time.Now(),
		Hostname:        hostname,
		CDSName:         cdsname,
		EventType:       fmt.Sprintf("%T", payload),
		Payload:         bts,
		ProjectKey:      key,
		EnvironmentName: envName,
	}
	if u != nil {
		event.Username = u.GetUsername()
		event.UserMail = u.GetEmail()
	}
	publishEvent(ctx, event)
}

// PublishEnvironmentAdd publishes an event for the creation of the given environment
func PublishEnvironmentAdd(ctx context.Context, projKey string, env sdk.Environment, u sdk.Identifiable) {
	e := sdk.EventEnvironmentAdd{
		Environment: env,
	}
	publishEnvironmentEvent(ctx, e, projKey, env.Name, u)
}

// PublishEnvironmentUpdate publishes an event for the update of the given Environment
func PublishEnvironmentUpdate(ctx context.Context, projKey string, env sdk.Environment, oldenv sdk.Environment, u sdk.Identifiable) {
	e := sdk.EventEnvironmentUpdate{
		NewName: env.Name,
		OldName: oldenv.Name,
	}
	publishEnvironmentEvent(ctx, e, projKey, env.Name, u)
}

// PublishEnvironmentDelete publishes an event for the deletion of the given Environment
func PublishEnvironmentDelete(ctx context.Context, projKey string, env sdk.Environment, u sdk.Identifiable) {
	e := sdk.EventEnvironmentDelete{}
	publishEnvironmentEvent(ctx, e, projKey, env.Name, u)
}

// PublishEnvironmentVariableAdd publishes an event when adding a new variable
func PublishEnvironmentVariableAdd(ctx context.Context, projKey string, env sdk.Environment, v sdk.Variable, u sdk.Identifiable) {
	if sdk.NeedPlaceholder(v.Type) {
		v.Value = sdk.PasswordPlaceholder
	}
	e := sdk.EventEnvironmentVariableAdd{
		Variable: v,
	}
	publishEnvironmentEvent(ctx, e, projKey, env.Name, u)
}

// PublishEnvironmentVariableUpdate publishes an event when updating a variable
func PublishEnvironmentVariableUpdate(ctx context.Context, projKey string, env sdk.Environment, v sdk.Variable, vOld sdk.Variable, u sdk.Identifiable) {
	e := sdk.EventEnvironmentVariableUpdate{
		OldVariable: vOld,
		NewVariable: v,
	}
	publishEnvironmentEvent(ctx, e, projKey, env.Name, u)
}

// PublishEnvironmentVariableDelete publishes an event when deleting a new variable
func PublishEnvironmentVariableDelete(ctx context.Context, projKey string, env sdk.Environment, v sdk.Variable, u sdk.Identifiable) {
	e := sdk.EventEnvironmentVariableDelete{
		Variable: v,
	}
	publishEnvironmentEvent(ctx, e, projKey, env.Name, u)
}

// PublishEnvironmentKeyAdd publishes an event when adding a key on the given environment
func PublishEnvironmentKeyAdd(ctx context.Context, projKey string, env sdk.Environment, k sdk.EnvironmentKey, u sdk.Identifiable) {
	e := sdk.EventEnvironmentKeyAdd{
		Key: k,
	}
	publishEnvironmentEvent(ctx, e, projKey, env.Name, u)
}

// PublishEnvironmentKeyDelete publishes an event when deleting a key on the given environment
func PublishEnvironmentKeyDelete(ctx context.Context, projKey string, env sdk.Environment, k sdk.EnvironmentKey, u sdk.Identifiable) {
	e := sdk.EventEnvironmentKeyDelete{
		Key: k,
	}
	publishEnvironmentEvent(ctx, e, projKey, env.Name, u)
}
