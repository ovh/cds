package event_v2

import (
	"context"
	"encoding/json"
	"time"

	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/sdk"
)

func PublishPluginEvent(ctx context.Context, store cache.Store, typeEvent string, p sdk.GRPCPlugin, u sdk.AuthentifiedUser) {
	bts, _ := json.Marshal(p)
	e := sdk.PluginEvent{
		GlobalEventV2: sdk.GlobalEventV2{
			ID:        sdk.UUID(),
			Type:      typeEvent,
			Payload:   bts,
			Timestamp: time.Now(),
		},
		Plugin:   p.Name,
		UserID:   u.ID,
		Username: u.Username,
	}
	publish(ctx, store, e)
}
