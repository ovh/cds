package repositories

import (
	"context"
	"net/http"

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
		op.Status = sdk.OperationStatusPending
		if err := s.dao.saveOperation(op); err != nil {
			return err
		}

		if err := s.dao.pushOperation(op); err != nil {
			return err
		}

		return api.WriteJSON(w, r, op, http.StatusAccepted)
	}
}

func (s *Service) getOperationsHandler() api.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		uuid := muxVar(r, "uuid")

		op := s.dao.loadOperation(uuid)

		return api.WriteJSON(w, r, op, http.StatusOK)
	}
}

func (s *Service) getStatusHandler() api.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		return nil
	}
}
