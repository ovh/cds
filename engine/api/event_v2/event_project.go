package event_v2

import (
	"context"
	"encoding/json"
	"time"

	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/sdk"
)

func PublishProjectEvent(ctx context.Context, store cache.Store, eventType sdk.EventType, p sdk.Project, u sdk.AuthentifiedUser) {
	bts, _ := json.Marshal(p)
	e := sdk.ProjectEvent{
		GlobalEventV2: sdk.GlobalEventV2{
			ID:        sdk.UUID(),
			Type:      eventType,
			Payload:   bts,
			Timestamp: time.Now(),
		},
		ProjectEventV2: sdk.ProjectEventV2{
			ProjectKey: p.Key,
		},
		UserID:   u.ID,
		Username: u.Username,
	}
	publish(ctx, store, e)
}
