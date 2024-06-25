package event_v2

import (
	"context"
	"encoding/json"
	"time"

	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/sdk"
)

func PublishIntegrationModelEvent(ctx context.Context, store cache.Store, eventType string, m sdk.IntegrationModel, u sdk.AuthentifiedUser) {
	bts, _ := json.Marshal(m)
	e := sdk.IntegrationModelEvent{
		GlobalEventV2: sdk.GlobalEventV2{
			ID:        sdk.UUID(),
			Type:      eventType,
			Payload:   bts,
			Timestamp: time.Now(),
		},
		IntegrationModel: m.Name,
		UserID:           u.ID,
		Username:         u.Username,
	}
	publish(ctx, store, e)
}

func PublishProjectIntegrationEvent(ctx context.Context, store cache.Store, eventType string, projectKey string, i sdk.ProjectIntegration, u sdk.AuthentifiedUser) {
	bts, _ := json.Marshal(i)
	e := sdk.ProjectIntegrationEvent{
		ProjectEventV2: sdk.ProjectEventV2{
			ID:         sdk.UUID(),
			Type:       eventType,
			Payload:    bts,
			ProjectKey: projectKey,
			Timestamp:  time.Now(),
		},
		Integration: i.Name,
		UserID:      u.ID,
		Username:    u.Username,
	}
	publish(ctx, store, e)
}
