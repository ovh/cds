package event

import (
	"fmt"
	"time"

	"github.com/fatih/structs"

	"github.com/ovh/cds/sdk"
)

func publishActionEvent(payload interface{}, u sdk.Identifiable) {
	event := sdk.Event{
		Timestamp: time.Now(),
		Hostname:  hostname,
		CDSName:   cdsname,
		EventType: fmt.Sprintf("%T", payload),
		Payload:   structs.Map(payload),
	}
	if u != nil {
		event.Username = u.GetUsername()
		event.UserMail = u.Email()
	}
	publishEvent(event)
}

// PublishActionAdd publishes an event for the creation of the given action.
func PublishActionAdd(a sdk.Action, u sdk.Identifiable) {
	a.FirstAudit = nil
	a.LastAudit = nil
	publishActionEvent(sdk.EventActionAdd{Action: a}, u)
}

// PublishActionUpdate publishes an event for the update of the given action.
func PublishActionUpdate(oldAction sdk.Action, newAction sdk.Action, u sdk.Identifiable) {
	oldAction.FirstAudit = nil
	oldAction.LastAudit = nil
	newAction.FirstAudit = nil
	newAction.LastAudit = nil
	publishActionEvent(sdk.EventActionUpdate{
		OldAction: oldAction,
		NewAction: newAction,
	}, u)
}
