package api

import (
	"context"
	"net/http"

	"github.com/ovh/cds/engine/api/action"

	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/sanity"
	"github.com/ovh/cds/engine/api/worker"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func (api *API) addWorkerModelHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
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

		// check if worker model already exists
		if _, err := worker.LoadWorkerModelByName(api.mustDB(), model.Name); err == nil {
			return sdk.WrapError(sdk.ErrModelNameExist, "getWorkerModel> worker model already exists")
		}

		//User must be admin of the group set in the model
		var ok bool
		for _, g := range getUser(ctx).Groups {
			if g.ID == model.GroupID {
				for _, a := range g.Admins {
					if a.ID == getUser(ctx).ID {
						ok = true
					}
				}
			}
		}

		//User should have the right permission or be admin
		if !getUser(ctx).Admin && !ok {
			return sdk.ErrForbidden

		}

		model.CreatedBy = sdk.User{
			Email:    getUser(ctx).Email,
			Username: getUser(ctx).Username,
			Admin:    getUser(ctx).Admin,
			Fullname: getUser(ctx).Fullname,
			ID:       getUser(ctx).ID,
			Origin:   getUser(ctx).Origin,
		}

		// Insert model in db
		if err := worker.InsertWorkerModel(api.mustDB(), &model); err != nil {
			return sdk.WrapError(err, "addWorkerModel> cannot add worker model")
		}

		return WriteJSON(w, r, model, http.StatusOK)
	}
}

func (api *API) spawnErrorWorkerModelHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		spawnErrorForm := &sdk.SpawnErrorForm{}
		if err := UnmarshalBody(r, spawnErrorForm); err != nil {
			return sdk.WrapError(err, "spawnErrorWorkerModelHandler> Unable to parse spawn error form")
		}

		workerModelID, errr := requestVarInt(r, "permModelID")
		if errr != nil {
			return sdk.WrapError(errr, "updateWorkerModel> Invalid permModelID")
		}

		tx, errBegin := api.mustDB().Begin()
		if errBegin != nil {
			return sdk.WrapError(errBegin, "spawnErrorWorkerModelHandler> Cannot start transaction")
		}
		defer tx.Rollback()

		model, errLoad := worker.LoadWorkerModelByID(api.mustDB(), workerModelID)
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
}

func (api *API) updateWorkerModelHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		workerModelID, errr := requestVarInt(r, "permModelID")
		if errr != nil {
			return sdk.WrapError(errr, "updateWorkerModel> Invalid permModelID")
		}

		old, errLoad := worker.LoadWorkerModelByID(api.mustDB(), workerModelID)
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
			// check if worker model already exists
			if _, err := worker.LoadWorkerModelByName(api.mustDB(), model.Name); err == nil {
				return sdk.WrapError(sdk.ErrModelNameExist, "getWorkerModel> worker model already exists")
			}
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
		for _, g := range getUser(ctx).Groups {
			if g.ID == model.GroupID {
				for _, a := range g.Admins {
					if a.ID == getUser(ctx).ID {
						ok = true
					}
				}
			}
		}

		//User should have the right permission or be admin
		if !getUser(ctx).Admin && !ok {
			return sdk.ErrForbidden
		}

		if workerModelID != model.ID {
			return sdk.WrapError(sdk.ErrInvalidID, "updateWorkerModel> wrong ID")
		}

		tx, errtx := api.mustDB().Begin()
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
				proj, errproj := project.LoadByPipelineID(tx, api.Cache, getUser(ctx), id)
				if errproj != nil {
					return sdk.WrapError(errproj, "updateWorkerModel> unable to load project")
				}

				if err := pipeline.UpdatePipelineLastModified(tx, proj, &sdk.Pipeline{ID: id}, getUser(ctx)); err != nil {
					return sdk.WrapError(err, "updateWorkerModel> cannot update pipeline")
				}
			}
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "updateWorkerModel> unable to commit transaction")
		}

		// Recompute warnings
		go func() {
			warnings, err := sanity.LoadAllWarnings(api.mustDB(), "")
			if err != nil {
				log.Warning("updateWorkerModel> cannot load warnings: %s", err)
			}

			for _, warning := range warnings {
				sanity.CheckPipeline(api.mustDB(), api.Cache, &warning.Project, &warning.Pipeline)
			}
		}()

		return WriteJSON(w, r, model, http.StatusOK)
	}
}

func (api *API) deleteWorkerModelHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		workerModelID, errr := requestVarInt(r, "permModelID")
		if errr != nil {
			return sdk.WrapError(errr, "deleteWorkerModel> Invalid permModelID")
		}

		tx, err := api.mustDB().Begin()
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
}

func (api *API) getWorkerModel(w http.ResponseWriter, r *http.Request, name string) error {
	m, err := worker.LoadWorkerModelByName(api.mustDB(), name)
	if err != nil {
		return sdk.WrapError(err, "getWorkerModel> cannot load worker model")
	}
	return WriteJSON(w, r, m, http.StatusOK)
}

func (api *API) getWorkerModelsEnabledHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		if getHatchery(ctx) == nil || getHatchery(ctx).GroupID == 0 {
			return sdk.WrapError(sdk.ErrWrongRequest, "getWorkerModelsEnabled> this route can be called only by hatchery")
		}
		models, errgroup := worker.LoadWorkerModelsUsableOnGroup(api.mustDB(), getHatchery(ctx).GroupID, group.SharedInfraGroup.ID)
		if errgroup != nil {
			return sdk.WrapError(errgroup, "getWorkerModels> cannot load worker models for hatchery %d with group %d", getHatchery(ctx).ID, getHatchery(ctx).GroupID)
		}
		return WriteJSON(w, r, models, http.StatusOK)
	}
}

func (api *API) getWorkerModelsHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		if err := r.ParseForm(); err != nil {
			return sdk.WrapError(sdk.ErrWrongRequest, "getWorkerModels> cannot parse form")
		}

		name := r.FormValue("name")
		if name != "" {
			return api.getWorkerModel(w, r, name)
		}

		models := []sdk.Model{}
		if getUser(ctx) != nil && getUser(ctx).ID > 0 {
			var errbyuser error
			models, errbyuser = worker.LoadWorkerModelsByUser(api.mustDB(), getUser(ctx))
			if errbyuser != nil {
				return sdk.WrapError(errbyuser, "getWorkerModels> cannot load worker models for user id %d", getUser(ctx).ID)
			}
			log.Debug("getWorkerModels> for user %d named %s (admin:%t): %s", getUser(ctx).ID, getUser(ctx).Username, getUser(ctx).Admin, models)
		} else {
			return sdk.WrapError(sdk.ErrWrongRequest, "getWorkerModels> this route can't be called by worker or hatchery")
		}

		return WriteJSON(w, r, models, http.StatusOK)
	}
}

func (api *API) getWorkerModelTypesHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		return WriteJSON(w, r, sdk.AvailableWorkerModelType, http.StatusOK)
	}
}

func (api *API) getWorkerModelCommunicationsHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		return WriteJSON(w, r, sdk.AvailableWorkerModelCommunication, http.StatusOK)
	}
}

func (api *API) getWorkerModelCapaTypesHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		return WriteJSON(w, r, sdk.AvailableRequirementsType, http.StatusOK)
	}
}
