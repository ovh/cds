package repositories

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api"
	"github.com/ovh/cds/sdk"
)

func muxVar(r *http.Request, s string) string {
	vars := mux.Vars(r)
	return vars[s]
}

func (s *Service) postOperationHandler() api.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		uuid := sdk.UUID()
		op := new(sdk.Operation)
		if err := api.UnmarshalBody(r, op); err != nil {
			return err
		}
		op.UUID = uuid
		now := time.Now()
		op.Date = &now
		op.Status = sdk.OperationStatusPending
		if err := s.dao.saveOperation(op); err != nil {
			return err
		}

		if err := s.dao.pushOperation(op); err != nil {
			return err
		}

		return api.WriteJSON(w, op, http.StatusAccepted)
	}
}

func (s *Service) getOperationsHandler() api.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		uuid := muxVar(r, "uuid")

		op := s.dao.loadOperation(uuid)

		return api.WriteJSON(w, op, http.StatusOK)
	}
}

// Status returns sdk.MonitoringStatus, implements interface service.Service
func (s *Service) Status() sdk.MonitoringStatus {
	t := time.Now()
	m := sdk.MonitoringStatus{Now: t}

	m.Lines = append(m.Lines, sdk.MonitoringStatusLine{Component: "Version", Value: sdk.VERSION, Status: sdk.MonitoringStatusOK})
	m.Lines = append(m.Lines, sdk.MonitoringStatusLine{Component: "Uptime", Value: fmt.Sprintf("%s", time.Since(s.StartupTime)), Status: sdk.MonitoringStatusOK})
	m.Lines = append(m.Lines, sdk.MonitoringStatusLine{Component: "Time", Value: fmt.Sprintf("%dh%dm%ds", t.Hour(), t.Minute(), t.Second()), Status: sdk.MonitoringStatusOK})
	return m
}

func (s *Service) getStatusHandler() api.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		var status = http.StatusOK
		return api.WriteJSON(w, s.Status(), status)
	}
}
