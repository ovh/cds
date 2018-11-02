package api

import (
	"context"
	"net/http"

	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
)

func (api *API) getUserNotificationTypeHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		return service.WriteJSON(w, []string{
			sdk.EmailUserNotification,
			sdk.JabberUserNotification,
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
