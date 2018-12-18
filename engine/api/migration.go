package api

import (
	"context"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/ovh/cds/sdk"

	"github.com/ovh/cds/engine/api/migrate"
	"github.com/ovh/cds/engine/service"
)

func (api *API) getAdminMigrationsHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		migrations, err := migrate.GetAll(api.mustDB())
		if err != nil {
			return err
		}
		return service.WriteJSON(w, migrations, http.StatusOK)
	}
}

func (api *API) postAdminMigrationCancelHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		id := vars["id"]

		migrationID, err := strconv.ParseInt(id, 10, 64)
		if err != nil {
			return sdk.NewError(sdk.Error{Status: http.StatusBadRequest, Message: "Incorrect migration id"}, err)
		}

		if err := migrate.UpdateStatus(api.mustDB(), migrationID, sdk.MigrationStatusCanceled); err != nil {
			return err
		}

		return nil
	}
}
