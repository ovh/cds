package event_v2

import (
	"context"
	"encoding/json"

	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/sdk"
)

func PublishVariableEvent(ctx context.Context, store cache.Store, eventType string, projectKey string, v sdk.ProjectVariable, u sdk.AuthentifiedUser) {
	bts, _ := json.Marshal(v)
	e := sdk.VariableEvent{
		ProjectEventV2: sdk.ProjectEventV2{
			ID:         sdk.UUID(),
			Type:       eventType,
			Payload:    bts,
			ProjectKey: projectKey,
		},
		Variable: v.Name,
		UserID:   u.ID,
		Username: u.Username,
	}
	publish(ctx, store, e)
}
