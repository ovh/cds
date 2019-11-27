package event

import (
	"context"
	"fmt"
	"time"

	"github.com/fatih/structs"

	"github.com/ovh/cds/sdk"
)

func publishActionEvent(ctx context.Context, payload interface{}, u sdk.Identifiable) {
	event := sdk.Event{
		Timestamp: time.Now(),
		Hostname:  hostname,
		CDSName:   cdsname,
		EventType: fmt.Sprintf("%T", payload),
		Payload:   structs.Map(payload),
	}
	if u != nil {
		event.Username = u.GetUsername()
		event.UserMail = u.GetEmail()
	}
	publishEvent(ctx, event)
}

// PublishActionAdd publishes an event for the creation of the given action.
func PublishActionAdd(ctx context.Context, a sdk.Action, u sdk.Identifiable) {
	a.FirstAudit = nil
	a.LastAudit = nil
	publishActionEvent(ctx, sdk.EventActionAdd{Action: a}, u)
}

// PublishActionUpdate publishes an event for the update of the given action.
func PublishActionUpdate(ctx context.Context, oldAction sdk.Action, newAction sdk.Action, u sdk.Identifiable) {
	oldAction.FirstAudit = nil
	oldAction.LastAudit = nil
	newAction.FirstAudit = nil
	newAction.LastAudit = nil
	publishActionEvent(ctx, sdk.EventActionUpdate{
		OldAction: oldAction,
		NewAction: newAction,
	}, u)
}
