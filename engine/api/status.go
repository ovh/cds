package api

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/artifact"
	"github.com/ovh/cds/engine/api/event"
	"github.com/ovh/cds/engine/api/mail"
	"github.com/ovh/cds/engine/api/objectstore"
	"github.com/ovh/cds/engine/api/repositoriesmanager"
	"github.com/ovh/cds/engine/api/services"
	"github.com/ovh/cds/engine/api/worker"
	"github.com/ovh/cds/sdk"
)

func (api *API) getVersionHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		s := sdk.Version{Version: sdk.VERSION}
		return WriteJSON(w, r, s, http.StatusOK)
	}
}

func (api *API) statusHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		eventStatusLabel, eventStatusValue, eventStatusHealthy, eventStatusError := event.Status()
		cacheStatusLabel, cacheStatusValue, cacheStatusHealthy, cacheStatusError := api.Cache.Status()
		sessionStoreLabel, sessionStoreValue, sessionStoreHealthy, sessionstoreError := api.Router.AuthDriver.Store().Status()
		storageLabel, storageValue, storageHealthy, storageError := objectstore.Status()
		smtpLabel, smtpValue, smtpHealthy, smtpError := mail.Status()
		dbLabel, dbValue, dbHealthy, dbError := api.DBConnectionFactory.Status()

		var modelValue, hooksValue string
		var modelHealthy, hooksHealthy bool
		var modelError, hooksError error
		if db := api.DBConnectionFactory.GetDBMap(); db != nil {
			modelValue, modelHealthy, modelError = worker.Status(db)
			artiSize := artifact.TotalSize(db)
			if artiSize > 0 {
				storageValue += fmt.Sprintf(" (%d)", artiSize)
			}

			repo := services.NewRepository(func() *gorp.DbMap { return db }, api.Cache)
			hooks, err := repo.FindByType("hooks")
			if err != nil {
				hooksValue = fmt.Sprintf("Hooks unavailable: %v", err)
				hooksError = err
				hooksHealthy = false
			} else if len(hooks) == 0 {
				hooksValue = fmt.Sprintf("No Hooks available")
				hooksError = fmt.Errorf("No Hooks available")
				hooksHealthy = true
			} else {
				hooksHealthy = true
				hooksError = nil
				hooksValue = fmt.Sprintf("%s (%d)", hooks[0].HTTPURL, len(hooks))
			}

		}

		s := ServiceStatus{
			Version: sdk.VERSION,
			Uptime:  fmt.Sprintf("%.2f hours", time.Since(api.StartupTime).Hours()),
			Router: RouterStatus{
				NbCall:         api.Router.nbCall,
				NbErrors:       api.Router.nbCallInError,
				NbPanics:       api.Router.nbPanic,
				Healthy:        !api.Router.panicked,
				Warning:        float64(api.Router.nbCallInError+api.Router.nbCallInError)/float64(api.Router.nbCall)*100 > 20,
				LastUpdateUser: int64(len(api.lastUpdateBroker.clients)),
			},
			EventQueue: Status{
				Label:   eventStatusLabel,
				Value:   eventStatusValue,
				Healthy: eventStatusHealthy,
				Warning: eventStatusError != nil,
			},
			InternalEventQueue: Status{
				Label:   "Internal Events Queue",
				Value:   fmt.Sprintf("%d Events", repositoriesmanager.EventsStatus(api.Cache)),
				Healthy: true,
				Warning: repositoriesmanager.EventsStatus(api.Cache) > 10,
			},
			Cache: Status{
				Label:   cacheStatusLabel,
				Value:   cacheStatusValue,
				Healthy: cacheStatusHealthy,
				Warning: cacheStatusError != nil,
			},
			SessionStore: Status{
				Label:   sessionStoreLabel,
				Value:   sessionStoreValue,
				Healthy: sessionStoreHealthy,
				Warning: sessionstoreError != nil,
			},
			Mail: Status{
				Label:   smtpLabel,
				Value:   smtpValue,
				Healthy: smtpHealthy,
				Warning: smtpError != nil,
			},
			ObjectStorage: Status{
				Label:   storageLabel,
				Value:   storageValue,
				Healthy: storageHealthy,
				Warning: storageError != nil,
			},
			Database: Status{
				Label:   dbLabel,
				Value:   dbValue,
				Healthy: dbHealthy,
				Warning: dbError != nil,
			},
			Models: Status{
				Label:   "Models",
				Value:   modelValue,
				Healthy: modelHealthy,
				Warning: modelError != nil,
			},
			Hooks: Status{
				Label:   "Hooks",
				Value:   hooksValue,
				Healthy: hooksHealthy,
				Warning: hooksError != nil,
			},
		}

		var status = http.StatusOK
		if api.Router.panicked {
			status = http.StatusServiceUnavailable
		}

		return WriteJSON(w, r, s, status)
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
