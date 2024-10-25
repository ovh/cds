package event_v2

import (
	"context"
	"encoding/json"
	"time"

	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/sdk"
)

func PublishRepositoryEvent(ctx context.Context, store cache.Store, eventType sdk.EventType, projectKey string, vcsName string, repo sdk.ProjectRepository, u sdk.AuthentifiedUser) {
	bts, _ := json.Marshal(repo)
	e := sdk.RepositoryEvent{
		GlobalEventV2: sdk.GlobalEventV2{
			ID:        sdk.UUID(),
			Type:      eventType,
			Payload:   bts,
			Timestamp: time.Now(),
		},
		ProjectEventV2: sdk.ProjectEventV2{
			ProjectKey: projectKey,
		},
		VCSName:    vcsName,
		Repository: repo.Name,
		UserID:     u.ID,
		Username:   u.Username,
	}
	publish(ctx, store, e)
}
