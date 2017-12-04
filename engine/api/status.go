package api

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/ovh/cds/engine/api/event"
	"github.com/ovh/cds/engine/api/mail"
	"github.com/ovh/cds/engine/api/objectstore"
	"github.com/ovh/cds/engine/api/repositoriesmanager"
	"github.com/ovh/cds/engine/api/scheduler"
	"github.com/ovh/cds/engine/api/sessionstore"
	"github.com/ovh/cds/engine/api/worker"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

// VersionHandler returns version of current uservice
func VersionHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		s := sdk.Version{Version: sdk.VERSION}
		return WriteJSON(w, r, s, http.StatusOK)
	}
}

func (api *API) statusHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		var output []string

		// Version
		output = append(output, fmt.Sprintf("Version: %s", sdk.VERSION))
		log.Debug("Status> Version: %s", sdk.VERSION)

		// Uptime
		output = append(output, fmt.Sprintf("Uptime: %s", time.Since(api.StartupTime)))
		log.Debug("Status> Uptime: %s", time.Since(api.StartupTime))

		output = append(output, fmt.Sprintf("Hostname: %s", event.GetHostname()))
		log.Debug("Status> Hostname: %s", event.GetHostname())

		output = append(output, fmt.Sprintf("CDSName: %s", event.GetCDSName()))
		log.Debug("Status> CDSName: %s", event.GetCDSName())

		t := time.Now()
		output = append(output, fmt.Sprintf("Time: %dh%dm%ds", t.Hour(), t.Minute(), t.Second()))
		log.Debug("Status> Time:  %dh%dm%ds", t.Hour(), t.Minute(), t.Second())

		//Nb Panics
		output = append(output, fmt.Sprintf("Nb of Panics: %d", api.Router.nbPanic))
		log.Debug("Status> Nb of Panics: %d", api.Router.nbPanic)

		// Check Scheduler
		output = append(output, fmt.Sprintf("Scheduler: %s", scheduler.Status()))
		log.Debug("Status> Scheduler: %s", scheduler.Status())

		// Check Event
		output = append(output, fmt.Sprintf("Event: %s", event.Status()))
		log.Debug("Status> Event: %s", event.Status())

		// Check Event
		output = append(output, fmt.Sprintf("Internal Events Queue: %s", repositoriesmanager.EventsStatus(api.Cache)))
		log.Debug("Status> Internal Events Queue: %s", repositoriesmanager.EventsStatus(api.Cache))

		// Check redis
		output = append(output, fmt.Sprintf("Cache: %s", api.Cache.Status()))
		log.Debug("Status> Cache: %s", api.Cache.Status())

		// Check session-store
		output = append(output, fmt.Sprintf("Session-Store: %s", sessionstore.Status))
		log.Debug("Status> Session-Store: %s", sessionstore.Status)

		// Check object-store
		output = append(output, fmt.Sprintf("Object-Store: %s", objectstore.Status()))
		log.Debug("Status> Object-Store: %s", objectstore.Status())

		// Check mail
		mailStatus := mail.Status()
		output = append(output, fmt.Sprintf("SMTP: %s", mailStatus))
		log.Debug("Status> SMTP: %s", mailStatus)

		// Check database
		output = append(output, api.DBConnectionFactory.Status())
		log.Debug("Status> %s", api.DBConnectionFactory.Status())

		// Check LastUpdate Connected User
		output = append(output, fmt.Sprintf("LastUpdate Connected: %d", len(api.lastUpdateBroker.clients)))
		log.Debug("Status> LastUpdate ConnectedUser> %d", len(api.lastUpdateBroker.clients))

		// Check Worker Model Error
		wmStatus := worker.Status(api.mustDB())
		output = append(output, fmt.Sprintf("Worker Model Errors: %s", wmStatus))
		log.Debug("Status> Worker Model Errors: %s", wmStatus)

		var status = http.StatusOK
		if api.Router.panicked {
			status = http.StatusServiceUnavailable
		}
		return WriteJSON(w, r, output, status)
	}
}

func (api *API) smtpPingHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		if getUser(ctx) == nil {
			return sdk.ErrForbidden
		}

		message := "mail sent"
		if err := mail.SendEmail("Ping", bytes.NewBufferString("Pong"), getUser(ctx).Email); err != nil {
			message = err.Error()
		}

		return WriteJSON(w, r, map[string]string{
			"message": message,
		}, http.StatusOK)
	}
}
