package api

import (
	"context"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/workermodel"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
)

func (api *API) postWorkerModelHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		consumer := getUserConsumer(ctx)

		// parse request and check data validity
		var data sdk.Model
		if err := service.UnmarshalBody(r, &data); err != nil {
			return err
		}
		if err := data.IsValid(); err != nil {
			return err
		}
		if err := data.IsValidType(); err != nil {
			return err
		}

		// Verify the image if any whitelist is setup
		if data.ModelDocker.Image != "" && len(api.WorkerModelDockerImageWhiteList) > 0 {
			var allowedImage = false
			for _, r := range api.WorkerModelDockerImageWhiteList { // At least one regexp must match
				if r.MatchString(data.ModelDocker.Image) {
					allowedImage = true
					break
				}
			}
			if !allowedImage {
				return sdk.WithStack(sdk.ErrWrongRequest)
			}
		}

		// check that given group id exits and that the user is admin of the group
		grp, err := group.LoadByID(ctx, api.mustDB(), data.GroupID, group.LoadOptions.WithMembers)
		if err != nil {
			return sdk.NewError(sdk.ErrWrongRequest, err)
		}
		if !isGroupAdmin(ctx, grp) {
			if isAdmin(ctx) {
				trackSudo(ctx, w)
			} else {
				return sdk.NewErrorFrom(sdk.ErrForbidden, "you should be admin of the group to import a worker model")
			}
		}

		if !isAdmin(ctx) {
			// if current user is not admin and model is not restricted, a pattern should be given
			if !data.Restricted && data.PatternName == "" {
				return sdk.NewErrorFrom(sdk.ErrWorkerModelNoPattern, "missing model pattern name")
			}
		}

		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WrapError(err, "cannot begin transaction")
		}
		defer tx.Rollback() // nolint

		model, err := workermodel.Create(ctx, tx, data, consumer)
		if err != nil {
			return err
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "unable to commit transaction")
		}

		// reload complete worker model
		new, err := workermodel.LoadByID(ctx, api.mustDB(), model.ID, workermodel.LoadOptions.Default)
		if err != nil {
			return err
		}

		new.Editable = true

		return service.WriteJSON(w, new, http.StatusOK)
	}
}

func (api *API) putWorkerModelHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		if isService(ctx) {
			return sdk.WithStack(sdk.ErrForbidden)
		}

		vars := mux.Vars(r)

		groupName := vars["permGroupName"]
		modelName := vars["permModelName"]

		g, err := group.LoadByName(ctx, api.mustDB(), groupName, group.LoadOptions.WithMembers)
		if err != nil {
			return err
		}

		old, err := workermodel.LoadByNameAndGroupID(ctx, api.mustDB(), modelName, g.ID)
		if err != nil {
			return sdk.WrapError(err, "cannot load worker model with name %s for group %s", modelName, g.Name)
		}

		// parse request and validate given data
		var data sdk.Model
		if err := service.UnmarshalBody(r, &data); err != nil {
			return err
		}
		if err := data.IsValid(); err != nil {
			return err
		}

		// Verify the image if any whitelist is setup
		if data.ModelDocker.Image != "" && len(api.WorkerModelDockerImageWhiteList) > 0 {
			var allowedImage = false
			for _, r := range api.WorkerModelDockerImageWhiteList { // At least one regexp must match
				if r.MatchString(data.ModelDocker.Image) {
					allowedImage = true
					break
				}
			}
			if !allowedImage {
				return sdk.WithStack(sdk.ErrWrongRequest)
			}
		}

		if old.GroupID != data.GroupID {
			// check that given group id exits and that the user is admin of the group
			grp, err := group.LoadByID(ctx, api.mustDB(), data.GroupID, group.LoadOptions.WithMembers)
			if err != nil {
				return sdk.NewError(sdk.ErrWrongRequest, err)
			}
			if !isGroupAdmin(ctx, grp) {
				if isAdmin(ctx) {
					trackSudo(ctx, w)
				} else {
					return sdk.NewErrorFrom(sdk.ErrForbidden, "you should be admin of the group to import a worker model")
				}
			}
		}

		if !isAdmin(ctx) {
			if err := workermodel.CopyModelTypeData(old, &data); err != nil {
				return err
			}
		}

		if err := data.IsValidType(); err != nil {
			return err
		}

		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WrapError(err, "cannot begin transaction")
		}
		defer tx.Rollback() // nolint

		model, err := workermodel.Update(ctx, tx, old, data)
		if err != nil {
			return err
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "unable to commit transaction")
		}

		new, err := workermodel.LoadByID(ctx, api.mustDB(), model.ID, workermodel.LoadOptions.Default)
		if err != nil {
			return err
		}

		new.Editable = true

		return service.WriteJSON(w, new, http.StatusOK)
	}
}

func (api *API) deleteWorkerModelHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		if isService(ctx) {
			return sdk.WithStack(sdk.ErrForbidden)
		}

		vars := mux.Vars(r)

		groupName := vars["permGroupName"]
		modelName := vars["permModelName"]

		g, err := group.LoadByName(ctx, api.mustDB(), groupName)
		if err != nil {
			return err
		}

		m, err := workermodel.LoadByNameAndGroupID(ctx, api.mustDB(), modelName, g.ID, workermodel.LoadOptions.Default)
		if err != nil {
			return err
		}

		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WrapError(err, "cannot start transaction")
		}
		defer tx.Rollback() // nolint

		if err := workermodel.DeleteByID(tx, m.ID); err != nil {
			return sdk.WrapError(err, "cannot delete worker model")
		}

		if err := tx.Commit(); err != nil {
			return sdk.WithStack(err)
		}

		return nil
	}
}

func (api *API) getWorkerModelHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)

		groupName := vars["permGroupName"]
		modelName := vars["permModelName"]

		g, err := group.LoadByName(ctx, api.mustDB(), groupName, group.LoadOptions.WithMembers)
		if err != nil {
			return err
		}

		m, err := workermodel.LoadByNameAndGroupID(ctx, api.mustDB(), modelName, g.ID, workermodel.LoadOptions.Default)
		if err != nil {
			return err
		}

		m.Editable = isGroupAdmin(ctx, g) || isAdmin(ctx)

		return service.WriteJSON(w, m, http.StatusOK)
	}
}

func (api *API) getWorkerModelsHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		if err := r.ParseForm(); err != nil {
			return sdk.WrapError(sdk.ErrWrongRequest, "cannot parse form")
		}

		var filter workermodel.LoadFilter

		binary := r.FormValue("binary")
		if binary != "" {
			filter.Binary = binary
		}

		stateString := r.FormValue("state")
		if stateString != "" {
			o := workermodel.StateFilter(stateString)
			if err := o.IsValid(); err != nil {
				return err
			}
			filter.State = o
		}

		consumer := getUserConsumer(ctx)

		models := make([]sdk.Model, 0)

		if consumer.AuthConsumerUser.Worker != nil {
			if consumer.AuthConsumerUser.Worker.ModelID != nil {
				model, err := workermodel.LoadByID(ctx, api.mustDB(), *consumer.AuthConsumerUser.Worker.ModelID, workermodel.LoadOptions.Default)
				if err != nil {
					return err
				}
				models = []sdk.Model{*model}
			}
			return service.WriteJSON(w, models, http.StatusOK)
		}

		hatchery, err := isHatchery(ctx)
		if err != nil {
			return err
		}
		if hatchery && len(consumer.AuthConsumerUser.GroupIDs) > 0 {
			models, err = workermodel.LoadAllByGroupIDs(ctx, api.mustDB(), consumer.GetGroupIDs(), &filter, workermodel.LoadOptions.Default)
		} else if isMaintainer(ctx) {
			models, err = workermodel.LoadAll(ctx, api.mustDB(), &filter, workermodel.LoadOptions.Default)
		} else {
			models, err = workermodel.LoadAllByGroupIDs(ctx, api.mustDB(), append(consumer.GetGroupIDs(), group.SharedInfraGroup.ID),
				&filter, workermodel.LoadOptions.Default)
		}
		if err != nil {
			return sdk.WrapError(err, "cannot load worker models")
		}

		return service.WriteJSON(w, models, http.StatusOK)
	}
}

func (api *API) getWorkerModelUsageHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)

		groupName := vars["permGroupName"]
		modelName := vars["permModelName"]

		g, err := group.LoadByName(ctx, api.mustDB(), groupName)
		if err != nil {
			return err
		}

		m, err := workermodel.LoadByNameAndGroupID(ctx, api.mustDB(), modelName, g.ID, workermodel.LoadOptions.Default)
		if err != nil {
			return err
		}

		var pips []sdk.Pipeline
		if isMaintainer(ctx) {
			pips, err = pipeline.LoadByWorkerModel(ctx, api.mustDB(), m)
		} else {
			pips, err = pipeline.LoadByWorkerModelAndGroupIDs(ctx, api.mustDB(), m,
				append(getUserConsumer(ctx).GetGroupIDs(), group.SharedInfraGroup.ID))
		}
		if err != nil {
			return sdk.WrapError(err, "cannot load pipelines linked to worker model")
		}

		return service.WriteJSON(w, pips, http.StatusOK)
	}
}

func (api *API) getWorkerModelsForProjectHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars[permProjectKey]

		proj, err := project.Load(ctx, api.mustDB(), key, project.LoadOptions.WithGroups)
		if err != nil {
			return sdk.WrapError(err, "unable to load project %s", key)
		}

		groupIDs := make([]int64, len(proj.ProjectGroups))
		for i := range proj.ProjectGroups {
			groupIDs[i] = proj.ProjectGroups[i].Group.ID
		}

		models, err := workermodel.LoadAllActiveAndNotDeprecatedForGroupIDs(ctx, api.mustDB(), append(groupIDs, group.SharedInfraGroup.ID), workermodel.LoadOptions.Default)
		if err != nil {
			return err
		}

		return service.WriteJSON(w, models, http.StatusOK)
	}
}

func (api *API) getWorkerModelsForGroupHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)

		groupName := vars["permGroupName"]

		g, err := group.LoadByName(ctx, api.mustDB(), groupName)
		if err != nil {
			return err
		}

		if !isGroupMember(ctx, g) && !isAdmin(ctx) {
			return sdk.WithStack(sdk.ErrForbidden)
		}

		wms, err := workermodel.LoadAllActiveAndNotDeprecatedForGroupIDs(ctx, api.mustDB(), []int64{g.ID, group.SharedInfraGroup.ID}, workermodel.LoadOptions.Default)
		if err != nil {
			return err
		}

		return service.WriteJSON(w, wms, http.StatusOK)
	}
}

func (api *API) getWorkerModelTypesHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		return service.WriteJSON(w, sdk.AvailableWorkerModelType, http.StatusOK)
	}
}
