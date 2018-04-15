package api

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"runtime"

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
		s := sdk.Version{
			Version:      sdk.VERSION,
			Architecture: runtime.GOARCH,
			OS:           runtime.GOOS,
		}
		return WriteJSON(w, s, http.StatusOK)
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
	m.Lines = append(m.Lines, getStatusLine(sdk.MonitoringStatusLine{Component: "LastUpdate Connected", Value: fmt.Sprintf("%d", len(api.lastUpdateBroker.clients)), Status: sdk.MonitoringStatusOK}))
	m.Lines = append(m.Lines, getStatusLine(worker.Status(api.mustDB())))

	return m
}

func (api *API) statusHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		var status = http.StatusOK
		if api.Router.panicked {
			status = http.StatusServiceUnavailable
		}
		return WriteJSON(w, api.Status(), status)
	}
}

func getStatusLine(s sdk.MonitoringStatusLine) sdk.MonitoringStatusLine {
	log.Debug("Status> %s", s.String())
	return s
}

func (api *API) smtpPingHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		if getUser(ctx) == nil {
			return sdk.ErrForbidden
		}

		message := "mail sent"
		if err := mail.SendEmail("Ping", bytes.NewBufferString("Pong"), getUser(ctx).Email, false); err != nil {
			message = err.Error()
		}

		return WriteJSON(w, map[string]string{"message": message}, http.StatusOK)
	}
}
