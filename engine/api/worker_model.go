package api

import (
	"context"
	"database/sql"
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

func (api *API) addWorkerModelHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		// Unmarshal body
		var model sdk.Model
		if err := service.UnmarshalBody(r, &model); err != nil {
			return sdk.WrapError(err, "Cannot unmarshal body")
		}

		if model.Type == "" {
			return sdk.WrapError(sdk.ErrWrongRequest, "addWorkerModel> Invalid type (empty)")
		}

		var modelPattern *sdk.ModelPattern
		if model.PatternName != "" {
			var errP error
			modelPattern, errP = worker.LoadWorkerModelPatternByName(api.mustDB(), model.Type, model.PatternName)
			if errP != nil {
				return sdk.WrapError(sdk.ErrWrongRequest, "addWorkerModel> Cannot load worker model pattern %s : %v", model.PatternName, errP)
			}
		}

		currentUser := deprecatedGetUser(ctx)
		//User must be admin of the group set in the model
		var isGroupAdmin bool
	currentUGroup:
		for _, g := range currentUser.Groups {
			if g.ID == model.GroupID {
				for _, a := range g.Admins {
					if a.ID == currentUser.ID {
						isGroupAdmin = true
						break currentUGroup
					}
				}
			}
		}

		//User should have the right permission or be admin
		if !currentUser.Admin && !isGroupAdmin {
			return sdk.ErrWorkerModelNoAdmin
		}

		switch model.Type {
		case sdk.Docker:
			if model.ModelDocker.Image == "" {
				return sdk.WrapError(sdk.ErrWrongRequest, "addWorkerModel> Invalid worker image")
			}
			if !currentUser.Admin && !model.Restricted {
				if modelPattern == nil {
					return sdk.ErrWorkerModelNoPattern
				}
				model.ModelDocker.Cmd = modelPattern.Model.Cmd
				model.ModelDocker.Shell = modelPattern.Model.Shell
				model.ModelDocker.Envs = modelPattern.Model.Envs
			}
			if model.ModelDocker.Cmd == "" || model.ModelDocker.Shell == "" {
				return sdk.WrapError(sdk.ErrWrongRequest, "updateWorkerModel> Invalid worker command or invalid shell command")
			}
		default:
			if model.ModelVirtualMachine.Image == "" {
				return sdk.WrapError(sdk.ErrWrongRequest, "addWorkerModel> Invalid worker command or invalid image")
			}
			if !currentUser.Admin && !model.Restricted {
				if modelPattern == nil {
					return sdk.ErrWorkerModelNoPattern
				}
				model.ModelVirtualMachine.PreCmd = modelPattern.Model.PreCmd
				model.ModelVirtualMachine.Cmd = modelPattern.Model.Cmd
				model.ModelVirtualMachine.PostCmd = modelPattern.Model.PostCmd
			}
		}

		if len(model.Name) == 0 {
			return sdk.WrapError(sdk.ErrWrongRequest, "addWorkerModel> model name is empty")
		}

		if model.GroupID == 0 {
			return sdk.WrapError(sdk.ErrWrongRequest, "addWorkerModel> groupID should be set")
		}

		if group.IsDefaultGroupID(model.GroupID) {
			return sdk.WrapError(sdk.ErrWrongRequest, "addWorkerModel> this group can't be owner of a worker model")
		}

		// check if worker model already exists
		if _, err := worker.LoadWorkerModelByName(api.mustDB(), model.Name); err == nil {
			return sdk.WrapError(sdk.ErrModelNameExist, "addWorkerModel> worker model already exists")
		}

		// provision is allowed only for CDS Admin
		// or by currentUser with a restricted model
		if !currentUser.Admin && !model.Restricted {
			model.Provision = 0
		}

		model.CreatedBy = sdk.User{
			Email:    currentUser.Email,
			Username: currentUser.Username,
			Admin:    currentUser.Admin,
			Fullname: currentUser.Fullname,
			ID:       currentUser.ID,
			Origin:   currentUser.Origin,
		}

		// Insert model in db
		if err := worker.InsertWorkerModel(api.mustDB(), &model); err != nil {
			return sdk.WrapError(err, "cannot add worker model")
		}

		key := cache.Key("api:workermodels:*")
		api.Cache.DeleteAll(key)

		return service.WriteJSON(w, model, http.StatusOK)
	}
}

func (api *API) bookWorkerModelHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		workerModelID, errr := requestVarInt(r, "permModelID")
		if errr != nil {
			return sdk.WrapError(errr, "bookWorkerModelHandler> Invalid permModelID")
		}
		if _, err := worker.BookForRegister(api.Cache, workerModelID, getHatchery(ctx)); err != nil {
			return sdk.WithStack(err)
		}
		return nil
	}
}

func (api *API) spawnErrorWorkerModelHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		var spawnErrorForm sdk.SpawnErrorForm
		if err := service.UnmarshalBody(r, &spawnErrorForm); err != nil {
			return sdk.WrapError(err, "Unable to parse spawn error form")
		}

		workerModelID, err := requestVarInt(r, "permModelID")
		if err != nil {
			return err
		}

		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WithStack(err)
		}
		defer tx.Rollback()

		model, err := worker.LoadWorkerModelByID(tx, workerModelID)
		if err != nil {
			return err
		}

		if err := worker.UpdateSpawnErrorWorkerModel(tx, model.ID, spawnErrorForm); err != nil {
			return sdk.WrapError(err, "cannot update spawn error on worker model")
		}

		if err := tx.Commit(); err != nil {
			return err
		}

		key := cache.Key("api:workermodels:*")
		api.Cache.DeleteAll(key)

		return service.WriteJSON(w, nil, http.StatusOK)
	}
}

func (api *API) updateWorkerModelHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		workerModelID, err := requestVarInt(r, "permModelID")
		if err != nil {
			return err
		}

		old, errLoad := worker.LoadWorkerModelByID(api.mustDB(), workerModelID)
		if errLoad != nil {
			return sdk.WrapError(errLoad, "cannot load worker model by id")
		}

		// Unmarshal body
		var model sdk.Model
		if err := service.UnmarshalBody(r, &model); err != nil {
			return sdk.WrapError(err, "Cannot unmarshal body")
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
				return sdk.WrapError(sdk.ErrModelNameExist, "updateWorkerModel> worker model already exists")
			}
		}

		//If the model image has not been set, keep the old image
		if model.ModelDocker.Image == "" && model.ModelVirtualMachine.Image == "" {
			switch model.Type {
			case sdk.Docker:
				model.ModelDocker.Image = old.ModelDocker.Image
			default:
				model.ModelVirtualMachine.Image = old.ModelVirtualMachine.Image
			}
		}

		//If the model RegisteredCapabilities has not been set, keep the old RegisteredCapabilities
		if len(model.RegisteredCapabilities) == 0 {
			model.RegisteredCapabilities = old.RegisteredCapabilities
		}

		//If the model GroupID has not been set, keep the old GroupID
		if model.GroupID == 0 {
			model.GroupID = old.GroupID
		}

		// we can't select the default group
		if group.IsDefaultGroupID(model.GroupID) {
			return sdk.WrapError(sdk.ErrWrongRequest, "this group can't be owner of a worker model")
		}

		//If the model Type has not been set, keep the old Type
		if model.Type == "" {
			model.Type = old.Type
		}

		var modelPattern *sdk.ModelPattern
		if model.PatternName != "" {
			var errP error
			modelPattern, errP = worker.LoadWorkerModelPatternByName(api.mustDB(), model.Type, model.PatternName)
			if errP != nil {
				return sdk.WrapError(sdk.ErrWrongRequest, "cannot load worker model pattern %s : %v", model.PatternName, errP)
			}
		}

		user := deprecatedGetUser(ctx)
		//User must be admin of the group set in the model
		var ok bool
	currentUGroup:
		for _, g := range deprecatedGetUser(ctx).Groups {
			if g.ID == model.GroupID {
				for _, a := range g.Admins {
					if a.ID == deprecatedGetUser(ctx).ID {
						ok = true
						break currentUGroup
					}
				}
			}
		}

		//User should have the right permission or be admin
		if !deprecatedGetUser(ctx).Admin && !ok {
			return sdk.ErrWorkerModelNoAdmin
		}

		switch model.Type {
		case sdk.Docker:
			if model.ModelDocker.Image == "" {
				return sdk.WrapError(sdk.ErrWrongRequest, "invalid worker image")
			}
			if !user.Admin && !model.Restricted {
				if modelPattern == nil {
					if old.Type != sdk.Docker { // Forbidden because we can't fetch previous user data
						return sdk.WrapError(sdk.ErrWorkerModelNoPattern, "we can't fetch previous user data because type is different")
					}
					model.ModelDocker.Cmd = old.ModelDocker.Cmd
					model.ModelDocker.Shell = old.ModelDocker.Shell
					model.ModelDocker.Envs = old.ModelDocker.Envs
				} else {
					model.ModelDocker.Cmd = modelPattern.Model.Cmd
					model.ModelDocker.Shell = modelPattern.Model.Shell
					model.ModelDocker.Envs = modelPattern.Model.Envs
				}
			}
			if model.ModelDocker.Cmd == "" || model.ModelDocker.Shell == "" {
				return sdk.WrapError(sdk.ErrWrongRequest, "invalid worker command or invalid shell command")
			}
		default:
			if model.ModelVirtualMachine.Image == "" {
				return sdk.WrapError(sdk.ErrWrongRequest, "invalid worker command or invalid image")
			}
			if !user.Admin && !model.Restricted {
				if modelPattern == nil {
					if old.Type == sdk.Docker { // Forbidden because we can't fetch previous user data
						return sdk.WrapError(sdk.ErrWorkerModelNoPattern, "we can't fetch previous user data because type is different")
					}
					model.ModelVirtualMachine.PreCmd = old.ModelVirtualMachine.PreCmd
					model.ModelVirtualMachine.Cmd = old.ModelVirtualMachine.Cmd
					model.ModelVirtualMachine.PostCmd = old.ModelVirtualMachine.PostCmd
				} else {
					model.ModelVirtualMachine.PreCmd = modelPattern.Model.PreCmd
					model.ModelVirtualMachine.Cmd = modelPattern.Model.Cmd
					model.ModelVirtualMachine.PostCmd = modelPattern.Model.PostCmd
				}
			}
		}

		//If the model modelID has not been set, keep the old modelID
		if model.ID == 0 {
			model.ID = old.ID
		}

		// provision is allowed only for CDS Admin
		// or by user with a restricted model
		if !deprecatedGetUser(ctx).Admin && !model.Restricted {
			model.Provision = 0
		}

		if workerModelID != model.ID {
			return sdk.WrapError(sdk.ErrInvalidID, "wrong ID")
		}

		tx, errtx := api.mustDB().Begin()
		if errtx != nil {
			return sdk.WrapError(errtx, "unable to start transaction")
		}

		defer tx.Rollback()

		// update model in db
		if err := worker.UpdateWorkerModel(tx, &model); err != nil {
			return sdk.WrapError(err, "cannot update worker model")
		}

		// update requirements if needed
		if renamed {
			actionsID, erru := action.UpdateRequirementsValue(tx, old.Name, model.Name, sdk.ModelRequirement)
			if erru != nil {
				return sdk.WrapError(erru, "cannot update action requirements")
			}
			log.Debug("updateWorkerModel> Update action %v", actionsID)
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "unable to commit transaction")
		}

		key := cache.Key("api:workermodels:*")
		api.Cache.DeleteAll(key)

		return service.WriteJSON(w, model, http.StatusOK)
	}
}

func (api *API) deleteWorkerModelHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		workerModelID, errr := requestVarInt(r, "permModelID")
		if errr != nil {
			return sdk.WrapError(errr, "deleteWorkerModel> Invalid permModelID")
		}

		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WrapError(err, "Cannot start transaction")
		}

		if err := worker.DeleteWorkerModel(tx, workerModelID); err != nil {
			return sdk.WrapError(err, "deleteWorkerModel: cannot delete worker model")
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "Cannot commit transaction")
		}

		key := cache.Key("api:workermodels:*")
		api.Cache.DeleteAll(key)

		return nil
	}
}

func (api *API) getWorkerModel(w http.ResponseWriter, r *http.Request, name string) error {
	m, err := worker.LoadWorkerModelByName(api.mustDB(), name)
	if err != nil {
		return sdk.WrapError(err, "cannot load worker model")
	}
	return service.WriteJSON(w, m, http.StatusOK)
}

func (api *API) getWorkerModelUsageHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		workerModelID, errr := requestVarInt(r, "modelID")
		if errr != nil {
			return sdk.WrapError(errr, "Invalid modelID")
		}
		db := api.mustDB()

		wm, err := worker.LoadWorkerModelByID(db, workerModelID)
		if err != nil {
			return sdk.WrapError(err, "cannot load worker model for id %d", workerModelID)
		}

		pips, errP := pipeline.LoadByWorkerModelName(db, wm.Name, deprecatedGetUser(ctx))
		if errP != nil {
			return sdk.WrapError(errP, "Cannot load pipelines linked to worker model")
		}

		return service.WriteJSON(w, pips, http.StatusOK)
	}
}

func (api *API) getWorkerModelsEnabledHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		h := getHatchery(ctx)
		if h == nil || h.GroupID == nil || *h.GroupID == 0 {
			return sdk.WrapError(sdk.ErrWrongRequest, "getWorkerModelsEnabled> this route can be called only by hatchery: %+v", h)
		}
		models, errgroup := worker.LoadWorkerModelsUsableOnGroup(api.mustDB(), api.Cache, *h.GroupID, group.SharedInfraGroup.ID)
		if errgroup != nil {
			return sdk.WrapError(errgroup, "getWorkerModelsEnabled> cannot load worker models for hatchery %d with group %d", h.ID, *h.GroupID)
		}
		return service.WriteJSON(w, models, http.StatusOK)
	}
}

func (api *API) getWorkerModelsHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		if err := r.ParseForm(); err != nil {
			return sdk.WrapError(sdk.ErrWrongRequest, "getWorkerModels> cannot parse form")
		}

		name := r.FormValue("name")
		if name != "" {
			return api.getWorkerModel(w, r, name)
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

func (api *API) putWorkerModelPatternHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		patternName := vars["name"]
		patternType := vars["type"]

		// Unmarshal body
		var modelPattern sdk.ModelPattern
		if err := service.UnmarshalBody(r, &modelPattern); err != nil {
			return sdk.WrapError(err, "Cannot unmarshal body")
		}

		if !sdk.NamePatternRegex.MatchString(modelPattern.Name) {
			return sdk.ErrInvalidName
		}

		if modelPattern.Model.Cmd == "" {
			return sdk.ErrInvalidPatternModel
		}

		if modelPattern.Type == sdk.Docker && modelPattern.Model.Shell == "" {
			return sdk.WrapError(sdk.ErrWrongRequest, "putWorkerModelPatternHandler> Cannot update a worker model pattern for %s without shell command", sdk.Docker)
		}

		var typeFound bool
		for _, availableType := range sdk.AvailableWorkerModelType {
			if availableType == modelPattern.Type {
				typeFound = true
				break
			}
		}

		if !typeFound {
			return sdk.ErrInvalidPatternModel
		}

		oldWmp, errOld := worker.LoadWorkerModelPatternByName(api.mustDB(), patternType, patternName)
		if errOld != nil {
			if sdk.Cause(errOld) == sql.ErrNoRows {
				return sdk.WrapError(sdk.ErrNotFound, "putWorkerModelPatternHandler> cannot load worker model pattern (%s/%s) : %v", patternType, patternName, errOld)
			}
			return sdk.WrapError(errOld, "putWorkerModelPatternHandler> cannot load worker model pattern")
		}
		modelPattern.ID = oldWmp.ID

		if err := worker.UpdateWorkerModelPattern(api.mustDB(), &modelPattern); err != nil {
			return sdk.WrapError(err, "cannot update worker model pattern")
		}

		return service.WriteJSON(w, modelPattern, http.StatusOK)
	}
}

func (api *API) deleteWorkerModelPatternHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		patternName := vars["name"]
		patternType := vars["type"]

		wmp, err := worker.LoadWorkerModelPatternByName(api.mustDB(), patternType, patternName)
		if err != nil {
			if sdk.Cause(err) == sql.ErrNoRows {
				return sdk.WrapError(sdk.ErrNotFound, "deleteWorkerModelPatternHandler> Cannot load worker model by name (%s/%s)", patternType, patternName)
			}
			return sdk.WrapError(err, "Cannot load worker model by name (%s/%s) : %v", patternType, patternName, err)
		}

		if err := worker.DeleteWorkerModelPattern(api.mustDB(), wmp.ID); err != nil {
			return sdk.WrapError(err, "Cannot delete worker model (%s/%s) : %v", patternType, patternName, err)
		}

		return service.WriteJSON(w, nil, http.StatusOK)
	}
}

func (api *API) getWorkerModelPatternHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		if deprecatedGetUser(ctx).ID == 0 {
			var username string
			if deprecatedGetUser(ctx) != nil {
				username = deprecatedGetUser(ctx).Username
			}
			return sdk.WrapError(sdk.ErrForbidden, "getWorkerModels> this route can't be called by worker or hatchery named %s", username)
		}
		vars := mux.Vars(r)
		patternName := vars["name"]
		patternType := vars["type"]

		modelPattern, err := worker.LoadWorkerModelPatternByName(api.mustDB(), patternType, patternName)
		if err != nil {
			return sdk.WrapError(err, "cannot load worker model patterns")
		}

		return service.WriteJSON(w, modelPattern, http.StatusOK)
	}
}

func (api *API) postAddWorkerModelPatternHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		// Unmarshal body
		var modelPattern sdk.ModelPattern
		if err := service.UnmarshalBody(r, &modelPattern); err != nil {
			return sdk.WrapError(err, "Cannot unmarshal body")
		}

		if !sdk.NamePatternRegex.MatchString(modelPattern.Name) {
			return sdk.ErrInvalidName
		}

		if modelPattern.Model.Cmd == "" {
			return sdk.ErrInvalidPatternModel
		}

		if modelPattern.Type == sdk.Docker && modelPattern.Model.Shell == "" {
			return sdk.WrapError(sdk.ErrWrongRequest, "postAddWorkerModelPatternHandler> Cannot add a worker model pattern for %s without shell command", sdk.Docker)
		}

		var typeFound bool
		for _, availableType := range sdk.AvailableWorkerModelType {
			if availableType == modelPattern.Type {
				typeFound = true
				break
			}
		}

		if !typeFound {
			return sdk.ErrInvalidPatternModel
		}

		// Insert model pattern in db
		if err := worker.InsertWorkerModelPattern(api.mustDB(), &modelPattern); err != nil {
			return sdk.WrapError(err, "cannot add worker model pattern")
		}

		return service.WriteJSON(w, modelPattern, http.StatusOK)
	}
}

func (api *API) getWorkerModelPatternsHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		if deprecatedGetUser(ctx) == nil || deprecatedGetUser(ctx).ID == 0 {
			var username string
			if deprecatedGetUser(ctx) != nil {
				username = deprecatedGetUser(ctx).Username
			}
			return sdk.WrapError(sdk.ErrForbidden, "getWorkerModels> this route can't be called by worker or hatchery named %s", username)
		}

		modelPatterns, err := worker.LoadWorkerModelPatterns(api.mustDB())
		if err != nil {
			return sdk.WrapError(err, "cannot load worker model patterns")
		}

		return service.WriteJSON(w, modelPatterns, http.StatusOK)
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
