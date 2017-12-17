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
		t := time.Now()
		output := sdk.MonitoringStatus{Now: t}

		output.Lines = append(output.Lines, getStatusLine(sdk.MonitoringStatusLine{Component: "Version", Value: sdk.VERSION, Status: sdk.MonitoringStatusOK}))
		output.Lines = append(output.Lines, getStatusLine(sdk.MonitoringStatusLine{Component: "Uptime", Value: fmt.Sprintf("%s", time.Since(api.StartupTime)), Status: sdk.MonitoringStatusOK}))
		output.Lines = append(output.Lines, getStatusLine(sdk.MonitoringStatusLine{Component: "Hostname", Value: event.GetHostname(), Status: sdk.MonitoringStatusOK}))
		output.Lines = append(output.Lines, getStatusLine(sdk.MonitoringStatusLine{Component: "CDSName", Value: event.GetCDSName(), Status: sdk.MonitoringStatusOK}))
		output.Lines = append(output.Lines, getStatusLine(sdk.MonitoringStatusLine{Component: "Time", Value: fmt.Sprintf("%dh%dm%ds", t.Hour(), t.Minute(), t.Second()), Status: sdk.MonitoringStatusOK}))
		output.Lines = append(output.Lines, getStatusLine(api.Router.StatusPanic()))
		output.Lines = append(output.Lines, getStatusLine(scheduler.Status()))
		output.Lines = append(output.Lines, getStatusLine(event.Status()))
		output.Lines = append(output.Lines, getStatusLine(repositoriesmanager.EventsStatus(api.Cache)))
		output.Lines = append(output.Lines, getStatusLine(api.Cache.Status()))
		output.Lines = append(output.Lines, getStatusLine(sessionstore.Status))
		output.Lines = append(output.Lines, getStatusLine(objectstore.Status()))
		output.Lines = append(output.Lines, getStatusLine(mail.Status()))
		output.Lines = append(output.Lines, getStatusLine(api.DBConnectionFactory.Status()))
		output.Lines = append(output.Lines, getStatusLine(sdk.MonitoringStatusLine{Component: "LastUpdate Connected", Value: fmt.Sprintf("%d", len(api.lastUpdateBroker.clients)), Status: sdk.MonitoringStatusOK}))
		output.Lines = append(output.Lines, getStatusLine(worker.Status(api.mustDB())))

		var status = http.StatusOK
		if api.Router.panicked {
			status = http.StatusServiceUnavailable
		}
		return WriteJSON(w, r, output, status)
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
		if err := mail.SendEmail("Ping", bytes.NewBufferString("Pong"), getUser(ctx).Email); err != nil {
			message = err.Error()
		}

		return WriteJSON(w, r, map[string]string{"message": message}, http.StatusOK)
	}
}
