package api

import (
	"context"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/event"
	"github.com/ovh/cds/engine/api/warning"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
)

func (api *API) getWarningsHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars[permProjectKey]

		al := r.Header.Get("Accept-Language")

		warnings, errW := warning.GetByProject(api.mustDB(), key)
		if errW != nil {
			return sdk.WrapError(errW, "getWarningsHandler> Unable to get warning for project %s", key)
		}

		for i := range warnings {
			w := &warnings[i]
			w.ComputeMessage(ctx, al)
		}
		return service.WriteJSON(w, warnings, http.StatusOK)
	}
}

func (api *API) putWarningsHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars[permProjectKey]
		warningHash := vars["hash"]

		warn, errW := warning.GetByProjectAndHash(api.mustDB(), key, warningHash)
		if errW != nil {
			return sdk.WrapError(errW, "putWarningsHandler> Unable to get warning %s for project %s", warningHash, key)
		}

		var wa sdk.Warning
		if err := service.UnmarshalBody(r, &wa); err != nil {
			return sdk.WrapError(err, "Unable to read body")
		}

		// Just update ignored flag
		warn.Ignored = wa.Ignored
		if err := warning.Update(api.mustDB(), warn); err != nil {
			return sdk.WrapError(err, "Unable to update warning")
		}

		event.PublishUpdateWarning(ctx, warn, getAPIConsumer(ctx))
		warn.Message = wa.Message
		return service.WriteJSON(w, warn, http.StatusOK)
	}
}
