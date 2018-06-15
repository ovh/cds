package migrateservice

import (
	"context"
	"net/http"

	"github.com/ovh/cds/engine/api"
)

func (s *dbmigservice) statusHandler() api.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		return api.WriteJSON(w, s.Status(), http.StatusOK)
	}
}

func (s *dbmigservice) getMigrationHandler() api.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		return api.WriteJSON(w, s.currentStatus.migrations, http.StatusOK)
	}
}
