package main

import (
	"net/http"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/businesscontext"
	"github.com/ovh/cds/engine/api/notification"
)

func getProjectNotificationsHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *businesscontext.Ctx) error {
	notifs, err := notification.LoadAllUserNotificationSettingsByProject(db, c.Project.Key, c.User)
	if err != nil {
		return err
	}
	return WriteJSON(w, r, notifs, http.StatusOK)
}
