package event

import (
	"context"
	"fmt"
	"time"

	"github.com/fatih/structs"

	"github.com/ovh/cds/sdk"
)

func publishAsCodeEvent(ctx context.Context, payload interface{}, key string, u sdk.Identifiable) {
	event := sdk.Event{
		Timestamp:  time.Now(),
		Hostname:   hostname,
		CDSName:    cdsname,
		EventType:  fmt.Sprintf("%T", payload),
		Payload:    structs.Map(payload),
		ProjectKey: key,
	}
	if u != nil {
		event.Username = u.GetUsername()
		event.UserMail = u.GetEmail()
	}
	_ = publishEvent(ctx, event)
}

func PublishAsCodeEvent(ctx context.Context, projKey string, asCodeEvent sdk.AsCodeEvent, u sdk.Identifiable) {
	e := sdk.EventAsCodeEvent{
		Event: asCodeEvent,
	}
	publishAsCodeEvent(ctx, e, projKey, u)
}
