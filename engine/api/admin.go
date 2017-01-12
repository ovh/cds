package main

import (
	"database/sql"
	"net/http"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/context"
)

func adminTruncateWarningsHandler(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {
	if _, err := db.Exec("truncate warning"); err != nil {
		WriteError(w, r, err)
	}
}

func postAdminMaintenanceHandler(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {
	cache.SetWithTTL("maintenance", true, -1)
}

func getAdminMaintenanceHandler(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {
	var m bool
	cache.Get("maintenance", &m)
	WriteJSON(w, r, m, http.StatusOK)
}

func deleteAdminMaintenanceHandler(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {
	cache.Delete("maintenance")
}
