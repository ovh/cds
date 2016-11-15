package main

import (
	"database/sql"
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/context"
	"github.com/ovh/cds/engine/api/hatchery"
	"github.com/ovh/cds/engine/api/worker"
	"github.com/ovh/cds/engine/log"
	"github.com/ovh/cds/sdk"
)

func registerHatchery(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {

	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		WriteError(w, r, sdk.ErrUnknownError)
		return
	}

	// Unmarshal body
	hatch := sdk.Hatchery{}
	if err = json.Unmarshal(data, &hatch); err != nil {
		log.Warning("registerHatchery: Cannot unmarshal data: %s\n", err)
		WriteError(w, r, sdk.ErrUnknownError)
		return
	}

	// Load token
	tk, err := worker.LoadToken(db, hatch.UID)
	if err != nil {
		log.Warning("registerHatchery: Invalid token> %s\n", err)
		WriteError(w, r, sdk.ErrUnauthorized)
		return
	}
	hatch.GroupID = tk.GroupID

	if err = hatchery.InsertHatchery(db, &hatch); err != nil {
		if err != sdk.ErrModelNameExist {
			log.Warning("registerHatchery> Cannot insert new hatchery: %s\n", err)
		}
		log.Warning("registerHatchery> Error %s", err)
		WriteError(w, r, err)
		return
	}

	log.Info("registerHatchery> Welcome %d", hatch.ID)

	WriteJSON(w, r, hatch, http.StatusOK)
}

func refreshHatcheryHandler(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {
	vars := mux.Vars(r)
	hatcheryID := vars["id"]

	if err := hatchery.RefreshHatchery(db, hatcheryID); err != nil {
		log.Warning("refreshHatcheryHandler> cannot refresh last beat of %s: %s\n", hatcheryID, err)
		WriteError(w, r, err)
		return
	}
}
