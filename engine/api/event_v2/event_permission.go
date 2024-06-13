package event_v2

import (
	"context"
	"encoding/json"
	"time"

	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/sdk"
)

func PublishPermissionEvent(ctx context.Context, store cache.Store, eventType string, perm sdk.RBAC, u sdk.AuthentifiedUser) {
	bts, _ := json.Marshal(perm)
	e := sdk.PermissionEvent{
		GlobalEventV2: sdk.GlobalEventV2{
			ID:        sdk.UUID(),
			Type:      eventType,
			Payload:   bts,
			Timestamp: time.Now(),
		},
		Permission: perm.Name,
		UserID:     u.ID,
		Username:   u.Username,
	}
	publish(ctx, store, e)
}
