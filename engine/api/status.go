package main

import (
	"bytes"
	"fmt"
	"net/http"
	"time"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/context"
	"github.com/ovh/cds/engine/api/database"
	"github.com/ovh/cds/engine/api/event"
	"github.com/ovh/cds/engine/api/internal"
	"github.com/ovh/cds/engine/api/mail"
	"github.com/ovh/cds/engine/api/objectstore"
	"github.com/ovh/cds/engine/api/scheduler"
	"github.com/ovh/cds/engine/api/secret"
	"github.com/ovh/cds/engine/api/sessionstore"
	"github.com/ovh/cds/engine/log"
	"github.com/ovh/cds/sdk"
)

func getError(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	return sdk.ErrInvalidProjectKey
}

func getVersionHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	s := struct {
		Version string `json:"version"`
	}{
		Version: internal.VERSION,
	}

	return WriteJSON(w, r, s, http.StatusOK)
}

func statusHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	var output []string

	// Version
	output = append(output, fmt.Sprintf("Version: %s", internal.VERSION))
	log.Info("Status> Version: %s", internal.VERSION)

	// Uptime
	output = append(output, fmt.Sprintf("Uptime: %s", time.Since(startupTime)))
	log.Info("Status> Uptime: %s", time.Since(startupTime))

	//Nb Panics
	output = append(output, fmt.Sprintf("Nb of Panics: %d", nbPanic))
	log.Info("Status> Nb of Panics: %d", nbPanic)

	// Check vault
	output = append(output, fmt.Sprintf("Secret Backend: %s", secret.Status()))
	log.Info("Status> Secret Backend: %s", secret.Status())

	// Check Scheduler
	output = append(output, fmt.Sprintf("Scheduler: %s", scheduler.Status()))
	log.Info("Status> Scheduler: %s", scheduler.Status())

	// Check Event
	output = append(output, fmt.Sprintf("Event: %s", event.Status()))
	log.Info("Status> Event: %s", event.Status())

	// Check redis
	output = append(output, fmt.Sprintf("Cache: %s", cache.Status))
	log.Info("Status> Cache: %s", cache.Status)

	// Check session-store
	output = append(output, fmt.Sprintf("Session-Store: %s", sessionstore.Status))
	log.Info("Status> Session-Store: %s", sessionstore.Status)

	// Check object-store
	output = append(output, fmt.Sprintf("Object-Store: %s", objectstore.Status()))
	log.Info("Status> Object-Store: %s", objectstore.Status())

	// Check mail
	mailStatus := mail.Status()
	output = append(output, fmt.Sprintf("SMTP: %s", mailStatus))
	log.Info("Status> SMTP: %s", mailStatus)

	// Check database
	output = append(output, database.Status())
	log.Info("Status> %s", database.Status())

	var status = http.StatusOK
	if panicked {
		status = http.StatusServiceUnavailable
	}
	return WriteJSON(w, r, output, status)
}

func smtpPingHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	if c.User == nil {
		return sdk.ErrForbidden
	}

	message := "mail sent"
	if err := mail.SendEmail("Ping", bytes.NewBufferString("Pong"), c.User.Email); err != nil {
		message = err.Error()
	}

	return WriteJSON(w, r, map[string]string{
		"message": message,
	}, http.StatusOK)
}
