package event_v2

import (
	"context"
	"encoding/json"

	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/sdk"
)

func PublishProjectKeyEvent(ctx context.Context, store cache.Store, eventType string, projectKey string, k sdk.ProjectKey, u sdk.AuthentifiedUser) {
	bts, _ := json.Marshal(k)
	e := sdk.KeyEvent{
		ProjectEventV2: sdk.ProjectEventV2{
			ID:         sdk.UUID(),
			Type:       eventType,
			Payload:    bts,
			ProjectKey: projectKey,
		},
		KeyName:  k.Name,
		KeyType:  k.Type.String(),
		UserID:   u.ID,
		Username: u.Username,
	}
	publish(ctx, store, e)
}
