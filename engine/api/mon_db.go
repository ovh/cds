package api

import (
	"context"
	"net/http"

	"github.com/rubenv/sql-migrate"

	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
)

func (api *API) getMonDBStatusMigrateHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		records, err := migrate.GetMigrationRecords(api.mustDB().Db, "postgres")
		if err != nil {
			return sdk.WrapError(err, "Cannot GetMigrationRecords")
		}
		m := []sdk.MonDBMigrate{}
		for _, r := range records {
			m = append(m, sdk.MonDBMigrate{ID: r.Id, AppliedAt: r.AppliedAt})
		}
		return service.WriteJSON(w, m, http.StatusOK)
	}
}
