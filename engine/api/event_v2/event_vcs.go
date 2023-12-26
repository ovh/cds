package event_v2

import (
	"context"

	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/sdk"
)

func PublishVCSCreateEvent(ctx context.Context, store cache.Store, projectKey string, vcs sdk.VCSProject, u *sdk.AuthentifiedUser) {
	e := sdk.EventV2{
		ID:         sdk.UUID(),
		ProjectKey: projectKey,
		VCSName:    vcs.Name,
		Type:       sdk.EventVCSCreated,
		Payload:    vcs,
	}
	if u != nil {
		e.UserID = u.ID
		e.Username = u.Username
	}
	publish(ctx, store, e)
}

func PublishVCSUpdatedEvent(ctx context.Context, store cache.Store, projectKey string, previousVCS, vcs sdk.VCSProject, u *sdk.AuthentifiedUser) {
	e := sdk.EventV2{
		ID:         sdk.UUID(),
		ProjectKey: projectKey,
		VCSName:    vcs.Name,
		Type:       sdk.EventVCSDeleted,
		Previous:   previousVCS,
		Payload:    vcs,
	}
	if u != nil {
		e.UserID = u.ID
		e.Username = u.Username
	}
	publish(ctx, store, e)
}

func PublishVCSDeleteEvent(ctx context.Context, store cache.Store, projectKey string, vcs sdk.VCSProject, u *sdk.AuthentifiedUser) {
	e := sdk.EventV2{
		ID:         sdk.UUID(),
		ProjectKey: projectKey,
		VCSName:    vcs.Name,
		Type:       sdk.EventVCSDeleted,
		Payload:    vcs,
	}
	if u != nil {
		e.UserID = u.ID
		e.Username = u.Username
	}
	publish(ctx, store, e)
}
