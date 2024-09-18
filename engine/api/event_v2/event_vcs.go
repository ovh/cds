package event_v2

import (
	"context"
	"encoding/json"
	"time"

	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/sdk"
)

func PublishVCSEvent(ctx context.Context, store cache.Store, eventType sdk.EventType, projectKey string, vcs sdk.VCSProject, u sdk.AuthentifiedUser) {
	bts, _ := json.Marshal(vcs)
	e := sdk.VCSEvent{
		GlobalEventV2: sdk.GlobalEventV2{
			ID:        sdk.UUID(),
			Type:      eventType,
			Payload:   bts,
			Timestamp: time.Now(),
		},
		ProjectEventV2: sdk.ProjectEventV2{
			ProjectKey: projectKey,
		},
		VCSName:  vcs.Name,
		UserID:   u.ID,
		Username: u.Username,
	}
	publish(ctx, store, e)
}
