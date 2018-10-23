package api

import (
	"bytes"
	"context"
	"net/http"

	"github.com/ovh/cds/engine/api/event"
	"github.com/ovh/cds/engine/api/mail"
	"github.com/ovh/cds/engine/api/metrics"
	"github.com/ovh/cds/engine/api/objectstore"
	"github.com/ovh/cds/engine/api/repositoriesmanager"
	"github.com/ovh/cds/engine/api/scheduler"
	"github.com/ovh/cds/engine/api/services"
	"github.com/ovh/cds/engine/api/sessionstore"
	"github.com/ovh/cds/engine/api/worker"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
)

// VersionHandler returns version of current uservice
func VersionHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		return service.WriteJSON(w, sdk.VersionCurrent(), http.StatusOK)
	}
}

// Status returns status, implements interface service.Service
func (api *API) Status() sdk.MonitoringStatus {
	m := api.CommonMonitoring()

	m.Lines = append(m.Lines, getStatusLine(sdk.MonitoringStatusLine{Component: "Hostname", Value: event.GetHostname(), Status: sdk.MonitoringStatusOK}))
	m.Lines = append(m.Lines, getStatusLine(sdk.MonitoringStatusLine{Component: "CDSName", Value: event.GetCDSName(), Status: sdk.MonitoringStatusOK}))
	m.Lines = append(m.Lines, getStatusLine(api.Router.StatusPanic()))
	m.Lines = append(m.Lines, getStatusLine(scheduler.Status()))
	m.Lines = append(m.Lines, getStatusLine(event.Status()))
	m.Lines = append(m.Lines, getStatusLine(repositoriesmanager.EventsStatus(api.Cache)))
	m.Lines = append(m.Lines, getStatusLine(api.Cache.Status()))
	m.Lines = append(m.Lines, getStatusLine(sessionstore.Status))
	m.Lines = append(m.Lines, getStatusLine(objectstore.Status()))
	m.Lines = append(m.Lines, getStatusLine(mail.Status()))
	m.Lines = append(m.Lines, getStatusLine(api.DBConnectionFactory.Status()))
	m.Lines = append(m.Lines, getStatusLine(worker.Status(api.mustDB())))

	return m
}

func getStatusLine(s sdk.MonitoringStatusLine) sdk.MonitoringStatusLine {
	return s
}

func (api *API) statusHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		var status = http.StatusOK
		if api.Router.panicked {
			status = http.StatusServiceUnavailable
		}

		srvs, err := services.All(api.mustDB())
		if err != nil {
			return err
		}

		mStatus := metrics.ComputeGlobalStatus(srvs)
		return service.WriteJSON(w, mStatus, status)
	}
}

func (api *API) smtpPingHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		if getUser(ctx) == nil {
			return sdk.ErrForbidden
		}

		message := "mail sent"
		if err := mail.SendEmail("Ping", bytes.NewBufferString("Pong"), getUser(ctx).Email, false); err != nil {
			message = err.Error()
		}

		return service.WriteJSON(w, map[string]string{"message": message}, http.StatusOK)
	}
}
