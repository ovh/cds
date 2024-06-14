package event_v2

import (
	"context"
	"encoding/json"
	"time"

	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/sdk"
)

func PublishProjectVariableSetEvent(ctx context.Context, store cache.Store, eventType string, projectKey string, vs sdk.ProjectVariableSet, u sdk.AuthentifiedUser) {
	bts, _ := json.Marshal(vs)
	e := sdk.ProjectVariableSetEvent{
		ProjectEventV2: sdk.ProjectEventV2{
			ID:         sdk.UUID(),
			Type:       eventType,
			Payload:    bts,
			ProjectKey: projectKey,
			Timestamp:  time.Now(),
		},
		VariableSet: vs.Name,
		UserID:      u.ID,
		Username:    u.Username,
	}
	publish(ctx, store, e)
}

func PublishProjectVariableSetItemEvent(ctx context.Context, store cache.Store, eventType string, projectKey string, vsName string, item sdk.ProjectVariableSetItem, u sdk.AuthentifiedUser) {
	bts, _ := json.Marshal(item)
	e := sdk.ProjectVariableSetItemEvent{
		ProjectEventV2: sdk.ProjectEventV2{
			ID:         sdk.UUID(),
			Type:       eventType,
			Payload:    bts,
			ProjectKey: projectKey,
			Timestamp:  time.Now(),
		},
		VariableSet: vsName,
		Item:        item.Name,
		UserID:      u.ID,
		Username:    u.Username,
	}
	publish(ctx, store, e)
}
