package event_v2

import (
	"context"
	"encoding/json"

	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/sdk"
)

func PublishUserCreateEvent(ctx context.Context, store cache.Store, u sdk.AuthentifiedUser) {
	bts, _ := json.Marshal(u)
	e := sdk.EventV2{
		ID:       sdk.UUID(),
		Type:     sdk.EventUserCreated,
		Payload:  bts,
		UserID:   u.ID,
		Username: u.Username,
	}
	publish(ctx, store, e)
}

func PublishUserUpdateEvent(ctx context.Context, store cache.Store, u sdk.AuthentifiedUser) {
	bts, _ := json.Marshal(u)
	e := sdk.EventV2{
		ID:       sdk.UUID(),
		Type:     sdk.EventUserUpdated,
		Payload:  bts,
		UserID:   u.ID,
		Username: u.Username,
	}
	publish(ctx, store, e)
}

func PublishUserDeleteEvent(ctx context.Context, store cache.Store, u sdk.AuthentifiedUser) {
	bts, _ := json.Marshal(u)
	e := sdk.EventV2{
		ID:       sdk.UUID(),
		Type:     sdk.EventUserDeleted,
		Payload:  bts,
		UserID:   u.ID,
		Username: u.Username,
	}
	publish(ctx, store, e)
}

func PublishUserGPGCreateEvent(ctx context.Context, store cache.Store, g sdk.UserGPGKey, u *sdk.AuthentifiedUser) {
	bts, _ := json.Marshal(g)
	e := sdk.EventV2{
		ID:      sdk.UUID(),
		GPGKey:  g.KeyID,
		Type:    sdk.EventUserGPGKeyCreated,
		Payload: bts,
	}
	if u != nil {
		e.UserID = u.ID
		e.Username = u.Username
	}
	publish(ctx, store, e)
}

func PublishUserGPGDeleteEvent(ctx context.Context, store cache.Store, g sdk.UserGPGKey, u *sdk.AuthentifiedUser) {
	bts, _ := json.Marshal(g)
	e := sdk.EventV2{
		ID:      sdk.UUID(),
		GPGKey:  g.KeyID,
		Type:    sdk.EventUserGPGKeyDeleted,
		Payload: bts,
	}
	if u != nil {
		e.UserID = u.ID
		e.Username = u.Username
	}
	publish(ctx, store, e)
}
