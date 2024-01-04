package event_v2

import (
	"context"
	"encoding/json"

	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/sdk"
)

func PublishProjectNotificationEvent(ctx context.Context, store cache.Store, eventType string, projectKey string, notif sdk.ProjectNotification, u sdk.AuthentifiedUser) {
	bts, _ := json.Marshal(notif)
	e := sdk.NotificationEvent{
		ProjectEventV2: sdk.ProjectEventV2{
			ID:         sdk.UUID(),
			Type:       eventType,
			Payload:    bts,
			ProjectKey: projectKey,
		},
		Notification: notif.Name,
		UserID:       u.ID,
		Username:     u.Username,
	}
	publish(ctx, store, e)
}
