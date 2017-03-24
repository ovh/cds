package main

import (
	"net/http"

	"github.com/go-gorp/gorp"
	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/context"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/sanity"
	"github.com/ovh/cds/engine/api/worker"
	"github.com/ovh/cds/engine/log"
	"github.com/ovh/cds/sdk"
)

func addWorkerModel(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	// Unmarshal body
	var model sdk.Model
	if err := UnmarshalBody(r, &model); err != nil {
		return sdk.WrapError(err, "addWorkerModel> cannot unmarshal body")
	}

	if model.Type == "" {
		return sdk.WrapError(sdk.ErrWrongRequest, "addWorkerModel> Invalid type (empty)")
	}

	if len(model.Name) == 0 {
		return sdk.WrapError(sdk.ErrWrongRequest, "addWorkerModel> model name is empty")
	}

	if model.GroupID == 0 {
		return sdk.WrapError(sdk.ErrWrongRequest, "addWorkerModel> groupID should be set")
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
		return sdk.WrapError(err, "addWorkerModel> cannot add worker model")
	}

	return WriteJSON(w, r, model, http.StatusOK)
}

func updateWorkerModel(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	workerModelID, errr := requestVarInt(r, "permModelID")
	if errr != nil {
		return sdk.WrapError(errr, "updateWorkerModel> Invalid permModelID")
	}

	old, errLoad := worker.LoadWorkerModelByID(db, workerModelID)
	if errLoad != nil {
		return sdk.WrapError(errLoad, "updateWorkerModel> cannot load worker model by id")
	}

	// Unmarshal body
	var model sdk.Model
	if err := UnmarshalBody(r, &model); err != nil {
		return sdk.WrapError(err, "updateWorkerModel> cannot unmarshal body")
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

	if workerModelID != model.ID {
		return sdk.WrapError(sdk.ErrInvalidID, "updateWorkerModel> wrong ID")
	}

	// update model in db
	if err := worker.UpdateWorkerModel(db, model); err != nil {
		return sdk.WrapError(err, "updateWorkerModel> cannot update worker model")
	}

	// Recompute warnings
	go func() {
		warnings, err := sanity.LoadAllWarnings(db, "")
		if err != nil {
			log.Warning("updateWorkerModel> cannot load warnings: %s", err)
		}

		for _, warning := range warnings {
			sanity.CheckPipeline(db, &warning.Project, &warning.Pipeline)
		}
	}()

	return WriteJSON(w, r, model, http.StatusOK)
}

func deleteWorkerModel(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	workerModelID, errr := requestVarInt(r, "permModelID")
	if errr != nil {
		return sdk.WrapError(errr, "deleteWorkerModel> Invalid permModelID")
	}

	tx, err := db.Begin()
	if err != nil {
		return sdk.WrapError(err, "deleteWorkerModel> Cannot start transaction")
	}

	if err := worker.DeleteWorkerModel(tx, workerModelID); err != nil {
		return sdk.WrapError(err, "deleteWorkerModel: cannot delete worker model")
	}

	if err := tx.Commit(); err != nil {
		return sdk.WrapError(err, "deleteWorkerModel> Cannot commit transaction")
	}

	return nil
}

func getWorkerModel(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx, name string) error {
	m, err := worker.LoadWorkerModelByName(db, name)
	if err != nil {
		return sdk.WrapError(err, "getWorkerModel> cannot load worker model")
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

	models := []sdk.Model{}
	if c.User != nil && c.User.ID > 0 {
		var errbyuser error
		models, errbyuser = worker.LoadWorkerModelsByUser(db, c.User)
		if errbyuser != nil {
			return sdk.WrapError(errbyuser, "getWorkerModels> cannot load worker models for user id %d", c.User.ID)
		}
		log.Debug("getWorkerModels> for user %d named %s (admin:%t): %s", c.User.ID, c.User.Username, c.User.Admin, models)
	} else if c.Hatchery != nil && c.Hatchery.GroupID > 0 {
		var errgroup error
		models, errgroup = worker.LoadWorkerModelsUsableOnGroup(db, c.Hatchery.GroupID, group.SharedInfraGroup.ID)
		if errgroup != nil {
			return sdk.WrapError(errgroup, "getWorkerModels> cannot load worker models for hatchery %d with group %d", c.Hatchery.ID, c.Hatchery.GroupID)
		}
		log.Debug("getWorkerModels> for hatchery %s with group %s : %s", c.Hatchery.ID, c.Hatchery.GroupID, models)
	}

	return WriteJSON(w, r, models, http.StatusOK)
}

func addWorkerModelCapa(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	workerModelID, errr := requestVarInt(r, "permModelID")
	if errr != nil {
		return sdk.WrapError(errr, "addWorkerModelCapa> Invalid permModelID")
	}

	workerModel, errLoad := worker.LoadWorkerModelByID(db, workerModelID)
	if errLoad != nil {
		return sdk.WrapError(errLoad, "addWorkerModelCapa> cannot load worker model by id")
	}

	var capa sdk.Requirement
	if err := UnmarshalBody(r, &capa); err != nil {
		return sdk.WrapError(err, "addWorkerModelCapa> cannot unmashal body")
	}
	workerModel.Capabilities = append(workerModel.Capabilities, capa)

	if err := worker.UpdateWorkerModel(db, *workerModel); err != nil {
		return sdk.WrapError(err, "addWorkerModelCapa> cannot insert new worker model capa")
	}

	// Recompute warnings
	go func() {
		warnings, err := sanity.LoadAllWarnings(db, "")
		if err != nil {
			log.Warning("updateWorkerModel> cannot load warnings: %s", err)
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
	capaName := vars["capa"]

	workerModelID, errr := requestVarInt(r, "permModelID")
	if errr != nil {
		return sdk.WrapError(errr, "updateWorkerModelCapa> Invalid permModelID")
	}

	// Unmarshal body
	var capa sdk.Requirement
	if err := UnmarshalBody(r, &capa); err != nil {
		return sdk.WrapError(err, "updateWorkerModelCapa> Cannot unmarshal body")
	}

	if capaName != capa.Name {
		return sdk.WrapError(sdk.ErrWrongRequest, "updateWorkerModelCapa> Wrong capability name %s != %s", capaName, capa.Name)
	}

	if err := worker.UpdateWorkerModelCapability(db, capa, workerModelID); err != nil {
		return sdk.WrapError(err, "updateWorkerModelCapa> cannot update worker model")
	}

	// Recompute warnings
	go func() {
		warnings, err := sanity.LoadAllWarnings(db, "")
		if err != nil {
			log.Warning("updateWorkerModel> cannot load warnings: %s", err)
		}

		for _, warning := range warnings {
			sanity.CheckPipeline(db, &warning.Project, &warning.Pipeline)
		}
	}()

	return nil
}

func deleteWorkerModelCapa(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	vars := mux.Vars(r)
	capaName := vars["capa"]

	workerModelID, errr := requestVarInt(r, "permModelID")
	if errr != nil {
		return sdk.WrapError(errr, "deleteWorkerModelCapa> Invalid permModelID")
	}

	if err := worker.DeleteWorkerModelCapability(db, workerModelID, capaName); err != nil {
		return sdk.WrapError(err, "updateWorkerModelCapa> cannot remove worker model capa")
	}

	// Recompute warnings
	go func() {
		warnings, err := sanity.LoadAllWarnings(db, "")
		if err != nil {
			log.Warning("updateWorkerModel> cannot load warnings: %s", err)
		}

		for _, warning := range warnings {
			sanity.CheckPipeline(db, &warning.Project, &warning.Pipeline)
		}
	}()

	return nil
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
	workerModelID, errr := requestVarInt(r, "permModelID")
	if errr != nil {
		return sdk.WrapError(errr, "getWorkerModelInstances> Invalid permModelID")
	}

	m, errLoad := worker.LoadWorkerModelByID(db, workerModelID)
	if errLoad != nil {
		return sdk.WrapError(errLoad, "getWorkerModelInstances> cannot load worker model")
	}

	ws, errW := worker.LoadWorkersByModel(db, m.ID)
	if errW != nil {
		return sdk.WrapError(errW, "getWorkerModelInstances> cannot load workers by model id")
	}

	return WriteJSON(w, r, ws, http.StatusOK)
}
