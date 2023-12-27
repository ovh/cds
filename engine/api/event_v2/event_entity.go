package event_v2

import (
	"context"
	"encoding/json"

	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/sdk"
)

func PublishEntityDeleteEvent(ctx context.Context, store cache.Store, vcsName, repoName string, ent sdk.Entity, u *sdk.AuthentifiedUser) {
	bts, _ := json.Marshal(ent)
	e := sdk.EventV2{
		ID:         sdk.UUID(),
		ProjectKey: ent.ProjectKey,
		VCSName:    vcsName,
		Repository: repoName,
		Entity:     ent.Name,
		Type:       sdk.EventEntityDeleted,
		Payload:    bts,
	}
	if u != nil {
		e.UserID = u.ID
		e.Username = u.Username
	}
	publish(ctx, store, e)
}

func PublishEntityCreateEvent(ctx context.Context, store cache.Store, vcsName, repoName string, ent sdk.Entity, u *sdk.AuthentifiedUser) {
	bts, _ := json.Marshal(ent)
	e := sdk.EventV2{
		ID:         sdk.UUID(),
		ProjectKey: ent.ProjectKey,
		VCSName:    vcsName,
		Repository: repoName,
		Entity:     ent.Name,
		Type:       sdk.EventEntityCreated,
		Payload:    bts,
	}
	if u != nil {
		e.UserID = u.ID
		e.Username = u.Username
	}
	publish(ctx, store, e)
}

func PublishEntityUpdateEvent(ctx context.Context, store cache.Store, vcsName, repoName string, previousEnt, ent sdk.Entity, u *sdk.AuthentifiedUser) {
	previousBts, _ := json.Marshal(previousEnt)
	bts, _ := json.Marshal(ent)
	e := sdk.EventV2{
		ID:         sdk.UUID(),
		ProjectKey: ent.ProjectKey,
		VCSName:    vcsName,
		Repository: repoName,
		Entity:     ent.Name,
		Type:       sdk.EventEntityUpdated,
		Previous:   previousBts,
		Payload:    bts,
	}
	if u != nil {
		e.UserID = u.ID
		e.Username = u.Username
	}
	publish(ctx, store, e)
}
