package event_v2

import (
	"context"
	"encoding/json"

	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/sdk"
)

func PublishVCSCreateEvent(ctx context.Context, store cache.Store, projectKey string, vcs sdk.VCSProject, u *sdk.AuthentifiedUser) {
	bts, _ := json.Marshal(vcs)
	e := sdk.EventV2{
		ID:         sdk.UUID(),
		ProjectKey: projectKey,
		VCSName:    vcs.Name,
		Type:       sdk.EventVCSCreated,
		Payload:    bts,
	}
	if u != nil {
		e.UserID = u.ID
		e.Username = u.Username
	}
	publish(ctx, store, e)
}

func PublishVCSUpdatedEvent(ctx context.Context, store cache.Store, projectKey string, previousVCS, vcs sdk.VCSProject, u *sdk.AuthentifiedUser) {
	previousBts, _ := json.Marshal(previousVCS)
	bts, _ := json.Marshal(vcs)
	e := sdk.EventV2{
		ID:         sdk.UUID(),
		ProjectKey: projectKey,
		VCSName:    vcs.Name,
		Type:       sdk.EventVCSDeleted,
		Previous:   previousBts,
		Payload:    bts,
	}
	if u != nil {
		e.UserID = u.ID
		e.Username = u.Username
	}
	publish(ctx, store, e)
}

func PublishVCSDeleteEvent(ctx context.Context, store cache.Store, projectKey string, vcs sdk.VCSProject, u *sdk.AuthentifiedUser) {
	bts, _ := json.Marshal(vcs)
	e := sdk.EventV2{
		ID:         sdk.UUID(),
		ProjectKey: projectKey,
		VCSName:    vcs.Name,
		Type:       sdk.EventVCSDeleted,
		Payload:    bts,
	}
	if u != nil {
		e.UserID = u.ID
		e.Username = u.Username
	}
	publish(ctx, store, e)
}
