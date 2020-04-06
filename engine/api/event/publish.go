package event

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/sdk"
)

var store cache.Store

func publishEvent(ctx context.Context, e sdk.Event) error {
	if store == nil {
		return nil
	}

	if err := store.Enqueue("events", e); err != nil {
		return err
	}
	b, err := json.Marshal(e)
	if err != nil {
		return sdk.WrapError(err, "Cannot marshal event %+v", e)
	}
	return store.Publish(ctx, "events_pubsub", string(b))
}

// Publish sends a event to a queue
func Publish(ctx context.Context, payload interface{}, u sdk.Identifiable) {
	bts, _ := json.Marshal(payload)
	event := sdk.Event{
		Timestamp: time.Now(),
		Hostname:  hostname,
		CDSName:   cdsname,
		EventType: fmt.Sprintf("%T", payload),
		Payload:   bts,
	}
	if u != nil {
		event.Username = u.GetUsername()
		event.UserMail = u.GetEmail()
	}
	_ = publishEvent(ctx, event)
}
