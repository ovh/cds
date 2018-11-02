package api

import (
	"context"
	"net/http"

	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
)

func (api *API) getUserNotificationTypeHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		True := true
		False := false
		return service.WriteJSON(w, map[string]sdk.UserNotificationSettings{
			sdk.EmailUserNotification: sdk.UserNotificationSettings{
				OnSuccess:    sdk.UserNotificationChange,
				OnFailure:    sdk.UserNotificationAlways,
				OnStart:      &False,
				SendToAuthor: &True,
				SendToGroups: &False,
				Template:     &sdk.UserNotificationTemplateEmail,
			},
			sdk.JabberUserNotification: sdk.UserNotificationSettings{
				OnSuccess:    sdk.UserNotificationChange,
				OnFailure:    sdk.UserNotificationAlways,
				OnStart:      &False,
				SendToAuthor: &True,
				SendToGroups: &False,
				Template:     &sdk.UserNotificationTemplateJabber,
			},
		}, http.StatusOK)
	}
}

func (api *API) getUserNotificationStateValueHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		return service.WriteJSON(w, []string{
			sdk.UserNotificationAlways,
			sdk.UserNotificationChange,
			sdk.UserNotificationNever,
		}, http.StatusOK)
	}
}
