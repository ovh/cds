package api

import (
	"context"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/database/dbmigrate"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
)

func (api *API) postMigrationUnlockedHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		id := vars["id"]

		if len(id) == 0 {
			return sdk.NewErrorFrom(sdk.ErrWrongRequest, "Id is mandatory. Check id from table gorp_migrations_lock")
		}

		return dbmigrate.UnlockMigrate(api.mustDB().Db, id)
	}
}
