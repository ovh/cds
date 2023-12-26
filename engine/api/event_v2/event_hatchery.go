package event_v2

import (
	"context"

	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/sdk"
)

func PublishHatcheryCreateEvent(ctx context.Context, store cache.Store, h sdk.Hatchery, u *sdk.AuthentifiedUser) {
	e := sdk.EventV2{
		ID:       sdk.UUID(),
		Hatchery: h.Name,
		Type:     sdk.EventHatcheryCreated,
		Payload:  h,
	}
	if u != nil {
		e.UserID = u.ID
		e.Username = u.Username
	}
	publish(ctx, store, e)
}

func PublishHatcheryUpdatedEvent(ctx context.Context, store cache.Store, previousHatchery, h sdk.Hatchery, u *sdk.AuthentifiedUser) {
	e := sdk.EventV2{
		ID:       sdk.UUID(),
		Hatchery: h.Name,
		Type:     sdk.EventHatcheryUpdated,
		Previous: previousHatchery,
		Payload:  h,
	}
	if u != nil {
		e.UserID = u.ID
		e.Username = u.Username
	}
	publish(ctx, store, e)
}

func PublishHatcheryDeleteEvent(ctx context.Context, store cache.Store, h sdk.Hatchery, u *sdk.AuthentifiedUser) {
	e := sdk.EventV2{
		ID:       sdk.UUID(),
		Hatchery: h.Name,
		Type:     sdk.EventHatcheryDeleted,
		Payload:  h,
	}
	if u != nil {
		e.UserID = u.ID
		e.Username = u.Username
	}
	publish(ctx, store, e)
}
