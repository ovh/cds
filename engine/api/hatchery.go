package api

import (
	"context"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/hatchery"
	"github.com/ovh/cds/engine/api/token"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func (api *API) registerHatcheryHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		hatch := sdk.Hatchery{}
		if err := UnmarshalBody(r, &hatch); err != nil {
			return err
		}

		// Load token
		tk, err := token.LoadToken(api.MustDB(), hatch.UID)
		if err != nil {
			return sdk.WrapError(sdk.ErrUnauthorized, "registerHatchery> Invalid token")
		}
		hatch.GroupID = tk.GroupID

		if err = hatchery.InsertHatchery(api.MustDB(), &hatch); err != nil {
			if err != sdk.ErrModelNameExist {
				return sdk.WrapError(err, "registerHatchery> Cannot insert new hatchery")
			}
			return sdk.WrapError(err, "registerHatchery> Error")
		}

		hatch.Uptodate = hatch.Version == sdk.VERSION

		log.Debug("registerHatchery> Welcome %d", hatch.ID)
		return WriteJSON(w, r, hatch, http.StatusOK)
	}
}

func (api *API) refreshHatcheryHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		hatcheryID := vars["id"]

		if err := hatchery.RefreshHatchery(api.MustDB(), hatcheryID); err != nil {
			return sdk.WrapError(err, "refreshHatcheryHandler> cannot refresh last beat of %s", hatcheryID)
		}
		return nil
	}
}
