package event_v2

import (
	"context"
	"encoding/json"

	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/sdk"
)

func PublishOrganizationEvent(ctx context.Context, store cache.Store, eventType string, org sdk.Organization, u sdk.AuthentifiedUser) {
	bts, _ := json.Marshal(org)
	e := sdk.OrganizationEvent{
		GlobalEventV2: sdk.GlobalEventV2{
			ID:      sdk.UUID(),
			Type:    eventType,
			Payload: bts,
		},
		Organization: org.Name,
		UserID:       u.ID,
		Username:     u.Username,
	}
	publish(ctx, store, e)
}
