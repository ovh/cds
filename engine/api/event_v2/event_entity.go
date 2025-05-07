package event_v2

import (
	"context"
	"encoding/json"
	"time"

	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/sdk"
)

func PublishEntityEvent(ctx context.Context, store cache.Store, eventType sdk.EventType, vcsName, repoName string, ent sdk.Entity, u *sdk.V2Initiator) {
	bts, _ := json.Marshal(ent)
	e := sdk.EntityEvent{
		GlobalEventV2: sdk.GlobalEventV2{
			ID:        sdk.UUID(),
			Type:      eventType,
			Payload:   bts,
			Timestamp: time.Now(),
		},
		ProjectEventV2: sdk.ProjectEventV2{
			ProjectKey: ent.ProjectKey,
		},
		VCSName:    vcsName,
		Repository: repoName,
		Entity:     ent.Name,
	}
	// User is nil for deletion (entity deletion is initiated by CDS itself)
	if u != nil {
		e.UserID = u.UserID
		e.Username = u.Username()
	}
	publish(ctx, store, e)
}
