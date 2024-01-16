package event_v2

import (
	"context"
	"encoding/json"

	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/sdk"
)

func PublishProjectEvent(ctx context.Context, store cache.Store, eventType string, p sdk.Project, u sdk.AuthentifiedUser) {
	bts, _ := json.Marshal(p)
	e := sdk.ProjectEvent{
		ProjectEventV2: sdk.ProjectEventV2{
			ID:         sdk.UUID(),
			Type:       eventType,
			Payload:    bts,
			ProjectKey: p.Key,
		},
		UserID:   u.ID,
		Username: u.Username,
	}
	publish(ctx, store, e)
}
