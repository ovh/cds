package event

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/sdk"
)

const DefaultPubSubKey = "events_pubsub"
const JobQueuedPubSubKey = "run:job:queued"

var pubSubKey = DefaultPubSubKey

func OverridePubSubKey(key string) {
	pubSubKey = key
}

type Store interface {
	cache.PubSubStore
	cache.QueueStore
}

var store Store

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
	return store.Publish(ctx, pubSubKey, string(b))
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
