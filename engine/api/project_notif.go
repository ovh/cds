package main

import (
	"net/http"

	"github.com/go-gorp/gorp"
	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/context"
	"github.com/ovh/cds/engine/api/notification"
)

func getProjectNotificationsHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	// Get project name in URL
	vars := mux.Vars(r)
	key := vars["permProjectKey"]

	notifs, err := notification.LoadAllUserNotificationSettingsByProject(db, key)
	if err != nil {
		return err
	}

	WriteJSON(w, r, notifs, http.StatusOK)

	return nil
}
