package api

import (
	"context"
	"github.com/gorilla/mux"
	"net/http"

	"github.com/ovh/cds/engine/api/warning"
	"github.com/ovh/cds/sdk"
)

func (api *API) getWarningsHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars["permProjectKey"]

		al := r.Header.Get("Accept-Language")

		warnings, errW := warning.GetByProject(api.mustDB(), key)
		if errW != nil {
			return sdk.WrapError(errW, "getWarningsHandler> Unable to get warning for project %s", key)
		}

		for i := range warnings {
			w := &warnings[i]
			w.ComputeMessage(al)
		}
		return WriteJSON(w, warnings, http.StatusOK)
	}
}
