package event_v2

import (
	"context"
	"encoding/json"

	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/sdk"
)

func PublishProjectSecretEvent(ctx context.Context, store cache.Store, eventType string, secret sdk.ProjectSecret, u sdk.AuthentifiedUser) {
	bts, _ := json.Marshal(secret)
	e := sdk.ProjectSecretEvent{
		ProjectEventV2: sdk.ProjectEventV2{
			ID:         sdk.UUID(),
			Type:       eventType,
			Payload:    bts,
			ProjectKey: secret.ProjectKey,
		},
		SecretName: secret.Name,
		UserID:     u.ID,
		Username:   u.Username,
	}
	publish(ctx, store, e)
}
