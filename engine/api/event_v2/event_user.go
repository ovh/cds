package event_v2

import (
	"context"

	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/sdk"
)

func PublishUserCreateEvent(ctx context.Context, store cache.Store, u sdk.AuthentifiedUser) {
	e := sdk.EventV2{
		ID:       sdk.UUID(),
		Type:     sdk.EventUserCreated,
		Payload:  u,
		UserID:   u.ID,
		Username: u.Username,
	}
	publish(ctx, store, e)
}

func PublishUserUpdateEvent(ctx context.Context, store cache.Store, u sdk.AuthentifiedUser) {
	e := sdk.EventV2{
		ID:       sdk.UUID(),
		Type:     sdk.EventUserUpdated,
		Payload:  u,
		UserID:   u.ID,
		Username: u.Username,
	}
	publish(ctx, store, e)
}

func PublishUserDeleteEvent(ctx context.Context, store cache.Store, u sdk.AuthentifiedUser) {
	e := sdk.EventV2{
		ID:       sdk.UUID(),
		Type:     sdk.EventUserDeleted,
		Payload:  u,
		UserID:   u.ID,
		Username: u.Username,
	}
	publish(ctx, store, e)
}
