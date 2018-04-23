package api

import (
	"context"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/sdk"
)

func (api *API) getParametersInPipelineHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars["key"]
		pipelineName := vars["permPipelineKey"]

		p, err := pipeline.LoadPipeline(api.mustDB(ctx), key, pipelineName, false)
		if err != nil {
			return sdk.WrapError(err, "getParametersInPipelineHandler: Cannot load %s", pipelineName)
		}

		parameters, err := pipeline.GetAllParametersInPipeline(api.mustDB(ctx), p.ID)
		if err != nil {
			return sdk.WrapError(err, "getParametersInPipelineHandler: Cannot get parameters for pipeline %s", pipelineName)
		}

		return WriteJSON(w, parameters, http.StatusOK)
	}
}

func (api *API) deleteParameterFromPipelineHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars["key"]
		pipelineName := vars["permPipelineKey"]
		paramName := vars["name"]

		p, err := pipeline.LoadPipeline(api.mustDB(ctx), key, pipelineName, false)
		if err != nil {
			return sdk.WrapError(err, "deleteParameterFromPipelineHandler: Cannot load %s", pipelineName)
		}

		tx, err := api.mustDB(ctx).Begin()
		if err != nil {
			return sdk.WrapError(err, "deleteParameterFromPipelineHandler: Cannot start transaction")
		}
		defer tx.Rollback()

		if err := pipeline.DeleteParameterFromPipeline(tx, p.ID, paramName); err != nil {
			return sdk.WrapError(err, "deleteParameterFromPipelineHandler: Cannot delete %s", paramName)
		}

		proj, errproj := project.Load(api.mustDB(ctx), api.Cache, key, getUser(ctx))
		if errproj != nil {
			return sdk.WrapError(errproj, "deleteParameterFromPipelineHandler> unable to load project")
		}

		if err := pipeline.UpdatePipelineLastModified(tx, api.Cache, proj, p, getUser(ctx)); err != nil {
			return sdk.WrapError(err, "deleteParameterFromPipelineHandler> Cannot update pipeline last_modified date")
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "deleteParameterFromPipelineHandler: Cannot commit transaction")
		}

		p.Parameter, err = pipeline.GetAllParametersInPipeline(api.mustDB(ctx), p.ID)
		if err != nil {
			return sdk.WrapError(err, "deleteParameterFromPipelineHandler: Cannot load pipeline parameters")
		}
		return WriteJSON(w, p, http.StatusOK)
	}
}

// Deprecated
func (api *API) updateParametersInPipelineHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars["key"]
		pipelineName := vars["permPipelineKey"]

		proj, errP := project.Load(api.mustDB(ctx), api.Cache, key, getUser(ctx))
		if errP != nil {
			return sdk.WrapError(errP, "updateParametersInPipelineHandler> Cannot load project")
		}

		var pipParams []sdk.Parameter
		if err := UnmarshalBody(r, &pipParams); err != nil {
			return err
		}

		pip, err := pipeline.LoadPipeline(api.mustDB(ctx), proj.Key, pipelineName, false)
		if err != nil {
			return sdk.WrapError(err, "updateParametersInPipelineHandler: Cannot load %s", pipelineName)
		}
		pip.Parameter, err = pipeline.GetAllParametersInPipeline(api.mustDB(ctx), pip.ID)
		if err != nil {
			return sdk.WrapError(err, "updateParametersInPipelineHandler> Cannot GetAllParametersInPipeline")
		}

		tx, err := api.mustDB(ctx).Begin()
		if err != nil {
			return sdk.WrapError(err, "updateParametersInPipelineHandler: Cannot start transaction")
		}
		defer tx.Rollback()

		// Check with existing parameter to know whether parameter has been deleted, update or added
		var deleted, updated, added []sdk.Parameter
		var found bool
		for _, p := range pip.Parameter {
			found = false
			for _, new := range pipParams {
				// If we found a parameter with the same id but different value, then its modified
				if p.ID == new.ID {
					updated = append(updated, new)
					found = true
					break
				}
			}
			// If parameter is not found in new batch, then it  has been deleted
			if !found {
				deleted = append(deleted, p)
			}
		}

		// Added parameter are the one present in new batch but not in db
		for _, new := range pipParams {
			found = false
			for _, p := range pip.Parameter {
				if p.ID == new.ID {
					found = true
					break
				}
			}
			if !found {
				added = append(added, new)
			}
		}

		// Ok now permform actual update
		for i := range added {
			p := &added[i]
			if err := pipeline.InsertParameterInPipeline(tx, pip.ID, p); err != nil {
				return sdk.WrapError(err, "UpdatePipelineParameters> Cannot insert new params %s", p.Name)
			}
		}
		for _, p := range updated {
			if err := pipeline.UpdateParameterInPipeline(tx, pip.ID, p.Name, p); err != nil {
				return sdk.WrapError(err, "UpdatePipelineParameters> Cannot update parameter %s", p.Name)
			}
		}
		for _, p := range deleted {
			if err := pipeline.DeleteParameterFromPipeline(tx, pip.ID, p.Name); err != nil {
				return sdk.WrapError(err, "UpdatePipelineParameters> Cannot delete parameter %s", p.Name)
			}
		}

		if err := pipeline.UpdatePipelineLastModified(tx, api.Cache, proj, pip, getUser(ctx)); err != nil {

			return sdk.WrapError(err, "UpdatePipelineParameters> Cannot update pipeline last_modified date")
		}

		apps, errA := application.LoadByPipeline(tx, api.Cache, pip.ID, getUser(ctx))
		if errA != nil {
			return sdk.WrapError(errA, "UpdatePipelineParameters> Cannot load applications using pipeline")
		}

		for _, app := range apps {
			if err := application.UpdateLastModified(tx, api.Cache, &app, getUser(ctx)); err != nil {
				return sdk.WrapError(errA, "UpdatePipelineParameters> Cannot update application last modified date")
			}
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "updateParametersInPipelineHandler: Cannot commit transaction")
		}

		return WriteJSON(w, append(added, updated...), http.StatusOK)
	}
}

func (api *API) updateParameterInPipelineHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars["key"]
		pipelineName := vars["permPipelineKey"]
		paramName := vars["name"]

		var newParam sdk.Parameter
		if err := UnmarshalBody(r, &newParam); err != nil {
			return err
		}

		p, err := pipeline.LoadPipeline(api.mustDB(ctx), key, pipelineName, false)
		if err != nil {
			return sdk.WrapError(err, "updateParameterInPipelineHandler: Cannot load %s", pipelineName)
		}

		paramInPipeline, err := pipeline.CheckParameterInPipeline(api.mustDB(ctx), p.ID, paramName)
		if err != nil {
			return sdk.WrapError(err, "updateParameterInPipelineHandler: Cannot check if parameter %s is already in the pipeline %s", paramName, pipelineName)
		}

		if !paramInPipeline {
			return sdk.WrapError(sdk.ErrParameterNotExists, "updateParameterInPipelineHandler> unable to find parameter %s", paramName)
		}

		tx, err := api.mustDB(ctx).Begin()
		if err != nil {
			return sdk.WrapError(err, "updateParameterInPipelineHandler: Cannot start transaction")
		}
		defer tx.Rollback()

		if err := pipeline.UpdateParameterInPipeline(tx, p.ID, paramName, newParam); err != nil {
			return sdk.WrapError(err, "updateParameterInPipelineHandler: Cannot update parameter %s in pipeline %s", paramName, pipelineName)
		}

		proj, errproj := project.Load(api.mustDB(ctx), api.Cache, key, getUser(ctx))
		if errproj != nil {
			return sdk.WrapError(errproj, "updateParameterInPipelineHandler> unable to load project")
		}

		if err := pipeline.UpdatePipelineLastModified(tx, api.Cache, proj, p, getUser(ctx)); err != nil {
			return sdk.WrapError(err, "updateParameterInPipelineHandler: Cannot update pipeline last_modified date")
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "updateParameterInPipelineHandler: Cannot commit transaction")
		}

		p.Parameter, err = pipeline.GetAllParametersInPipeline(api.mustDB(ctx), p.ID)
		if err != nil {
			return sdk.WrapError(err, "updateParameterInPipelineHandler: Cannot load pipeline parameters")
		}
		return WriteJSON(w, p, http.StatusOK)
	}
}

func (api *API) addParameterInPipelineHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars["key"]
		pipelineName := vars["permPipelineKey"]
		paramName := vars["name"]

		var newParam sdk.Parameter
		if err := UnmarshalBody(r, &newParam); err != nil {
			return err
		}
		if newParam.Name != paramName {
			return sdk.WrapError(sdk.ErrWrongRequest, "addParameterInPipelineHandler> Wrong param name got %s instead of %s", newParam.Name, paramName)
		}

		p, err := pipeline.LoadPipeline(api.mustDB(ctx), key, pipelineName, false)
		if err != nil {
			return sdk.WrapError(err, "addParameterInPipelineHandler: Cannot load %s", pipelineName)
		}

		paramInProject, err := pipeline.CheckParameterInPipeline(api.mustDB(ctx), p.ID, paramName)
		if err != nil {
			return sdk.WrapError(err, "addParameterInPipelineHandler: Cannot check if parameter %s is already in the pipeline %s", paramName, pipelineName)
		}
		if paramInProject {
			return sdk.WrapError(sdk.ErrParameterExists, "addParameterInPipelineHandler:Parameter %s is already in the pipeline %s", paramName, pipelineName)
		}

		tx, err := api.mustDB(ctx).Begin()
		if err != nil {
			return sdk.WrapError(err, "addParameterInPipelineHandler: Cannot start transaction")
		}
		defer tx.Rollback()

		if !paramInProject {
			if err := pipeline.InsertParameterInPipeline(tx, p.ID, &newParam); err != nil {
				return sdk.WrapError(err, "addParameterInPipelineHandler: Cannot add parameter %s in pipeline %s", paramName, pipelineName)
			}
		}

		proj, errproj := project.Load(api.mustDB(ctx), api.Cache, key, getUser(ctx))
		if errproj != nil {
			return sdk.WrapError(errproj, "addParameterInPipelineHandler> unable to load project")
		}

		if err := pipeline.UpdatePipelineLastModified(tx, api.Cache, proj, p, getUser(ctx)); err != nil {
			return sdk.WrapError(err, "addParameterInPipelineHandler> Cannot update pipeline last_modified date")
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "addParameterInPipelineHandler: Cannot commit transaction")
		}

		p.Parameter, err = pipeline.GetAllParametersInPipeline(api.mustDB(ctx), p.ID)
		if err != nil {
			return sdk.WrapError(err, "addParameterInPipelineHandler: Cannot get pipeline parameters")
		}

		return WriteJSON(w, p, http.StatusOK)
	}
}
