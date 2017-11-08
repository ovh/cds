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
		tk, err := token.LoadToken(api.mustDB(), hatch.UID)
		if err != nil {
			return sdk.WrapError(sdk.ErrUnauthorized, "registerHatcheryHandler> Invalid token")
		}
		hatch.GroupID = tk.GroupID

		oldH, errL := hatchery.LoadHatcheryByNameAndToken(api.mustDB(), hatch.Name, tk.Token)
		if errL != nil && errL != sdk.ErrNoHatchery {
			return sdk.WrapError(err, "registerHatcheryHandler> Cannot load hatchery %s", hatch.Name)
		}

		if oldH != nil {
			hatch.ID = oldH.ID
			if err := hatchery.Update(api.mustDB(), hatch); err != nil {
				return sdk.WrapError(err, "registerHatcheryHandler> Cannot insert new hatchery")
			}
		} else {
			if err := hatchery.InsertHatchery(api.mustDB(), &hatch); err != nil {
				return sdk.WrapError(err, "registerHatcheryHandler> Cannot insert new hatchery")
			}
		}

		hatch.Uptodate = hatch.Version == sdk.VERSION

		log.Debug("registerHatcheryHandler> Welcome %d", hatch.ID)
		return WriteJSON(w, r, hatch, http.StatusOK)
	}
}

func (api *API) refreshHatcheryHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		hatcheryID := vars["id"]

		if err := hatchery.RefreshHatchery(api.mustDB(), hatcheryID); err != nil {
			return sdk.WrapError(err, "refreshHatcheryHandler> cannot refresh last beat of %s", hatcheryID)
		}
		return nil
	}
}
