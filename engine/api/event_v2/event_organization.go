package event_v2

import (
	"context"

	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/sdk"
)

func PublishOrganizationCreateEvent(ctx context.Context, store cache.Store, org sdk.Organization, u *sdk.AuthentifiedUser) {
	e := sdk.EventV2{
		ID:           sdk.UUID(),
		Organization: org.Name,
		Type:         sdk.EventOrganizationCreated,
		Payload:      org,
	}
	if u != nil {
		e.UserID = u.ID
		e.Username = u.Username
	}
	publish(ctx, store, e)
}

func PublishOrganizationDeleteEvent(ctx context.Context, store cache.Store, org sdk.Organization, u *sdk.AuthentifiedUser) {
	e := sdk.EventV2{
		ID:           sdk.UUID(),
		Organization: org.Name,
		Type:         sdk.EventOrganizationDeleted,
		Payload:      org,
	}
	if u != nil {
		e.UserID = u.ID
		e.Username = u.Username
	}
	publish(ctx, store, e)
}
