package api

import (
	"context"
	"net/http"

	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/hatchery"
	"github.com/ovh/cds/engine/api/token"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func (api *API) registerHatcheryHandler() service.Handler {
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
		hatch.IsSharedInfra = tk.GroupID == group.SharedInfraGroup.ID

		oldH, errL := hatchery.LoadHatcheryByNameAndToken(api.mustDB(), hatch.Name, tk.Token)
		if errL != nil && errL != sdk.ErrNoHatchery {
			return sdk.WrapError(err, "registerHatcheryHandler> Cannot load hatchery %s", hatch.Name)
		}

		tx, errBegin := api.mustDB().Begin()
		defer tx.Rollback()
		if errBegin != nil {
			return sdk.WrapError(errBegin, "registerHatcheryHandler> Cannot start tx")
		}

		if oldH != nil {
			hatch.ID = oldH.ID
			hatch.Model.ID = oldH.Model.ID
			if err := hatchery.Update(tx, hatch); err != nil {
				return sdk.WrapError(err, "registerHatcheryHandler> Cannot update existing hatchery")
			}
		} else {
			if err := hatchery.InsertHatchery(tx, &hatch); err != nil {
				return sdk.WrapError(err, "registerHatcheryHandler> Cannot insert new hatchery")
			}
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "registerHatcheryHandler> Cannot commit transaction")
		}

		hatch.Uptodate = hatch.Version == sdk.VERSION

		log.Debug("registerHatcheryHandler> Welcome hatchery %s %d group:%d sharedInfra:%t", hatch.Type, hatch.ID, hatch.GroupID, hatch.IsSharedInfra)
		return service.WriteJSON(w, hatch, http.StatusOK)
	}
}

func (api *API) hatcheryCountHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		wfNodeRunID, err := requestVarInt(r, "workflowNodeRunID")
		if err != nil {
			return sdk.WrapError(err, "cannot convert workflow node run ID")
		}

		count, err := hatchery.CountHatcheries(api.mustDB(), wfNodeRunID)
		if err != nil {
			return sdk.WrapError(err, "hatcheryCountHandler> cannot get hatcheries count")
		}

		return service.WriteJSON(w, count, http.StatusOK)
	}
}
