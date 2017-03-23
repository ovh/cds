package main

import (
	"database/sql"
	"net/http"
	"time"

	"github.com/go-gorp/gorp"
	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/context"
	"github.com/ovh/cds/engine/api/database"
	"github.com/ovh/cds/engine/api/hatchery"
	"github.com/ovh/cds/engine/api/internal"
	"github.com/ovh/cds/engine/api/worker"
	"github.com/ovh/cds/engine/log"
	"github.com/ovh/cds/sdk"
)

func registerWorkerHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	// Unmarshal body
	params := &worker.RegistrationForm{}
	if err := UnmarshalBody(r, params); err != nil {
		return sdk.WrapError(err, "registerWorkerHandler> Unable to parse registration form")
	}

	// Check that hatchery exists
	var h *sdk.Hatchery
	if params.Hatchery != 0 {
		if err := hatchery.Exists(db, params.Hatchery); err != nil {
			return sdk.WrapError(err, "registerWorkerHandler> Unable to check if hatchery (%d) exists on register worker %s (model:%d)", params.Hatchery, params.Name, params.Model)
		}

		var errH error
		h, errH = hatchery.LoadHatcheryByID(db, params.Hatchery)
		if errH != nil {
			return sdk.WrapError(errH, "registerWorkerHandler> Unable to load hatchery %d", params.Hatchery)
		}
	}

	// Try to register worker
	worker, err := worker.RegisterWorker(db, params.Name, params.UserKey, params.Model, h, params.BinaryCapabilities)
	if err != nil {
		err = sdk.NewError(sdk.ErrUnauthorized, err)
		return sdk.WrapError(err, "registerWorkerHandler> [%s] Registering failed", params.Name)
	}

	worker.Uptodate = params.Version == internal.VERSION

	log.Debug("New worker: [%s] - %s\n", worker.ID, worker.Name)

	// Return worker info to worker itself
	return WriteJSON(w, r, worker, http.StatusOK)
}

func getOrphanWorker(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	workers, err := worker.LoadWorkersByModel(db, 0)
	if err != nil {
		return sdk.WrapError(err, "getOrphanWorker> Cannot load workers")
	}
	return WriteJSON(w, r, workers, http.StatusOK)
}

func getWorkersHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	if err := r.ParseForm(); err != nil {
		return sdk.WrapError(err, "getWorkerModels> cannot parse form")
	}

	name := r.FormValue("orphan")
	if name == "true" {
		return getOrphanWorker(w, r, db, c)
	}

	workers, errl := worker.LoadWorkers(db)
	if errl != nil {
		return sdk.WrapError(errl, "getWorkerModels> cannot load workers")
	}

	return WriteJSON(w, r, workers, http.StatusOK)
}

func disableWorkerHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	// Get pipeline and action name in URL
	vars := mux.Vars(r)
	id := vars["id"]

	tx, err := db.Begin()
	if err != nil {
		return sdk.WrapError(err, "disabledWorkerHandler> Cannot start tx")
	}
	defer tx.Rollback()

	wor, err := worker.LoadWorker(tx, id)
	if err != nil {
		if err != sql.ErrNoRows {
			return sdk.WrapError(err, "disabledWorkerHandler> Cannot load worker %s", id)
		}
		return sdk.WrapError(sdk.ErrNotFound, "disabledWorkerHandler> Cannot load worker %s", id)
	}

	if wor.Status == sdk.StatusBuilding {
		return sdk.WrapError(sdk.ErrForbidden, "Cannot disable a worker with status %s\n", wor.Status)
	}

	if wor.Status == sdk.StatusChecking {
		log.Warning("disableWorkerHandler> Next time, we will see (%s) %s at status waiting, we will kill it\n", wor.ID, wor.Name)
		go func(w *sdk.Worker) {
			for {
				var attempts int
				time.Sleep(500 * time.Millisecond)
				db := database.DBMap(database.DB())
				if db != nil {
					attempts++
					w1, err := worker.LoadWorker(db, w.ID)
					if err != nil {
						log.Warning("disableWorkerHandler> Error getting worker %s", w.ID)
						return
					}
					//Give up is worker is building
					if w1.Status == sdk.StatusBuilding {
						return
					}
					if w1.Status == sdk.StatusWaiting {
						if err := worker.UpdateWorkerStatus(tx, id, sdk.StatusDisabled); err != nil {
							log.Warning("disableWorkerHandler> Error disabling worker %s", w.ID)
							return
						}
					}
					if attempts > 100 {
						log.Critical("disableWorkerHandler> Unable to disabled worker %s %s", w.ID, w.Name)
						return
					}
				}
			}
		}(wor)
	}

	if wor.HatcheryID == 0 {
		return sdk.WrapError(sdk.ErrForbidden, "disableWorkerHandler> Cannot disable a worker (%s) not started by an hatchery", wor.Name)
	}

	if err := worker.UpdateWorkerStatus(tx, id, sdk.StatusDisabled); err != nil {
		if err == worker.ErrNoWorker || err == sql.ErrNoRows {
			return sdk.WrapError(sdk.ErrWrongRequest, "disableWorkerHandler> handler %s does not exists", id)
		}
		return sdk.WrapError(err, "disableWorkerHandler> cannot update worker status")
	}

	if err := tx.Commit(); err != nil {
		return sdk.WrapError(err, "disableWorkerHandler> cannot commit tx")
	}

	return nil
}

func refreshWorkerHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	if err := worker.RefreshWorker(db, c.Worker.ID); err != nil && (err != sql.ErrNoRows || err != worker.ErrNoWorker) {
		return sdk.WrapError(err, "refreshWorkerHandler> cannot refresh last beat of %s", c.Worker.ID)
	}
	return nil
}

func unregisterWorkerHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	if err := worker.DeleteWorker(db, c.Worker.ID); err != nil {
		return sdk.WrapError(err, "unregisterWorkerHandler> cannot delete worker %s", c.Worker.ID)
	}
	return nil
}

func workerCheckingHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	wk, errW := worker.LoadWorker(db, c.Worker.ID)
	if errW != nil {
		return sdk.WrapError(errW, "workerCheckingHandler> Unable to load worker %s", c.Worker.ID)
	}

	if wk.Status != sdk.StatusWaiting {
		log.Info("workerCheckingHandler> Worker %s cannot be Checking. Current status: %s", wk.Name, wk.Status)
		return nil
	}

	if err := worker.SetStatus(db, c.Worker.ID, sdk.StatusChecking); err != nil {
		return sdk.WrapError(err, "workerCheckingHandler> cannot update worker %s", c.Worker.ID)
	}

	return nil
}

func workerWaitingHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	wk, errW := worker.LoadWorker(db, c.Worker.ID)
	if errW != nil {
		return sdk.WrapError(errW, "workerWaitingHandler> Unable to load worker %s", c.Worker.ID)
	}

	if wk.Status == sdk.StatusWaiting {
		return nil
	}

	if wk.Status != sdk.StatusChecking && wk.Status != sdk.StatusBuilding {
		log.Info("workerWaitingHandler> Worker %s cannot be Waiting. Current status: %s", wk.Name, wk.Status)
		return nil
	}

	if err := worker.SetStatus(db, c.Worker.ID, sdk.StatusWaiting); err != nil {
		return sdk.WrapError(err, "workerWaitingHandler> cannot update worker %s", c.Worker.ID)
	}

	return nil
}
