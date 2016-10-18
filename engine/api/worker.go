package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/context"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/hatchery"
	"github.com/ovh/cds/engine/api/sanity"
	"github.com/ovh/cds/engine/api/worker"
	"github.com/ovh/cds/engine/log"
	"github.com/ovh/cds/sdk"
)

func registerWorkerHandler(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {

	// Read body
	// Get body
	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Unmarshal body
	params := &worker.RegistrationForm{}
	err = json.Unmarshal(data, params)
	if err != nil {
		fmt.Printf("registerWorkerHandler: Cannot unmarshal parameters: %s\n", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Check that hatchery exists
	if params.Hatchery != 0 {
		if err := hatchery.Exists(db, params.Hatchery); err != nil {
			WriteError(w, r, err)
			return
		}
	}

	// Try to register worker
	worker, err := worker.RegisterWorker(db, params.Name, params.UserKey, params.Model, params.Hatchery, params.BinaryCapabilities)
	if err != nil {
		log.Warning("registerWorkerHandler: [%s] Registering failed: %s\n", params.Name, err)
		w.WriteHeader(http.StatusUnauthorized)
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
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	WriteJSON(w, r, workers, http.StatusOK)
}

func getWorkersHandler(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {
	err := r.ParseForm()
	if err != nil {
		log.Warning("getWorkerModels> cannot parse form")
		w.WriteHeader(http.StatusInternalServerError)
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
		w.WriteHeader(http.StatusInternalServerError)
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

	err = worker.UpdateWorkerStatus(tx, id, sdk.StatusDisabled)
	if err != nil && (err == worker.ErrNoWorker || err == sql.ErrNoRows) {
		log.Warning("disableWorkerHandler> handler %s does not exists\n", id)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if err != nil {
		log.Warning("disableWorkerHandler> cannot update worker status : %s\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	err = tx.Commit()
	if err != nil {
		log.Warning("disableWorkerHandler> cannot commit tx: %s\n", err)
		WriteError(w, r, err)
		return
	}
}

func refreshWorkerHandler(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {

	err := worker.RefreshWorker(db, c.Worker.ID)
	if err != nil && (err != sql.ErrNoRows || err != worker.ErrNoWorker) {
		log.Warning("refreshWorkerHandler> cannot refresh last beat of %s: %s\n", c.Worker.ID, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

// generateTokenHandler allows a user to generate a token associated to a group permission
// and used by worker to take action from API.
// User generating the token needs to be admin of given group
func generateTokenHandler(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {
	vars := mux.Vars(r)
	groupName := vars["permGroupName"]
	expiration := vars["expiration"]

	exp, err := sdk.ExpirationFromString(expiration)
	if err != nil {
		log.Warning("generateTokenHandler> '%s' -> %s\n", expiration, err)
		WriteError(w, r, err)
		return
	}

	g, err := group.LoadGroup(db, groupName)
	if err != nil {
		log.Warning("generateTokenHandler> cannot load group '%s': %s\n", groupName, err)
		WriteError(w, r, err)
		return
	}

	tk, err := worker.GenerateToken()
	if err != nil {
		log.Warning("generateTokenHandler: cannot generate key: %s\n", err)
		WriteError(w, r, err)
		return
	}

	err = worker.InsertToken(db, g.ID, tk, exp)
	if err != nil {
		log.Warning("generateTokenHandler> cannot insert new key: %s\n", err)
		WriteError(w, r, err)
		return
	}

	s := struct {
		Key string `json:"key"`
	}{
		Key: tk,
	}
	WriteJSON(w, r, s, http.StatusOK)
}

/*
func generateUserKeyHandler(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {
	vars := mux.Vars(r)
	e := vars["expiration"]

	if c.User == nil {
		WriteError(w, r, sdk.ErrUnauthorized)
		return
	}

	exp, err := sdk.ExpirationFromString(e)
	if err != nil {
		log.Warning("generateUserKeyHandler> '%s' -> %s\n", e, err)
		WriteError(w, r, err)
		return
	}

	key, err := worker.GenerateKey()
	if err != nil {
		log.Warning("generateUserKeyHandler> key generation failed: %s\n", err)
		WriteError(w, r, err)
		return
	}

	err = worker.InsertUserKey(db, c.User.ID, key, exp)
	if err != nil {
		log.Warning("generateUserKeyHandler> cannot insert new key: %s\n", err)
		WriteError(w, r, err)
		return
	}

	s := struct {
		Key string `json:"key"`
	}{
		Key: key,
	}
	WriteJSON(w, r, s, http.StatusOK)
}
*/

func addWorkerModel(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {

	if !c.User.Admin {
		WriteError(w, r, sdk.ErrForbidden)
		return
	}

	// Get body
	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Warning("addWorkerModel> cannot read body: %s\n", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Unmarshal body
	var model sdk.Model
	err = json.Unmarshal(data, &model)
	if err != nil {
		log.Warning("addWorkerModel> cannot unmarshal body data: %s\n", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if len(model.Name) == 0 {
		log.Warning("addWorkerModel> model name is empty: %s\n", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Insert model in db
	model.OwnerID = c.User.ID
	err = worker.InsertWorkerModel(db, &model)
	if err != nil {
		log.Warning("addWorkerModel> cannot add worker model: %s\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	WriteJSON(w, r, model, http.StatusOK)
}

func updateWorkerModel(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {
	vars := mux.Vars(r)
	idString := vars["id"]

	// Get body
	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Warning("updateWorkerModel> cannot read body: %s\n", err)
		WriteError(w, r, err)
		return
	}

	// Unmarshal body
	var model sdk.Model
	err = json.Unmarshal(data, &model)
	if err != nil {
		log.Warning("updateWorkerModel> cannot unmarshal body data: %s\n", err)
		WriteError(w, r, err)
		return
	}

	if model.Name == "" {
		log.Warning("updateWorkerModel> Name is empty\n")
		WriteError(w, r, sdk.ErrInvalidName)
		return
	}

	if model.Image == "" {
		log.Warning("updateWorkerModel> Image is empty\n")
		WriteError(w, r, sdk.ErrUnauthorized)
		return
	}

	modelID, err := strconv.ParseInt(idString, 10, 64)
	if err != nil {
		log.Warning("updateWorkerModel> modelID must be an integer : %s\n", err)
		WriteError(w, r, sdk.ErrInvalidID)
		return
	}

	if modelID != model.ID {
		log.Warning("updateWorkerModel> wrong ID.\n", err)
		WriteError(w, r, sdk.ErrInvalidID)
		return
	}

	// update model in db
	err = worker.UpdateWorkerModel(db, model)
	if err != nil {
		log.Warning("updateWorkerModel> cannot update worker model: %s\n", err)
		WriteError(w, r, err)
		return
	}

	// Recompute warnings
	go func() {
		warnings, err := sanity.LoadAllWarnings(db, "")
		if err != nil {
			log.Warning("updateWorkerModel> cannot load warnings: %s\n", err)
			return
		}

		for _, warning := range warnings {
			sanity.CheckPipeline(db, &warning.Project, &warning.Pipeline)
		}
	}()
}

func deleteWorkerModel(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {
	vars := mux.Vars(r)
	workerModelIDs := vars["id"]

	workerModelID, err := strconv.ParseInt(workerModelIDs, 10, 64)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	tx, err := db.Begin()
	if err != nil {
		WriteError(w, r, sdk.ErrUnknownError)
		return
	}

	err = worker.DeleteWorkerModel(tx, workerModelID)
	if err != nil {
		log.Warning("deleteWorkerModel: cannot delete worker model: %s\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	err = tx.Commit()
	if err != nil {
		WriteError(w, r, sdk.ErrUnknownError)
		return
	}
}

func getWorkerModel(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context, name string) {
	m, err := worker.LoadWorkerModel(db, name)
	if err != nil {
		if err != sdk.ErrNoWorkerModel {
			log.Warning("getWorkerModel> cannot load worker model: %s\n", err)
		}
		WriteError(w, r, err)
		return
	}

	WriteJSON(w, r, m, http.StatusOK)
}

func getWorkerModels(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {

	err := r.ParseForm()
	if err != nil {
		log.Warning("getWorkerModels> cannot parse form")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	name := r.FormValue("name")
	if name != "" {
		getWorkerModel(w, r, db, c, name)
		return
	}

	models, err := worker.LoadWorkerModels(db)
	if err != nil {
		log.Warning("getWorkerModels> cannot load worker models: %s\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	WriteJSON(w, r, models, http.StatusOK)
}

func addWorkerModelCapa(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {
	vars := mux.Vars(r)
	workerModelIDs := vars["id"]

	workerModelID, err := strconv.ParseInt(workerModelIDs, 10, 64)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Get body
	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Warning("addWorkerModelCapa> cannot read body: %s\n", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Unmarshal body
	var capa sdk.Requirement
	err = json.Unmarshal(data, &capa)
	if err != nil {
		log.Warning("addWorkerModelCapa> cannot unmarshal body data: %s\n", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	err = worker.InsertWorkerModelCapability(db, workerModelID, capa)
	if err != nil {
		log.Warning("addWorkerModelCapa: cannot insert new worker model capa: %s\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Recompute warnings
	go func() {
		warnings, err := sanity.LoadAllWarnings(db, "")
		if err != nil {
			log.Warning("updateWorkerModel> cannot load warnings: %s\n", err)
			return
		}

		for _, warning := range warnings {
			sanity.CheckPipeline(db, &warning.Project, &warning.Pipeline)
		}
	}()

}

func getWorkerModelTypes(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {
	WriteJSON(w, r, sdk.AvailableWorkerModelType, http.StatusOK)
}

func getWorkerModelCapaTypes(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {
	WriteJSON(w, r, sdk.AvailableRequirementsType, http.StatusOK)
}

func updateWorkerModelCapa(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {
	vars := mux.Vars(r)
	workerModelIDs := vars["id"]
	capaName := vars["capa"]

	workerModelID, err := strconv.ParseInt(workerModelIDs, 10, 64)
	if err != nil {
		log.Warning("updateWorkerModelCapa> id must be a integer: %s\n", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Get body
	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Warning("updateWorkerModelCapa> cannot read body: %s\n", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Unmarshal body
	var capa sdk.Requirement
	err = json.Unmarshal(data, &capa)
	if err != nil {
		log.Warning("updateWorkerModelCapa> cannot unmarshal body data: %s\n", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if capaName != capa.Name {
		log.Warning("updateWorkerModelCapa> Wrong capability name\n", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	err = worker.UpdateWorkerModelCapability(db, capa, workerModelID)
	if err != nil {
		if err == sdk.ErrNoWorkerModelCapa {
			log.Warning("updateWorkerModelCapa> cannot update worker model capa: %s\n", err)
		}
		log.Warning("updateWorkerModelCapa: cannot update capability: %s\n", err)
		WriteError(w, r, err)
		return
	}

	// Recompute warnings
	go func() {
		warnings, err := sanity.LoadAllWarnings(db, "")
		if err != nil {
			log.Warning("updateWorkerModel> cannot load warnings: %s\n", err)
			return
		}

		for _, warning := range warnings {
			sanity.CheckPipeline(db, &warning.Project, &warning.Pipeline)
		}
	}()

}

func deleteWorkerModelCapa(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {
	vars := mux.Vars(r)
	workerModelIDs := vars["id"]
	capaName := vars["capa"]

	workerModelID, err := strconv.ParseInt(workerModelIDs, 10, 64)
	if err != nil {
		log.Warning("deleteWorkerModelCapa> modelID is no integer '%s': %s\n", workerModelIDs, err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	err = worker.DeleteWorkerModelCapability(db, workerModelID, capaName)
	if err != nil {
		if err == sdk.ErrNoWorkerModelCapa {
			log.Warning("updateWorkerModelCapa> cannot remove worker model capa: %s\n", err)
		}
		log.Warning("deleteWorkerModelCapa: cannot remove capability: %s\n", err)
		WriteError(w, r, err)
		return
	}

	// Recompute warnings
	go func() {
		warnings, err := sanity.LoadAllWarnings(db, "")
		if err != nil {
			log.Warning("updateWorkerModel> cannot load warnings: %s\n", err)
			return
		}

		for _, warning := range warnings {
			sanity.CheckPipeline(db, &warning.Project, &warning.Pipeline)
		}
	}()
}

func getWorkerModelStatus(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {
	ms, err := worker.EstimateWorkerModelNeeds(db, c.User)
	if err != nil {
		log.Warning("getWorkerModelStatus> Cannot estimate worker model needs: %s\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	WriteJSON(w, r, ms, http.StatusOK)
}

func unregisterWorkerHandler(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {
	err := worker.DeleteWorker(db, c.WorkerID)
	if err != nil {
		log.Warning("unregisterWorkerHandler> cannot delete worker %s\n", err)
		WriteError(w, r, err)
		return
	}
}

func getWorkerModelsStatsHandler(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {
	res := []struct {
		Model string
		Used  int
	}{}

	cache.Get("stats:models", &res)
	if len(res) > 0 {
		WriteJSON(w, r, res, http.StatusOK)
		return
	}

	//This can be very long, so run it in a goroutine and send 202
	go func() {
		var loading string
		cache.Get("stats:models:loading", &loading)
		if loading != "" {
			return
		}
		loading = "true"
		cache.Set("stats:models:loading", loading)
		query := `
		select model, sum(used)
		from (
			select worker_model_name as model, count(action_build.id) as used from action_build group by worker_model_name
			union
			select m.model as model, count(1) as used
			from (
				select jsonb_array_elements(b.builds)->>'model' as model
				from
				(
					select stages->'builds' as builds
					from pipeline_history h, jsonb_array_elements(h.data->'stages') stages
					where jsonb_typeof(h.data->'stages') = 'array'
				) b
			) m
			group by m.model
			) m_u
		where model is not null
		group by model;
	`

		rows, err := db.Query(query)
		if err != nil {
			log.Warning("getWorkerModelsStatusHandler> %s", err)
			WriteError(w, r, err)
			return
		}
		defer rows.Close()

		for rows.Next() {
			var model string
			var used int

			if err := rows.Scan(&model, &used); err != nil {
				log.Warning("getWorkerModelsStatusHandler> %s", err)
				WriteError(w, r, err)
				return
			}
			res = append(res, struct {
				Model string
				Used  int
			}{model, used})
		}

		cache.Set("stats:models", res)
		cache.Delete("stats:models:loading")
	}()

	WriteJSON(w, r, res, http.StatusAccepted)
}
