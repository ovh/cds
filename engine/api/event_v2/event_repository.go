package event_v2

import (
	"context"
	"encoding/json"
	"time"

	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/sdk"
)

func PublishRepositoryEvent(ctx context.Context, store cache.Store, eventType string, projectKey string, vcsName string, repo sdk.ProjectRepository, u sdk.AuthentifiedUser) {
	bts, _ := json.Marshal(repo)
	e := sdk.RepositoryEvent{
		ProjectEventV2: sdk.ProjectEventV2{
			ID:         sdk.UUID(),
			Type:       eventType,
			Payload:    bts,
			ProjectKey: projectKey,
			Timestamp:  time.Now(),
		},
		VCSName:    vcsName,
		Repository: repo.Name,
		UserID:     u.ID,
		Username:   u.Username,
	}
	publish(ctx, store, e)
}
