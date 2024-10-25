package event_v2

import (
	"context"
	"encoding/json"
	"time"

	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/sdk"
)

func PublishUserEvent(ctx context.Context, store cache.Store, typeEvent sdk.EventType, u sdk.AuthentifiedUser) {
	bts, _ := json.Marshal(u)
	e := sdk.UserEvent{
		GlobalEventV2: sdk.GlobalEventV2{
			ID:        sdk.UUID(),
			Type:      typeEvent,
			Payload:   bts,
			Timestamp: time.Now(),
		},
		UserID:   u.ID,
		Username: u.Username,
	}
	publish(ctx, store, e)
}

func PublishUserGPGEvent(ctx context.Context, store cache.Store, typeEvent sdk.EventType, g sdk.UserGPGKey, u sdk.AuthentifiedUser) {
	bts, _ := json.Marshal(g)
	e := sdk.UserGPGEvent{
		GlobalEventV2: sdk.GlobalEventV2{
			ID:        sdk.UUID(),
			Type:      typeEvent,
			Payload:   bts,
			Timestamp: time.Now(),
		},
		GPGKey:   g.KeyID,
		UserID:   u.ID,
		Username: u.Username,
	}
	publish(ctx, store, e)
}
