package api

import (
	"context"
	"net/http"

	"github.com/ovh/cds/sdk/log"
)

func (api *API) adminTruncateWarningsHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		if _, err := api.mustDB().Exec("delete from warning"); err != nil {
			log.Warning("adminTruncateWarningsHandler> Unable to truncate warning : %s", err)
			return err
		}
		return nil
	}
}

func (api *API) postAdminMaintenanceHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		api.Cache.SetWithTTL("maintenance", true, -1)
		return nil
	}
}

func (api *API) getAdminMaintenanceHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		var m bool
		api.Cache.Get("maintenance", &m)
		return WriteJSON(w, r, m, http.StatusOK)
	}
}

func (api *API) deleteAdminMaintenanceHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		api.Cache.Delete("maintenance")
		return nil
	}
}
