package api

import (
	"context"
	"github.com/gorilla/mux"
	"net/http"

	"github.com/ovh/cds/engine/api/event"
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

func (api *API) putWarningsHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars["permProjectKey"]
		warningHash := vars["hash"]

		warn, errW := warning.GetByProjectAndHash(api.mustDB(), key, warningHash)
		if errW != nil {
			return sdk.WrapError(errW, "putWarningsHandler> Unable to get warning %s for project %s", warningHash, key)
		}

		var wa sdk.Warning
		if err := UnmarshalBody(r, &wa); err != nil {
			return sdk.WrapError(err, "putWarningsHandler> Unable to read body")
		}

		// Just update ignored flag
		warn.Ignored = wa.Ignored
		if err := warning.Update(api.mustDB(), warn); err != nil {
			return sdk.WrapError(err, "putWarningsHandler> Unable to update warning")
		}

		event.PublishUpdateWarning(warn, getUser(ctx))
		warn.Message = wa.Message
		return WriteJSON(w, warn, http.StatusOK)
	}
}
