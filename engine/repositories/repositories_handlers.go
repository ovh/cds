package repositories

import (
	"context"
	"net/http"
	"time"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
)

func muxVar(r *http.Request, s string) string {
	vars := mux.Vars(r)
	return vars[s]
}

func (s *Service) postOperationHandler() service.Handler {
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

		return service.WriteJSON(w, op, http.StatusAccepted)
	}
}

func (s *Service) getOperationsHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		uuid := muxVar(r, "uuid")

		op := s.dao.loadOperation(uuid)

		return service.WriteJSON(w, op, http.StatusOK)
	}
}

// Status returns sdk.MonitoringStatus, implements interface service.Service
func (s *Service) Status() sdk.MonitoringStatus {
	m := s.CommonMonitoring()
	return m
}

func (s *Service) getStatusHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		var status = http.StatusOK
		return service.WriteJSON(w, s.Status(), status)
	}
}
