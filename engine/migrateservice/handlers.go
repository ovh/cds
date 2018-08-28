package migrateservice

import (
	"context"
	"net/http"

	"github.com/ovh/cds/engine/service"
)

func (s *dbmigservice) statusHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		return service.WriteJSON(w, s.Status(), http.StatusOK)
	}
}

func (s *dbmigservice) getMigrationHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		return service.WriteJSON(w, s.currentStatus.migrations, http.StatusOK)
	}
}
