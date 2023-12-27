package event_v2

import (
	"context"
	"encoding/json"

	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/sdk"
)

func PublishPluginCreateEvent(ctx context.Context, store cache.Store, p sdk.GRPCPlugin, u *sdk.AuthentifiedUser) {
	bts, _ := json.Marshal(p)
	e := sdk.EventV2{
		ID:      sdk.UUID(),
		Type:    sdk.EventPluginCreated,
		Plugin:  p.Name,
		Payload: bts,
	}
	if u != nil {
		e.UserID = u.ID
		e.Username = u.Username
	}
	publish(ctx, store, e)
}

func PublishPluginUpdateEvent(ctx context.Context, store cache.Store, pOld, p sdk.GRPCPlugin, u *sdk.AuthentifiedUser) {
	previousBts, _ := json.Marshal(pOld)
	bts, _ := json.Marshal(p)
	e := sdk.EventV2{
		ID:       sdk.UUID(),
		Type:     sdk.EventPluginUpdated,
		Plugin:   p.Name,
		Previous: previousBts,
		Payload:  bts,
	}
	if u != nil {
		e.UserID = u.ID
		e.Username = u.Username
	}
	publish(ctx, store, e)
}

func PublishPluginDeleteEvent(ctx context.Context, store cache.Store, p sdk.GRPCPlugin, u *sdk.AuthentifiedUser) {
	bts, _ := json.Marshal(p)
	e := sdk.EventV2{
		ID:      sdk.UUID(),
		Type:    sdk.EventPluginDeleted,
		Plugin:  p.Name,
		Payload: bts,
	}
	if u != nil {
		e.UserID = u.ID
		e.Username = u.Username
	}
	publish(ctx, store, e)
}
