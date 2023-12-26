package event_v2

import (
	"context"

	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/sdk"
)

func PublishPermissionCreateEvent(ctx context.Context, store cache.Store, perm sdk.RBAC, u *sdk.AuthentifiedUser) {
	e := sdk.EventV2{
		ID:         sdk.UUID(),
		Type:       sdk.EventPermissionCreated,
		Permission: perm.Name,
		Payload:    perm,
	}
	if u != nil {
		e.UserID = u.ID
		e.Username = u.Username
	}
	publish(ctx, store, e)
}

func PublishPermissionUpdatedEvent(ctx context.Context, store cache.Store, previousPerm, perm sdk.RBAC, u *sdk.AuthentifiedUser) {
	e := sdk.EventV2{
		ID:         sdk.UUID(),
		Type:       sdk.EventPermissionUpdated,
		Permission: perm.Name,
		Previous:   previousPerm,
		Payload:    perm,
	}
	if u != nil {
		e.UserID = u.ID
		e.Username = u.Username
	}
	publish(ctx, store, e)
}

func PublishPermissionDeleteEvent(ctx context.Context, store cache.Store, perm sdk.RBAC, u *sdk.AuthentifiedUser) {
	e := sdk.EventV2{
		ID:         sdk.UUID(),
		Type:       sdk.EventPermissionDeleted,
		Permission: perm.Name,
		Payload:    perm,
	}
	if u != nil {
		e.UserID = u.ID
		e.Username = u.Username
	}
	publish(ctx, store, e)
}
