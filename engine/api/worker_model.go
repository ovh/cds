package api

import (
	"context"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/action"
	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/worker"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func (api *API) postWorkerModelHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		// parse request and check data validity
		var data sdk.Model
		if err := service.UnmarshalBody(r, &data); err != nil {
			return err
		}
		if err := data.IsValid(); err != nil {
			return err
		}

		// the default group cannot own worker model
		if group.IsDefaultGroupID(data.GroupID) {
			return sdk.WrapError(sdk.ErrWrongRequest, "this group can't be owner of a worker model")
		}

		// check that the group exists and user is admin for group id
		grp, err := group.LoadGroupByID(api.mustDB(), data.GroupID)
		if err != nil {
			return err
		}
		u := deprecatedGetUser(ctx)
		if err := group.CheckUserIsGroupAdmin(grp, u); err != nil {
			return err
		}

		// check if worker model already exists
		if _, err := worker.LoadWorkerModelByNameAndGroupID(api.mustDB(), data.Name, grp.ID); err == nil {
			return sdk.NewErrorFrom(sdk.ErrModelNameExist, "worker model already exists with name %s for group %s", data.Name, grp.Name)
		}

		// provision is allowed only for CDS Admin or by user with a restricted model
		if !u.Admin && !data.Restricted {
			data.Provision = 0
		}

		// if current user is not admin and model is not restricted, a pattern should be given
		if !u.Admin && !data.Restricted && data.PatternName == "" {
			return sdk.NewErrorFrom(sdk.ErrWorkerModelNoPattern, "missing model pattern name")
		}

		// if a model pattern is given try to get it from database
		if data.PatternName != "" {
			modelPattern, err := worker.LoadWorkerModelPatternByName(api.mustDB(), data.Type, data.PatternName)
			if err != nil {
				return sdk.NewErrorWithStack(err, sdk.NewErrorFrom(sdk.ErrWrongRequest, "invalid given worker model name"))
			}

			// set pattern data on given model
			switch data.Type {
			case sdk.Docker:
				data.ModelDocker.Cmd = modelPattern.Model.Cmd
				data.ModelDocker.Shell = modelPattern.Model.Shell
				data.ModelDocker.Envs = modelPattern.Model.Envs
			default:
				data.ModelVirtualMachine.PreCmd = modelPattern.Model.PreCmd
				data.ModelVirtualMachine.Cmd = modelPattern.Model.Cmd
				data.ModelVirtualMachine.PostCmd = modelPattern.Model.PostCmd
			}
		}

		// init new model from given data
		var model sdk.Model
		model.Update(data)

		model.CreatedBy = sdk.User{
			Email:    u.Email,
			Username: u.Username,
			Admin:    u.Admin,
			Fullname: u.Fullname,
			ID:       u.ID,
			Origin:   u.Origin,
		}

		if err := worker.InsertWorkerModel(api.mustDB(), &model); err != nil {
			return sdk.WrapError(err, "cannot add worker model")
		}

		key := cache.Key("api:workermodels:*")
		api.Cache.DeleteAll(key)

		// reload complete worker model
		new, err := worker.LoadWorkerModelByID(api.mustDB(), model.ID)
		if err != nil {
			return err
		}

		new.Editable = true

		return service.WriteJSON(w, new, http.StatusOK)
	}
}

func (api *API) putWorkerModelHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		u := deprecatedGetUser(ctx)

		vars := mux.Vars(r)

		groupName := vars["groupName"]
		modelName := vars["permModelName"]

		g, err := group.LoadGroup(api.mustDB(), groupName)
		if err != nil {
			return err
		}

		old, errLoad := worker.LoadWorkerModelByNameAndGroupID(api.mustDB(), modelName, g.ID)
		if errLoad != nil {
			return sdk.WrapError(errLoad, "cannot load worker model")
		}

		// parse request and validate given data
		var data sdk.Model
		if err := service.UnmarshalBody(r, &data); err != nil {
			return err
		}

		// if current user is not admin and model is not restricted and a pattern is not given, reuse old model info
		if !u.Admin && !data.Restricted && data.PatternName == "" {
			if old.Type != data.Type {
				return sdk.WrapError(sdk.ErrWorkerModelNoPattern, "we can't fetch previous user data because type or restricted is different")
			}
			// set pattern data on given model
			switch data.Type {
			case sdk.Docker:
				data.ModelDocker.Cmd = old.ModelDocker.Cmd
				data.ModelDocker.Shell = old.ModelDocker.Shell
				data.ModelDocker.Envs = old.ModelDocker.Envs
			default:
				data.ModelVirtualMachine.PreCmd = old.ModelVirtualMachine.PreCmd
				data.ModelVirtualMachine.Cmd = old.ModelVirtualMachine.Cmd
				data.ModelVirtualMachine.PostCmd = old.ModelVirtualMachine.PostCmd
			}
		}

		if err := data.IsValid(); err != nil {
			return err
		}

		// the default group cannot own worker model
		if group.IsDefaultGroupID(data.GroupID) {
			return sdk.WrapError(sdk.ErrWrongRequest, "this group can't be owner of a worker model")
		}

		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WrapError(err, "cannot begin transaction")
		}
		defer tx.Rollback() // nolint

		grp, err := group.LoadGroupByID(tx, data.GroupID)
		if err != nil {
			return err
		}

		if old.GroupID != data.GroupID || old.Name != data.Name {
			// check that the group exists and user is admin for group id
			if err := group.CheckUserIsGroupAdmin(grp, u); err != nil {
				return err
			}

			// check that no worker model already exists for same group/name
			if _, err := worker.LoadWorkerModelByNameAndGroupID(tx, data.Name, grp.ID); err == nil {
				return sdk.NewErrorFrom(sdk.ErrAlreadyExist, "an action already exists for given name on this group")
			}
		}

		// provision is allowed only for CDS Admin or by user with a restricted model
		if !u.Admin && !data.Restricted {
			data.Provision = 0
		}

		// if a model pattern is given try to get it from database
		if data.PatternName != "" {
			modelPattern, err := worker.LoadWorkerModelPatternByName(tx, data.Type, data.PatternName)
			if err != nil {
				return sdk.NewErrorWithStack(err, sdk.NewErrorFrom(sdk.ErrWrongRequest, "invalid given worker model name"))
			}

			// set pattern data on given model
			switch data.Type {
			case sdk.Docker:
				data.ModelDocker.Cmd = modelPattern.Model.Cmd
				data.ModelDocker.Shell = modelPattern.Model.Shell
				data.ModelDocker.Envs = modelPattern.Model.Envs
			default:
				data.ModelVirtualMachine.PreCmd = modelPattern.Model.PreCmd
				data.ModelVirtualMachine.Cmd = modelPattern.Model.Cmd
				data.ModelVirtualMachine.PostCmd = modelPattern.Model.PostCmd
			}
		}

		// update fields from request data
		model := sdk.Model(*old)
		model.Update(data)

		// update model in db
		if err := worker.UpdateWorkerModel(tx, &model); err != nil {
			return sdk.WrapError(err, "cannot update worker model")
		}

		// if the model has been renamed, we will have to update requirements
		renamed := data.Name != old.Name

		// update requirements if needed
		// FIXME requirements shoudl contains group name
		if renamed {
			actionsID, err := action.UpdateRequirementsValue(tx, old.Name, model.Name, sdk.ModelRequirement)
			if err != nil {
				return sdk.WrapError(err, "cannot update action requirements")
			}
			log.Debug("putWorkerModelHandler> Update requirement %s/%s for actions %v", grp.Name, model.Name, actionsID)
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "unable to commit transaction")
		}

		key := cache.Key("api:workermodels:*")
		api.Cache.DeleteAll(key)

		new, err := worker.LoadWorkerModelByID(api.mustDB(), model.ID)
		if err != nil {
			return err
		}

		new.Editable = true

		return service.WriteJSON(w, new, http.StatusOK)
	}
}

func (api *API) deleteWorkerModelHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)

		groupName := vars["groupName"]
		modelName := vars["permModelName"]

		g, err := group.LoadGroup(api.mustDB(), groupName)
		if err != nil {
			return err
		}

		m, err := worker.LoadWorkerModelByNameAndGroupID(api.mustDB(), modelName, g.ID)
		if err != nil {
			return sdk.WrapError(err, "cannot load worker model")
		}

		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WrapError(err, "cannot start transaction")
		}

		if err := worker.DeleteWorkerModel(tx, m.ID); err != nil {
			return sdk.WrapError(err, "cannot delete worker model")
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "Cannot commit transaction")
		}

		key := cache.Key("api:workermodels:*")
		api.Cache.DeleteAll(key)

		return nil
	}
}

func (api *API) getWorkerModelHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)

		groupName := vars["groupName"]
		modelName := vars["permModelName"]

		g, err := group.LoadGroup(api.mustDB(), groupName)
		if err != nil {
			return err
		}

		// FIXME implements load options for worker model dao.
		m, err := worker.LoadWorkerModelByNameAndGroupID(api.mustDB(), modelName, g.ID)
		if err != nil {
			return sdk.WrapError(err, "cannot load worker model")
		}

		if err := group.CheckUserIsGroupAdmin(g, deprecatedGetUser(ctx)); err == nil {
			m.Editable = true
		}

		return service.WriteJSON(w, m, http.StatusOK)
	}
}

func (api *API) getWorkerModelsHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		if err := r.ParseForm(); err != nil {
			return sdk.WrapError(sdk.ErrWrongRequest, "getWorkerModels> cannot parse form")
		}

		binary := r.FormValue("binary")
		state := r.FormValue("state")
		var opt *worker.StateLoadOption
		switch state {
		case "", worker.StateDisabled.String(), worker.StateOfficial.String(), worker.StateError.String(), worker.StateRegister.String(), worker.StateDeprecated.String(), worker.StateActive.String():
			opt = new(worker.StateLoadOption)
			*opt = worker.StateLoadOption(state)
			break
		default:
			return sdk.ErrWrongRequest
		}

		u := deprecatedGetUser(ctx)
		if u == nil || u.ID == 0 {
			var username string
			if u != nil {
				username = u.Username
			}
			return sdk.WrapError(sdk.ErrForbidden, "getWorkerModels> this route can't be called by worker or hatchery named %s", username)
		}

		models := []sdk.Model{}
		var errbyuser error
		if binary != "" {
			models, errbyuser = worker.LoadWorkerModelsByUserAndBinary(api.mustDB(), deprecatedGetUser(ctx), binary)
		} else {
			models, errbyuser = worker.LoadWorkerModelsByUser(api.mustDB(), api.Cache, deprecatedGetUser(ctx), opt)
		}
		if errbyuser != nil {
			return sdk.WrapError(errbyuser, "getWorkerModels> cannot load worker models for user id %d", deprecatedGetUser(ctx).ID)
		}

		return service.WriteJSON(w, models, http.StatusOK)
	}
}

func (api *API) getWorkerModelUsageHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)

		groupName := vars["groupName"]
		modelName := vars["permModelName"]

		g, err := group.LoadGroup(api.mustDB(), groupName)
		if err != nil {
			return err
		}

		m, err := worker.LoadWorkerModelByNameAndGroupID(api.mustDB(), modelName, g.ID)
		if err != nil {
			return sdk.WrapError(err, "cannot load worker model")
		}

		pips, err := pipeline.LoadByWorkerModelName(api.mustDB(), m.Name, deprecatedGetUser(ctx))
		if err != nil {
			return sdk.WrapError(err, "cannot load pipelines linked to worker model")
		}

		return service.WriteJSON(w, pips, http.StatusOK)
	}
}

func (api *API) getWorkerModelTypesHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		return service.WriteJSON(w, sdk.AvailableWorkerModelType, http.StatusOK)
	}
}

func (api *API) getWorkerModelCommunicationsHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		return service.WriteJSON(w, sdk.AvailableWorkerModelCommunication, http.StatusOK)
	}
}
