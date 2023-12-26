package event_v2

import (
	"context"

	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/sdk"
)

func PublishEntityDeleteEvent(ctx context.Context, store cache.Store, vcsName, repoName string, ent sdk.Entity, u *sdk.AuthentifiedUser) {
	e := sdk.EventV2{
		ID:         sdk.UUID(),
		ProjectKey: ent.ProjectKey,
		VCSName:    vcsName,
		Repository: repoName,
		Entity:     ent.Name,
		Type:       sdk.EventEntityDeleted,
		Payload:    ent,
	}
	if u != nil {
		e.UserID = u.ID
		e.Username = u.Username
	}
	publish(ctx, store, e)
}

func PublishEntityCreateEvent(ctx context.Context, store cache.Store, vcsName, repoName string, ent sdk.Entity, u *sdk.AuthentifiedUser) {
	e := sdk.EventV2{
		ID:         sdk.UUID(),
		ProjectKey: ent.ProjectKey,
		VCSName:    vcsName,
		Repository: repoName,
		Entity:     ent.Name,
		Type:       sdk.EventEntityCreated,
		Payload:    ent,
	}
	if u != nil {
		e.UserID = u.ID
		e.Username = u.Username
	}
	publish(ctx, store, e)
}

func PublishEntityUpdateEvent(ctx context.Context, store cache.Store, vcsName, repoName string, previousEnt, ent sdk.Entity, u *sdk.AuthentifiedUser) {
	e := sdk.EventV2{
		ID:         sdk.UUID(),
		ProjectKey: ent.ProjectKey,
		VCSName:    vcsName,
		Repository: repoName,
		Entity:     ent.Name,
		Type:       sdk.EventEntityUpdated,
		Previous:   previousEnt,
		Payload:    ent,
	}
	if u != nil {
		e.UserID = u.ID
		e.Username = u.Username
	}
	publish(ctx, store, e)
}
