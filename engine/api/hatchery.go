package main

import (
	"net/http"

	"github.com/go-gorp/gorp"
	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/businesscontext"
	"github.com/ovh/cds/engine/api/hatchery"
	"github.com/ovh/cds/engine/api/internal"
	"github.com/ovh/cds/engine/api/token"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func registerHatchery(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *businesscontext.Ctx) error {
	hatch := sdk.Hatchery{}
	if err := UnmarshalBody(r, &hatch); err != nil {
		return err
	}

	// Load token
	tk, err := token.LoadToken(db, hatch.UID)
	if err != nil {
		return sdk.WrapError(sdk.ErrUnauthorized, "registerHatchery> Invalid token")
	}
	hatch.GroupID = tk.GroupID

	if err = hatchery.InsertHatchery(db, &hatch); err != nil {
		if err != sdk.ErrModelNameExist {
			return sdk.WrapError(err, "registerHatchery> Cannot insert new hatchery")
		}
		return sdk.WrapError(err, "registerHatchery> Error")
	}

	hatch.Uptodate = hatch.Version == internal.VERSION

	log.Debug("registerHatchery> Welcome %d", hatch.ID)
	return WriteJSON(w, r, hatch, http.StatusOK)
}

func refreshHatcheryHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *businesscontext.Ctx) error {
	vars := mux.Vars(r)
	hatcheryID := vars["id"]

	if err := hatchery.RefreshHatchery(db, hatcheryID); err != nil {
		return sdk.WrapError(err, "refreshHatcheryHandler> cannot refresh last beat of %s", hatcheryID)
	}
	return nil
}
