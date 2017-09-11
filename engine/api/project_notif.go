package api

import (
	"context"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/notification"
	"github.com/ovh/cds/engine/api/project"
)

func (api *API) getProjectNotificationsHandler() Handler {
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

		WriteJSON(w, r, notifs, http.StatusOK)
		return nil
	}
}
