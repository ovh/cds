package event_v2

import (
	"context"
	"encoding/json"

	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/sdk"
)

func PublishEntityEvent(ctx context.Context, store cache.Store, eventType string, vcsName, repoName string, ent sdk.Entity, u *sdk.AuthentifiedUser) {
	bts, _ := json.Marshal(ent)
	e := sdk.EntityEvent{
		ProjectEventV2: sdk.ProjectEventV2{
			ID:         sdk.UUID(),
			Type:       eventType,
			Payload:    bts,
			ProjectKey: ent.ProjectKey,
		},
		VCSName:    vcsName,
		Repository: repoName,
		Entity:     ent.Name,
	}
	// User is nil for deletion (entity deletion is initiated by CDS itself)
	if u != nil {
		e.UserID = u.ID
		e.Username = u.Username
	}
	publish(ctx, store, e)
}
