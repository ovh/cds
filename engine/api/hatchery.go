package main

import (
	"net/http"

	"github.com/go-gorp/gorp"
	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/context"
	"github.com/ovh/cds/engine/api/hatchery"
	"github.com/ovh/cds/engine/api/worker"
	"github.com/ovh/cds/engine/log"
	"github.com/ovh/cds/sdk"
)

func registerHatchery(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	// Unmarshal body
	hatch := sdk.Hatchery{}
	if err := UnmarshalBody(r, &hatch); err != nil {
		return err
	}

	// Load token
	tk, err := worker.LoadToken(db, hatch.UID)
	if err != nil {
		log.Warning("registerHatchery: Invalid token> %s\n", err)
		return sdk.ErrUnauthorized
	}
	hatch.GroupID = tk.GroupID

	if err = hatchery.InsertHatchery(db, &hatch); err != nil {
		if err != sdk.ErrModelNameExist {
			log.Warning("registerHatchery> Cannot insert new hatchery: %s\n", err)
		}
		log.Warning("registerHatchery> Error %s", err)
		return err
	}

	log.Info("registerHatchery> Welcome %d", hatch.ID)

	return WriteJSON(w, r, hatch, http.StatusOK)
}

func refreshHatcheryHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	vars := mux.Vars(r)
	hatcheryID := vars["id"]

	if err := hatchery.RefreshHatchery(db, hatcheryID); err != nil {
		log.Warning("refreshHatcheryHandler> cannot refresh last beat of %s: %s\n", hatcheryID, err)
		return err
	}

	return nil
}
