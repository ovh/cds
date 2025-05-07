package event_v2

import (
	"context"
	"encoding/json"
	"time"

	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/sdk"
)

func PublishConcurrencyEvent(ctx context.Context, store cache.Store, eventType sdk.EventType, projectKey string, concu sdk.ProjectConcurrency, u sdk.AuthentifiedUser) {
	bts, _ := json.Marshal(concu)
	e := sdk.ConcurrencyEvent{
		GlobalEventV2: sdk.GlobalEventV2{
			ID:        sdk.UUID(),
			Type:      eventType,
			Payload:   bts,
			Timestamp: time.Now(),
		},
		ProjectEventV2: sdk.ProjectEventV2{
			ProjectKey: projectKey,
		},
		UserID:      u.ID,
		Username:    u.Username,
		Concurrency: concu.Name,
	}
	publish(ctx, store, e)
}
