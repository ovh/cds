package event_v2

import (
	"context"
	"encoding/json"
	"time"

	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/sdk"
)

func PublishProjectVariableSetEvent(ctx context.Context, store cache.Store, eventType sdk.EventType, projectKey string, vs sdk.ProjectVariableSet, u sdk.AuthentifiedUser) {
	bts, _ := json.Marshal(vs)
	e := sdk.ProjectVariableSetEvent{
		GlobalEventV2: sdk.GlobalEventV2{
			ID:        sdk.UUID(),
			Type:      eventType,
			Payload:   bts,
			Timestamp: time.Now(),
		},
		ProjectEventV2: sdk.ProjectEventV2{
			ProjectKey: projectKey,
		},
		VariableSet: vs.Name,
		UserID:      u.ID,
		Username:    u.Username,
	}
	publish(ctx, store, e)
}

func PublishProjectVariableSetItemEvent(ctx context.Context, store cache.Store, eventType sdk.EventType, projectKey string, vsName string, item sdk.ProjectVariableSetItem, u sdk.AuthentifiedUser) {
	bts, _ := json.Marshal(item)
	e := sdk.ProjectVariableSetItemEvent{
		GlobalEventV2: sdk.GlobalEventV2{
			ID:        sdk.UUID(),
			Type:      eventType,
			Payload:   bts,
			Timestamp: time.Now(),
		},
		ProjectEventV2: sdk.ProjectEventV2{
			ProjectKey: projectKey,
		},
		VariableSet: vsName,
		Item:        item.Name,
		UserID:      u.ID,
		Username:    u.Username,
	}
	publish(ctx, store, e)
}
