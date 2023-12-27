package event_v2

import (
	"context"

	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/sdk"
)

func PublishPluginCreateEvent(ctx context.Context, store cache.Store, p sdk.GRPCPlugin, u *sdk.AuthentifiedUser) {
	e := sdk.EventV2{
		ID:      sdk.UUID(),
		Type:    sdk.EventPluginCreated,
		Plugin:  p.Name,
		Payload: p,
	}
	if u != nil {
		e.UserID = u.ID
		e.Username = u.Username
	}
	publish(ctx, store, e)
}

func PublishPluginUpdateEvent(ctx context.Context, store cache.Store, pOld, p sdk.GRPCPlugin, u *sdk.AuthentifiedUser) {
	e := sdk.EventV2{
		ID:       sdk.UUID(),
		Type:     sdk.EventPluginUpdated,
		Plugin:   p.Name,
		Previous: pOld,
		Payload:  p,
	}
	if u != nil {
		e.UserID = u.ID
		e.Username = u.Username
	}
	publish(ctx, store, e)
}

func PublishPluginDeleteEvent(ctx context.Context, store cache.Store, p sdk.GRPCPlugin, u *sdk.AuthentifiedUser) {
	e := sdk.EventV2{
		ID:      sdk.UUID(),
		Type:    sdk.EventPluginDeleted,
		Plugin:  p.Name,
		Payload: p,
	}
	if u != nil {
		e.UserID = u.ID
		e.Username = u.Username
	}
	publish(ctx, store, e)
}
