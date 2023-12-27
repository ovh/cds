package event_v2

import (
	"context"
	"encoding/json"

	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/sdk"
)

func PublishRepositoryCreateEvent(ctx context.Context, store cache.Store, projectKey string, vcsName string, repo sdk.ProjectRepository, u *sdk.AuthentifiedUser) {
	bts, _ := json.Marshal(repo)
	e := sdk.EventV2{
		ID:         sdk.UUID(),
		ProjectKey: projectKey,
		VCSName:    vcsName,
		Repository: repo.Name,
		Type:       sdk.EventRepositoryCreated,
		Payload:    bts,
	}
	if u != nil {
		e.UserID = u.ID
		e.Username = u.Username
	}
	publish(ctx, store, e)
}

func PublishRepositoryDeleteEvent(ctx context.Context, store cache.Store, projectKey string, vcsName string, repo sdk.ProjectRepository, u *sdk.AuthentifiedUser) {
	bts, _ := json.Marshal(repo)
	e := sdk.EventV2{
		ID:         sdk.UUID(),
		ProjectKey: projectKey,
		VCSName:    vcsName,
		Repository: repo.Name,
		Type:       sdk.EventRepositoryDeleted,
		Payload:    bts,
	}
	if u != nil {
		e.UserID = u.ID
		e.Username = u.Username
	}
	publish(ctx, store, e)
}
