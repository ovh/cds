package api

import (
	"context"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/group"
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
		tk, err := token.LoadToken(api.mustDB(ctx), hatch.UID)
		if err != nil {
			return sdk.WrapError(sdk.ErrUnauthorized, "registerHatcheryHandler> Invalid token")
		}
		hatch.GroupID = tk.GroupID
		hatch.IsSharedInfra = tk.GroupID == group.SharedInfraGroup.ID

		oldH, errL := hatchery.LoadHatcheryByNameAndToken(api.mustDB(ctx), hatch.Name, tk.Token)
		if errL != nil && errL != sdk.ErrNoHatchery {
			return sdk.WrapError(err, "registerHatcheryHandler> Cannot load hatchery %s", hatch.Name)
		}

		if oldH != nil {
			hatch.ID = oldH.ID
			hatch.Model.ID = oldH.Model.ID
			if err := hatchery.Update(api.mustDB(ctx), hatch); err != nil {
				return sdk.WrapError(err, "registerHatcheryHandler> Cannot insert new hatchery")
			}
		} else {
			if err := hatchery.InsertHatchery(api.mustDB(ctx), &hatch); err != nil {
				return sdk.WrapError(err, "registerHatcheryHandler> Cannot insert new hatchery")
			}
		}

		hatch.Uptodate = hatch.Version == sdk.VERSION

		log.Debug("registerHatcheryHandler> Welcome %d", hatch.ID)
		return WriteJSON(w, hatch, http.StatusOK)
	}
}

func (api *API) refreshHatcheryHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		hatcheryID := vars["id"]

		if err := hatchery.RefreshHatchery(api.mustDB(ctx), hatcheryID); err != nil {
			return sdk.WrapError(err, "refreshHatcheryHandler> cannot refresh last beat of %s", hatcheryID)
		}
		return nil
	}
}

func (api *API) hatcheryCountHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		wfNodeRunID, err := requestVarInt(r, "workflowNodeRunID")
		if err != nil {
			return sdk.WrapError(err, "cannot convert workflow node run ID")
		}

		count, err := hatchery.CountHatcheries(api.mustDB(ctx), wfNodeRunID)
		if err != nil {
			return sdk.WrapError(err, "hatcheryCountHandler> cannot get hatcheries count")
		}

		return WriteJSON(w, count, http.StatusOK)
	}
}
