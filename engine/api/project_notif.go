package api

import (
	"context"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/notification"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/service"
)

func (api *API) getProjectNotificationsHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		// Get project name in URL
		vars := mux.Vars(r)
		key := vars["permProjectKey"]

		if _, err := project.Load(api.mustDB(), api.Cache, key, nil); err != nil {
			return err
		}

		notifs, err := notification.LoadAllUserNotificationSettingsByProject(api.mustDB(), key, getUser(ctx))
		if err != nil {
			return err
		}

		return service.WriteJSON(w, notifs, http.StatusOK)
	}
}
