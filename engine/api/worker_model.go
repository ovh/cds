package main

import (
	"net/http"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/action"
	"github.com/ovh/cds/engine/api/businesscontext"
	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/sanity"
	"github.com/ovh/cds/engine/api/worker"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func addWorkerModel(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *businesscontext.Ctx) error {
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

func spawnErrorWorkerModelHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *businesscontext.Ctx) error {
	spawnErrorForm := &sdk.SpawnErrorForm{}
	if err := UnmarshalBody(r, spawnErrorForm); err != nil {
		return sdk.WrapError(err, "spawnErrorWorkerModelHandler> Unable to parse spawn error form")
	}

	workerModelID, errr := requestVarInt(r, "permModelID")
	if errr != nil {
		return sdk.WrapError(errr, "updateWorkerModel> Invalid permModelID")
	}

	tx, errBegin := db.Begin()
	if errBegin != nil {
		return sdk.WrapError(errBegin, "spawnErrorWorkerModelHandler> Cannot start transaction")
	}
	defer tx.Rollback()

	model, errLoad := worker.LoadWorkerModelByID(db, workerModelID)
	if errLoad != nil {
		return sdk.WrapError(errLoad, "spawnErrorWorkerModelHandler> cannot load worker model by id")
	}

	if err := worker.UpdateSpawnErrorWorkerModel(tx, model.ID, spawnErrorForm.Error); err != nil {
		return sdk.WrapError(err, "spawnErrorWorkerModelHandler> cannot update spawn error on worker model")
	}

	if err := tx.Commit(); err != nil {
		return sdk.WrapError(err, "spawnErrorWorkerModelHandler> Cannot commit tx")
	}

	return WriteJSON(w, r, nil, http.StatusOK)
}

func updateWorkerModel(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *businesscontext.Ctx) error {
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

	//If the model has been renamed, we will have to update requirements
	var renamed bool
	if model.Name != old.Name {
		renamed = true
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

	tx, errtx := db.Begin()
	if errtx != nil {
		return sdk.WrapError(errtx, "updateWorkerModel> unable to start transaction")
	}

	defer tx.Rollback()

	// update model in db
	if err := worker.UpdateWorkerModel(tx, model); err != nil {
		return sdk.WrapError(err, "updateWorkerModel> cannot update worker model")
	}

	// update requirements if needed
	if renamed {
		actionsID, erru := action.UpdateAllRequirements(tx, old.Name, model.Name, sdk.ModelRequirement)
		if erru != nil {
			return sdk.WrapError(erru, "updateWorkerModel> cannot update action requirements")
		}

		log.Debug("updateWorkerModel> Update action %v", actionsID)

		//update all the pipelines using this action
		actions, erra := action.LoadJoinedActionsByActionID(tx, actionsID)
		if erra != nil {
			return sdk.WrapError(erra, "updateWorkerModel> cannot load joined actions")
		}

		log.Debug("updateWorkerModel> Loaded action %v", actions)

		for _, a := range actions {
			log.Debug("updateWorkerModel> Loading pipeline for action %d", a.ID)
			id, err := pipeline.GetPipelineIDFromJoinedActionID(tx, a.ID)
			if err != nil {
				return sdk.WrapError(err, "updateWorkerModel> cannot get pipeline")
			}
			log.Debug("updateWorkerModel> Updating pipeline %d", id)
			//Load the project
			proj, errproj := project.LoadByPipelineID(tx, c.User, id)
			if errproj != nil {
				return sdk.WrapError(errproj, "updateWorkerModel> unable to load project")
			}

			if err := pipeline.UpdatePipelineLastModified(tx, proj, &sdk.Pipeline{ID: id}, c.User); err != nil {
				return sdk.WrapError(err, "updateWorkerModel> cannot update pipeline")
			}
		}

	}

	if err := tx.Commit(); err != nil {
		return sdk.WrapError(err, "updateWorkerModel> unable to commit transaction")
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

func deleteWorkerModel(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *businesscontext.Ctx) error {
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

func getWorkerModel(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *businesscontext.Ctx, name string) error {
	m, err := worker.LoadWorkerModelByName(db, name)
	if err != nil {
		return sdk.WrapError(err, "getWorkerModel> cannot load worker model")
	}
	return WriteJSON(w, r, m, http.StatusOK)
}

func getWorkerModelsEnabled(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *businesscontext.Ctx) error {
	if c.Hatchery == nil || c.Hatchery.GroupID == 0 {
		return sdk.WrapError(sdk.ErrWrongRequest, "getWorkerModelsEnabled> this route can be called only by hatchery")
	}
	models, errgroup := worker.LoadWorkerModelsUsableOnGroup(db, c.Hatchery.GroupID, group.SharedInfraGroup.ID)
	if errgroup != nil {
		return sdk.WrapError(errgroup, "getWorkerModels> cannot load worker models for hatchery %d with group %d", c.Hatchery.ID, c.Hatchery.GroupID)
	}
	return WriteJSON(w, r, models, http.StatusOK)
}

func getWorkerModels(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *businesscontext.Ctx) error {
	if err := r.ParseForm(); err != nil {
		return sdk.WrapError(sdk.ErrWrongRequest, "getWorkerModels> cannot parse form")
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
	} else {
		return sdk.WrapError(sdk.ErrWrongRequest, "getWorkerModels> this route can't be called by worker or hatchery")
	}

	return WriteJSON(w, r, models, http.StatusOK)
}

func getWorkerModelTypes(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *businesscontext.Ctx) error {
	return WriteJSON(w, r, sdk.AvailableWorkerModelType, http.StatusOK)
}

func getWorkerModelCommunications(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *businesscontext.Ctx) error {
	return WriteJSON(w, r, sdk.AvailableWorkerModelCommunication, http.StatusOK)
}

func getWorkerModelCapaTypes(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *businesscontext.Ctx) error {
	return WriteJSON(w, r, sdk.AvailableRequirementsType, http.StatusOK)
}

func getWorkerModelsStatsHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *businesscontext.Ctx) error {
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
