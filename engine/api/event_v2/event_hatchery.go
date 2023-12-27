package event_v2

import (
	"context"
	"encoding/json"

	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/sdk"
)

func PublishHatcheryCreateEvent(ctx context.Context, store cache.Store, h sdk.Hatchery, u *sdk.AuthentifiedUser) {
	bts, _ := json.Marshal(h)
	e := sdk.EventV2{
		ID:       sdk.UUID(),
		Hatchery: h.Name,
		Type:     sdk.EventHatcheryCreated,
		Payload:  bts,
	}
	if u != nil {
		e.UserID = u.ID
		e.Username = u.Username
	}
	publish(ctx, store, e)
}

func PublishHatcheryUpdatedEvent(ctx context.Context, store cache.Store, previousHatchery, h sdk.Hatchery, u *sdk.AuthentifiedUser) {
	previousH, _ := json.Marshal(previousHatchery)
	bts, _ := json.Marshal(h)
	e := sdk.EventV2{
		ID:       sdk.UUID(),
		Hatchery: h.Name,
		Type:     sdk.EventHatcheryUpdated,
		Previous: previousH,
		Payload:  bts,
	}
	if u != nil {
		e.UserID = u.ID
		e.Username = u.Username
	}
	publish(ctx, store, e)
}

func PublishHatcheryDeleteEvent(ctx context.Context, store cache.Store, h sdk.Hatchery, u *sdk.AuthentifiedUser) {
	bts, _ := json.Marshal(h)
	e := sdk.EventV2{
		ID:       sdk.UUID(),
		Hatchery: h.Name,
		Type:     sdk.EventHatcheryDeleted,
		Payload:  bts,
	}
	if u != nil {
		e.UserID = u.ID
		e.Username = u.Username
	}
	publish(ctx, store, e)
}
