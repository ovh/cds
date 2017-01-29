package main

import (
	"net/http"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/context"
	"github.com/ovh/cds/engine/log"
)

func adminTruncateWarningsHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) {
	if _, err := db.Exec("delete from warning"); err != nil {
		log.Warning("adminTruncateWarningsHandler> Unable to truncate warning : %s", err)
		WriteError(w, r, err)
	}
}

func postAdminMaintenanceHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) {
	cache.SetWithTTL("maintenance", true, -1)
}

func getAdminMaintenanceHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) {
	var m bool
	cache.Get("maintenance", &m)
	WriteJSON(w, r, m, http.StatusOK)
}

func deleteAdminMaintenanceHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) {
	cache.Delete("maintenance")
}
