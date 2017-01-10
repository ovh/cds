package main

import (
	"bytes"
	"database/sql"
	"fmt"
	"net/http"
	"time"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/context"
	"github.com/ovh/cds/engine/api/database"
	"github.com/ovh/cds/engine/api/internal"
	"github.com/ovh/cds/engine/api/mail"
	"github.com/ovh/cds/engine/api/objectstore"
	"github.com/ovh/cds/engine/api/repositoriesmanager/polling"
	"github.com/ovh/cds/engine/api/sessionstore"
	"github.com/ovh/cds/engine/log"

	"github.com/ovh/cds/engine/api/secret"
	"github.com/ovh/cds/sdk"
)

func getError(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {
	WriteError(w, r, sdk.ErrInvalidProjectKey)
}

func getVersionHandler(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {

	s := struct {
		Version string `json:"version"`
	}{
		Version: internal.VERSION,
	}

	WriteJSON(w, r, s, http.StatusOK)
}

func statusHandler(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {
	var output []string

	// TODO: CHECK IF USER IS ADMIN

	// Version
	output = append(output, fmt.Sprintf("Version: %s", internal.VERSION))

	// Uptime
	output = append(output, fmt.Sprintf("Uptime: %s", time.Since(startupTime)))

	//Nb Panics
	output = append(output, fmt.Sprintf("Nb of Panics: %d", nbPanic))

	// Check vault
	output = append(output, fmt.Sprintf("Secret Backend: %s", secret.Status()))

	// Check redis
	output = append(output, fmt.Sprintf("Cache: %s", cache.Status))

	// Check session-store
	output = append(output, fmt.Sprintf("Session-Store: %s", sessionstore.Status))

	// Check object-store
	output = append(output, fmt.Sprintf("Object-Store: %s", objectstore.Status()))

	//Check smtp
	output = append(output, fmt.Sprintf("SMTP: %s", mail.Status))

	// Check database
	output = append(output, database.Status())

	var status = http.StatusOK
	if panicked {
		status = http.StatusServiceUnavailable
	}
	WriteJSON(w, r, output, status)
}

func pollinStatusHandler(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {
	project := r.FormValue("project")
	application := r.FormValue("application")
	pipeline := r.FormValue("pipeline")

	exec, err := polling.LoadExecutions(db, project, application, pipeline)
	if err != nil {
		log.Warning("Error %s\n", err)
		WriteError(w, r, err)
		return
	}

	WriteJSON(w, r, exec, 200)
	return
}

func smtpPingHandler(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {
	if c.User == nil {
		WriteError(w, r, sdk.ErrForbidden)
		return
	}

	message := "mail sent"
	if err := mail.SendEmail("Ping", bytes.NewBufferString("Pong"), c.User.Email); err != nil {
		message = err.Error()
	}

	WriteJSON(w, r, map[string]string{
		"message": message,
	}, http.StatusOK)
}
