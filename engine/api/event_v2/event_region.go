package event_v2

import (
	"context"
	"encoding/json"

	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/sdk"
)

func PublishRegionCreateEvent(ctx context.Context, store cache.Store, reg sdk.Region, u *sdk.AuthentifiedUser) {
	bts, _ := json.Marshal(reg)
	e := sdk.EventV2{
		ID:      sdk.UUID(),
		Region:  reg.Name,
		Type:    sdk.EventRegionCreated,
		Payload: bts,
	}
	if u != nil {
		e.UserID = u.ID
		e.Username = u.Username
	}
	publish(ctx, store, e)
}

func PublishRegionDeleteEvent(ctx context.Context, store cache.Store, reg sdk.Region, u *sdk.AuthentifiedUser) {
	bts, _ := json.Marshal(reg)
	e := sdk.EventV2{
		ID:      sdk.UUID(),
		Region:  reg.Name,
		Type:    sdk.EventRegionDeleted,
		Payload: bts,
	}
	if u != nil {
		e.UserID = u.ID
		e.Username = u.Username
	}
	publish(ctx, store, e)
}
