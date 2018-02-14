package api

import (
	"context"
	"net/http"

	"github.com/ovh/cds/sdk"
)

// ConfigUserHandler return url of CDS UI
func (api *API) ConfigUserHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		return WriteJSON(w, map[string]string{sdk.ConfigURLUIKey: api.Config.URL.UI}, http.StatusOK)
	}
}
