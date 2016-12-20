package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/context"
	"github.com/ovh/cds/engine/api/hatchery"
	"github.com/ovh/cds/engine/api/worker"
	"github.com/ovh/cds/engine/log"
	"github.com/ovh/cds/sdk"
)

func registerWorkerHandler(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {
	// Read body
	// Get body
	data, errRead := ioutil.ReadAll(r.Body)
	if errRead != nil {
		WriteError(w, r, sdk.ErrWrongRequest)
		return
	}

	// Unmarshal body
	params := &worker.RegistrationForm{}
	if err := json.Unmarshal(data, params); err != nil {
		fmt.Printf("registerWorkerHandler: Cannot unmarshal parameters: %s\n", err)
		WriteError(w, r, sdk.ErrWrongRequest)
		return
	}

	// Check that hatchery exists
	var h *sdk.Hatchery
	if params.Hatchery != 0 {
		if err := hatchery.Exists(db, params.Hatchery); err != nil {
			WriteError(w, r, err)
			return
		}

		var errH error
		h, errH = hatchery.LoadHatcheryByID(db, params.Hatchery)
		if errH != nil {
			fmt.Printf("registerWorkerHandler> Unable to load hatchery: %s\n", errH)
			WriteError(w, r, errH)
			return
		}
	}

	// Try to register worker
	worker, err := worker.RegisterWorker(db, params.Name, params.UserKey, params.Model, h, params.BinaryCapabilities)
	if err != nil {
		log.Warning("registerWorkerHandler: [%s] Registering failed: %s\n", params.Name, err)
		WriteError(w, r, sdk.ErrUnauthorized)
		return
	}

	// Return worker info to worker itself
	WriteJSON(w, r, worker, http.StatusOK)
	log.Debug("New worker: [%s] - %s\n", worker.ID, worker.Name)
}

func getOrphanWorker(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {
	workers, err := worker.LoadWorkersByModel(db, 0)
	if err != nil {
		log.Warning("getOrphanWorker> Cannot load workers: %s\n", err)
		WriteError(w, r, err)
		return
	}
	WriteJSON(w, r, workers, http.StatusOK)
}

func getWorkersHandler(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {
	err := r.ParseForm()
	if err != nil {
		log.Warning("getWorkerModels> cannot parse form")
		WriteError(w, r, err)
		return
	}

	name := r.FormValue("orphan")
	if name == "true" {
		getOrphanWorker(w, r, db, c)
		return
	}

	workers, err := worker.LoadWorkers(db)
	if err != nil {
		log.Warning("getWorkersHandler> Cannot load workers: %s\n", err)
		WriteError(w, r, err)
		return
	}

	WriteJSON(w, r, workers, http.StatusOK)
}

func disableWorkerHandler(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {
	// Get pipeline and action name in URL
	vars := mux.Vars(r)
	id := vars["id"]

	tx, err := db.Begin()
	if err != nil {
		log.Warning("disabledWorkerHandler> Cannot start tx: %s\n", err)
		WriteError(w, r, sdk.ErrUnknownError)
		return
	}
	defer tx.Rollback()

	wor, err := worker.LoadWorker(tx, id)
	if err != nil {
		if err != sql.ErrNoRows {
			log.Warning("disableWorkerHandler> Cannot load worker: %s\n", err)
		}
		WriteError(w, r, err)
		return
	}

	if wor.Status != sdk.StatusWaiting {
		log.Warning("disableWorkerHandler> Cannot disable a worker with status %s\n", wor.Status)
		WriteError(w, r, sdk.ErrForbidden)
		return
	}

	if wor.HatcheryID == 0 {
		log.Warning("disableWorkerHandler> Cannot disable a worker (%s) not started by an hatchery", wor.Name)
		WriteError(w, r, sdk.ErrForbidden)
		return
	}

	if err := worker.UpdateWorkerStatus(tx, id, sdk.StatusDisabled); err != nil {
		if err == worker.ErrNoWorker || err == sql.ErrNoRows {
			log.Warning("disableWorkerHandler> handler %s does not exists\n", id)
			WriteError(w, r, sdk.ErrWrongRequest)
			return
		}
		log.Warning("disableWorkerHandler> cannot update worker status : %s\n", err)
		WriteError(w, r, err)
		return
	}

	if err := tx.Commit(); err != nil {
		log.Warning("disableWorkerHandler> cannot commit tx: %s\n", err)
		WriteError(w, r, err)
		return
	}
}

func refreshWorkerHandler(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {
	if err := worker.RefreshWorker(db, c.Worker.ID); err != nil && (err != sql.ErrNoRows || err != worker.ErrNoWorker) {
		log.Warning("refreshWorkerHandler> cannot refresh last beat of %s: %s\n", c.Worker.ID, err)
		WriteError(w, r, err)
		return
	}
}

func unregisterWorkerHandler(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {
	if err := worker.DeleteWorker(db, c.Worker.ID); err != nil {
		log.Warning("unregisterWorkerHandler> cannot delete worker %s\n", err)
		WriteError(w, r, err)
		return
	}
}

func workerCheckingHandler(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {
	wk, errW := worker.LoadWorker(db, c.Worker.ID)
	if errW != nil {
		WriteError(w, r, errW)
		return
	}

	if wk.Status != sdk.StatusWaiting {
		log.Warning("workerCheckingHandler> Worker %s cannot be Checking", wk.Name)
		WriteError(w, r, sdk.ErrWrongRequest)
		return
	}

	if err := worker.SetStatus(db, c.Worker.ID, sdk.StatusChecking); err != nil {
		log.Warning("workerCheckingHandler> cannot update worker %s\n", err)
		WriteError(w, r, err)
		return
	}
}

func workerWaitingHandler(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {
	wk, errW := worker.LoadWorker(db, c.Worker.ID)
	if errW != nil {
		WriteError(w, r, errW)
		return
	}

	if wk.Status != sdk.StatusChecking && wk.Status != sdk.StatusBuilding {
		log.Warning("workerWaitingHandler> Worker %s cannot be Waiting", wk.Name)
		WriteError(w, r, sdk.ErrWrongRequest)
		return
	}

	if err := worker.SetStatus(db, c.Worker.ID, sdk.StatusWaiting); err != nil {
		log.Warning("workerWaitingHandler> cannot update worker %s\n", err)
		WriteError(w, r, err)
		return
	}
}
