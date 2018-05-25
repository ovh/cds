package event

import (
	"fmt"
	"time"

	"github.com/fatih/structs"

	"github.com/ovh/cds/sdk"
)

func publishBroadcastEvent(payload interface{}, key string, u *sdk.User) {
	p := structs.Map(payload)

	event := sdk.Event{
		Timestamp:  time.Now(),
		Hostname:   hostname,
		CDSName:    cdsname,
		EventType:  fmt.Sprintf("%T", payload),
		Payload:    p,
		ProjectKey: key,
	}
	if u != nil {
		event.Username = u.Username
		event.UserMail = u.Email
	}
	publishEvent(event)
}

// PublishBroadcastAdd publish event when adding a broadcast
func PublishBroadcastAdd(bc sdk.Broadcast, u *sdk.User) {
	e := sdk.EventBroadcastAdd{
		Broadcast: bc,
	}
	publishBroadcastEvent(e, bc.ProjectKey, u)
}

// PublishBroadcastUpdate publish event when updating a broadcast
func PublishBroadcastUpdate(oldBc sdk.Broadcast, bc sdk.Broadcast, u *sdk.User) {
	e := sdk.EventBroadcastUpdate{
		NewBroadcast: bc,
		OldBroadcast: oldBc,
	}
	publishBroadcastEvent(e, bc.ProjectKey, u)
}

// PublishBroadcastDelete publish event when deleting a broadcast
func PublishBroadcastDelete(id int64, u *sdk.User) {
	e := sdk.EventBroadcastDelete{
		BroadcastID: id,
	}
	publishBroadcastEvent(e, "", u)
}
