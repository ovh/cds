package event_v2

import (
	"context"
	"encoding/json"
	"time"

	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/sdk"
)

func PublishRegionEvent(ctx context.Context, store cache.Store, typeEvent string, reg sdk.Region, u sdk.AuthentifiedUser) {
	bts, _ := json.Marshal(reg)
	e := sdk.RegionEvent{
		GlobalEventV2: sdk.GlobalEventV2{
			ID:        sdk.UUID(),
			Type:      typeEvent,
			Payload:   bts,
			Timestamp: time.Now(),
		},
		Region:   reg.Name,
		UserID:   u.ID,
		Username: u.Username,
	}
	publish(ctx, store, e)
}
