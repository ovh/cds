package event

import (
	"context"
	"fmt"
	"time"

	"github.com/fatih/structs"

	"github.com/ovh/cds/sdk"
)

func publishBroadcastEvent(ctx context.Context, payload interface{}, key string, u sdk.Identifiable) {
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
		event.Username = u.GetUsername()
		event.UserMail = u.GetEmail()
	}
	publishEvent(ctx, event)
}

// PublishBroadcastAdd publish event when adding a broadcast
func PublishBroadcastAdd(ctx context.Context, bc sdk.Broadcast, u sdk.Identifiable) {
	e := sdk.EventBroadcastAdd{
		Broadcast: bc,
	}
	publishBroadcastEvent(ctx, e, bc.ProjectKey, u)
}

// PublishBroadcastUpdate publish event when updating a broadcast
func PublishBroadcastUpdate(ctx context.Context, oldBc sdk.Broadcast, bc sdk.Broadcast, u sdk.Identifiable) {
	e := sdk.EventBroadcastUpdate{
		NewBroadcast: bc,
		OldBroadcast: oldBc,
	}
	publishBroadcastEvent(ctx, e, bc.ProjectKey, u)
}

// PublishBroadcastDelete publish event when deleting a broadcast
func PublishBroadcastDelete(ctx context.Context, id int64, u sdk.Identifiable) {
	e := sdk.EventBroadcastDelete{
		BroadcastID: id,
	}
	publishBroadcastEvent(ctx, e, "", u)
}
