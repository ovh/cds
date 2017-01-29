package main

import (
	"net/http"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/context"
	"github.com/ovh/cds/engine/log"
)

func adminTruncateWarningsHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	if _, err := db.Exec("delete from warning"); err != nil {
		log.Warning("adminTruncateWarningsHandler> Unable to truncate warning : %s", err)
		return err
	}
	return nil
}

func postAdminMaintenanceHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	cache.SetWithTTL("maintenance", true, -1)
	return nil
}

func getAdminMaintenanceHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	var m bool
	cache.Get("maintenance", &m)
	return WriteJSON(w, r, m, http.StatusOK)
}

func deleteAdminMaintenanceHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	cache.Delete("maintenance")
	return nil
}
