package main

import (
	"net/http"
	"strconv"

	"github.com/go-gorp/gorp"
	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/context"
	"github.com/ovh/cds/engine/api/sanity"
	"github.com/ovh/cds/engine/api/worker"
	"github.com/ovh/cds/engine/log"
	"github.com/ovh/cds/sdk"
)

func addWorkerModel(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	// Unmarshal body
	var model sdk.Model
	if err := UnmarshalBody(r, &model); err != nil {
		return err
	}

	if model.Type == "" {
		return sdk.ErrWrongRequest
	}

	if len(model.Name) == 0 {
		log.Warning("addWorkerModel> model name is empty")
		return sdk.ErrWrongRequest
	}

	if model.GroupID == 0 {
		log.Warning("addWorkerModel> groupID should be set")
		return sdk.ErrWrongRequest
	}

	//User must be admin of the group set in the model
	var ok bool
	for _, g := range c.User.Groups {
		if g.ID == model.GroupID {
			for _, a := range g.Admins {
				if a.ID == c.User.ID {
					ok = true
				}
			}
		}
	}

	//User should have the right permission or be admin
	if !c.User.Admin && !ok {
		return sdk.ErrForbidden

	}

	model.CreatedBy = sdk.User{
		Email:    c.User.Email,
		Username: c.User.Username,
		Admin:    c.User.Admin,
		Fullname: c.User.Fullname,
		ID:       c.User.ID,
		Origin:   c.User.Origin,
	}

	// Insert model in db
	if err := worker.InsertWorkerModel(db, &model); err != nil {
		log.Warning("addWorkerModel> cannot add worker model: %s\n", err)
		return err

	}

	return WriteJSON(w, r, model, http.StatusOK)
}

func updateWorkerModel(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	vars := mux.Vars(r)
	idString := vars["permModelID"]

	modelID, errParse := strconv.ParseInt(idString, 10, 64)
	if errParse != nil {
		log.Warning("updateWorkerModel> modelID must be an integer : %s\n", errParse)
		return sdk.ErrInvalidID

	}

	old, errLoad := worker.LoadWorkerModelByID(db, modelID)
	if errLoad != nil {
		return errLoad
	}

	// Unmarshal body
	var model sdk.Model
	if err := UnmarshalBody(r, &model); err != nil {
		return err
	}

	//If the model name has not been set, keep the old name
	if model.Name == "" {
		model.Name = old.Name
	}

	//If the model image has not been set, keep the old image
	if model.Image == "" {
		model.Image = old.Image
	}

	//If the model Capabilities has not been set, keep the old Capabilities
	if len(model.Capabilities) == 0 {
		model.Capabilities = old.Capabilities
	}

	//If the model GroupID has not been set, keep the old GroupID
	if model.GroupID == 0 {
		model.GroupID = old.GroupID
	}

	//If the model Type has not been set, keep the old Type
	if model.Type == "" {
		model.Type = old.Type
	}

	//If the model modelID has not been set, keep the old modelID
	if model.ID == 0 {
		model.ID = old.ID
	}

	//User must be admin of the group set in the new model
	var ok bool
	for _, g := range c.User.Groups {
		if g.ID == model.GroupID {
			for _, a := range g.Admins {
				if a.ID == c.User.ID {
					ok = true
				}
			}
		}
	}

	//User should have the right permission or be admin
	if !c.User.Admin && !ok {
		return sdk.ErrForbidden

	}

	if modelID != model.ID {
		log.Warning("updateWorkerModel> wrong ID.\n")
		return sdk.ErrInvalidID

	}

	// update model in db
	if err := worker.UpdateWorkerModel(db, model); err != nil {
		log.Warning("updateWorkerModel> cannot update worker model: %s\n", err)
		return err

	}

	// Recompute warnings
	go func() {
		warnings, err := sanity.LoadAllWarnings(db, "")
		if err != nil {
			log.Warning("updateWorkerModel> cannot load warnings: %s\n", err)

		}

		for _, warning := range warnings {
			sanity.CheckPipeline(db, &warning.Project, &warning.Pipeline)
		}
	}()

	return WriteJSON(w, r, model, http.StatusOK)
}

func deleteWorkerModel(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	vars := mux.Vars(r)
	workerModelIDs := vars["permModelID"]

	workerModelID, err := strconv.ParseInt(workerModelIDs, 10, 64)
	if err != nil {
		return sdk.ErrWrongRequest

	}

	tx, err := db.Begin()
	if err != nil {
		return err
	}

	if err := worker.DeleteWorkerModel(tx, workerModelID); err != nil {
		log.Warning("deleteWorkerModel: cannot delete worker model: %s\n", err)
		return err

	}

	if err := tx.Commit(); err != nil {
		return err
	}

	return nil
}

func getWorkerModel(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx, name string) error {
	m, err := worker.LoadWorkerModelByName(db, name)
	if err != nil {
		if err != sdk.ErrNoWorkerModel {
			log.Warning("getWorkerModel> cannot load worker model: %s\n", err)
		}
		return err
	}

	return WriteJSON(w, r, m, http.StatusOK)
}

func getWorkerModels(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	if err := r.ParseForm(); err != nil {
		log.Warning("getWorkerModels> cannot parse form")
		return sdk.ErrWrongRequest
	}

	name := r.FormValue("name")
	if name != "" {
		return getWorkerModel(w, r, db, c, name)
	}

	models, err := worker.LoadWorkerModelsByUser(db, c.User.ID)
	if err != nil {
		log.Warning("getWorkerModels> cannot load worker models: %s\n", err)
		return err
	}

	log.Debug("getWorkerModels> %s", models)

	return WriteJSON(w, r, models, http.StatusOK)
}

func addWorkerModelCapa(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	vars := mux.Vars(r)
	workerModelIDs := vars["permModelID"]

	workerModelID, errParse := strconv.ParseInt(workerModelIDs, 10, 64)
	if errParse != nil {
		return sdk.ErrWrongRequest

	}

	workerModel, errLoad := worker.LoadWorkerModelByID(db, workerModelID)
	if errLoad != nil {
		return errLoad
	}

	var capa sdk.Requirement
	if err := UnmarshalBody(r, &capa); err != nil {
		return err
	}
	workerModel.Capabilities = append(workerModel.Capabilities, capa)

	if err := worker.UpdateWorkerModel(db, *workerModel); err != nil {
		log.Warning("addWorkerModelCapa> cannot insert new worker model capa: %s\n", err)
		return err
	}

	// Recompute warnings
	go func() {
		warnings, err := sanity.LoadAllWarnings(db, "")
		if err != nil {
			log.Warning("updateWorkerModel> cannot load warnings: %s\n", err)

		}

		for _, warning := range warnings {
			sanity.CheckPipeline(db, &warning.Project, &warning.Pipeline)
		}
	}()

	return nil
}

func getWorkerModelTypes(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	return WriteJSON(w, r, sdk.AvailableWorkerModelType, http.StatusOK)
}

func getWorkerModelCapaTypes(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	return WriteJSON(w, r, sdk.AvailableRequirementsType, http.StatusOK)
}

func updateWorkerModelCapa(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	vars := mux.Vars(r)
	workerModelIDs := vars["permModelID"]
	capaName := vars["capa"]

	workerModelID, err := strconv.ParseInt(workerModelIDs, 10, 64)
	if err != nil {
		log.Warning("updateWorkerModelCapa> id must be a integer: %s\n", err)
		return sdk.ErrWrongRequest

	}
	// Unmarshal body
	var capa sdk.Requirement
	if err := UnmarshalBody(r, &capa); err != nil {
		return err
	}

	if capaName != capa.Name {
		log.Warning("updateWorkerModelCapa> Wrong capability name\n", err)
		return sdk.ErrWrongRequest

	}

	err = worker.UpdateWorkerModelCapability(db, capa, workerModelID)
	if err != nil {
		if err == sdk.ErrNoWorkerModelCapa {
			log.Warning("updateWorkerModelCapa> cannot update worker model capa: %s\n", err)
		}
		log.Warning("updateWorkerModelCapa: cannot update capability: %s\n", err)
		return err

	}

	// Recompute warnings
	go func() {
		warnings, err := sanity.LoadAllWarnings(db, "")
		if err != nil {
			log.Warning("updateWorkerModel> cannot load warnings: %s\n", err)

		}

		for _, warning := range warnings {
			sanity.CheckPipeline(db, &warning.Project, &warning.Pipeline)
		}
	}()

	return nil
}

func deleteWorkerModelCapa(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	vars := mux.Vars(r)
	workerModelIDs := vars["permModelID"]
	capaName := vars["capa"]

	workerModelID, err := strconv.ParseInt(workerModelIDs, 10, 64)
	if err != nil {
		log.Warning("deleteWorkerModelCapa> modelID is no integer '%s': %s\n", workerModelIDs, err)
		return sdk.ErrWrongRequest
	}

	if err := worker.DeleteWorkerModelCapability(db, workerModelID, capaName); err != nil {
		if err == sdk.ErrNoWorkerModelCapa {
			log.Warning("updateWorkerModelCapa> cannot remove worker model capa: %s\n", err)
		}
		log.Warning("deleteWorkerModelCapa: cannot remove capability: %s\n", err)
		return err

	}

	// Recompute warnings
	go func() {
		warnings, err := sanity.LoadAllWarnings(db, "")
		if err != nil {
			log.Warning("updateWorkerModel> cannot load warnings: %s\n", err)

		}

		for _, warning := range warnings {
			sanity.CheckPipeline(db, &warning.Project, &warning.Pipeline)
		}
	}()

	return nil
}

func getWorkerModelStatus(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	if c.Agent == sdk.HatcheryAgent {
		ms, err := worker.EstimateWorkerModelNeeds(db, c.User.Groups[0].ID, worker.LoadWorkerModelStatusForGroup, worker.LoadGroupActionCount)
		if err != nil {
			log.Warning("getWorkerModelStatus> Cannot estimate worker model needs: %s\n", err)
			return err
		}
		return WriteJSON(w, r, ms, http.StatusOK)
	}

	if c.User.Admin == true {
		ms, err := worker.EstimateWorkerModelNeeds(db, c.User.ID, worker.LoadWorkerModelStatusForAdminUser, worker.LoadAllActionCount)
		if err != nil {
			log.Warning("getWorkerModelStatus> Cannot estimate worker model needs: %s\n", err)
			return err
		}
		return WriteJSON(w, r, ms, http.StatusOK)
	}

	return sdk.ErrForbidden
}

func getWorkerModelsStatsHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	res := []struct {
		Model string
		Used  int
	}{}

	cache.Get("stats:models", &res)
	if len(res) > 0 {
		return WriteJSON(w, r, res, http.StatusOK)

	}

	//This can be very long, so run it in a goroutine and send 202
	go func() {
		var loading string
		cache.Get("stats:models:loading", &loading)
		if loading != "" {

		}
		loading = "true"
		cache.Set("stats:models:loading", loading)
		query := `
		select model, sum(used)
		from (
			select worker_model_name as model, count(pipeline_build_job.id) as used from pipeline_build_job group by worker_model_name
			union
			select m.model as model, count(1) as used
			from (
				select jsonb_array_elements(b.builds)->>'model' as model
				from
				(
					select stages->'builds' as builds
					from pipeline_build h, jsonb_array_elements(h.stages) stages
					where jsonb_typeof(h.stages) = 'array'
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
			return
		}
		defer rows.Close()

		for rows.Next() {
			var model string
			var used int

			if err := rows.Scan(&model, &used); err != nil {
				log.Warning("getWorkerModelsStatusHandler> %s", err)
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

	return WriteJSON(w, r, res, http.StatusAccepted)
}

func getWorkerModelInstances(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	vars := mux.Vars(r)
	idString := vars["permModelID"]

	modelID, errParse := strconv.ParseInt(idString, 10, 64)
	if errParse != nil {
		log.Warning("getWorkerModelInstances> modelID must be an integer : %s\n", errParse)
		return sdk.ErrInvalidID
	}

	m, errLoad := worker.LoadWorkerModelByID(db, modelID)
	if errLoad != nil {
		return errLoad
	}

	ws, errW := worker.LoadWorkersByModel(db, m.ID)
	if errW != nil {
		return errW
	}

	return WriteJSON(w, r, ws, http.StatusOK)
}
