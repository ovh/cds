package event

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/ovh/cds/sdk"
)

func publishAsCodeEvent(ctx context.Context, payload interface{}, key string, u sdk.Identifiable) {
	bts, _ := json.Marshal(payload)
	event := sdk.Event{
		Timestamp:  time.Now(),
		Hostname:   hostname,
		CDSName:    cdsname,
		EventType:  fmt.Sprintf("%T", payload),
		Payload:    bts,
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
