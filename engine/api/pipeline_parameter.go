package api

import (
	"context"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/event"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
)

func (api *API) getParametersInPipelineHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars["key"]
		pipelineName := vars["permPipelineKey"]

		p, err := pipeline.LoadPipeline(api.mustDB(), key, pipelineName, false)
		if err != nil {
			return sdk.WrapError(err, "getParametersInPipelineHandler: Cannot load %s", pipelineName)
		}

		parameters, err := pipeline.GetAllParametersInPipeline(ctx, api.mustDB(), p.ID)
		if err != nil {
			return sdk.WrapError(err, "getParametersInPipelineHandler: Cannot get parameters for pipeline %s", pipelineName)
		}

		return service.WriteJSON(w, parameters, http.StatusOK)
	}
}

func (api *API) deleteParameterFromPipelineHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars["key"]
		pipelineName := vars["permPipelineKey"]
		paramName := vars["name"]

		p, err := pipeline.LoadPipeline(api.mustDB(), key, pipelineName, false)
		if err != nil {
			return sdk.WrapError(err, "deleteParameterFromPipelineHandler: Cannot load %s", pipelineName)
		}

		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WrapError(err, "deleteParameterFromPipelineHandler: Cannot start transaction")
		}
		defer tx.Rollback()

		if err := pipeline.DeleteParameterFromPipeline(tx, p.ID, paramName); err != nil {
			return sdk.WrapError(err, "deleteParameterFromPipelineHandler: Cannot delete %s", paramName)
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "deleteParameterFromPipelineHandler: Cannot commit transaction")
		}

		event.PublishPipelineParameterDelete(key, pipelineName, sdk.Parameter{Name: paramName}, getUser(ctx))

		p.Parameter, err = pipeline.GetAllParametersInPipeline(ctx, api.mustDB(), p.ID)
		if err != nil {
			return sdk.WrapError(err, "deleteParameterFromPipelineHandler: Cannot load pipeline parameters")
		}
		return service.WriteJSON(w, p, http.StatusOK)
	}
}

func (api *API) updateParameterInPipelineHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars["key"]
		pipelineName := vars["permPipelineKey"]
		paramName := vars["name"]

		var newParam sdk.Parameter
		if err := UnmarshalBody(r, &newParam); err != nil {
			return err
		}

		p, err := pipeline.LoadPipeline(api.mustDB(), key, pipelineName, false)
		if err != nil {
			return sdk.WrapError(err, "updateParameterInPipelineHandler: Cannot load %s", pipelineName)
		}

		oldParam := sdk.ParameterFind(&p.Parameter, paramName)

		if oldParam == nil {
			return sdk.WrapError(sdk.ErrParameterNotExists, "updateParameterInPipelineHandler> unable to find parameter %s", paramName)
		}

		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WrapError(err, "updateParameterInPipelineHandler: Cannot start transaction")
		}
		defer tx.Rollback()

		if err := pipeline.UpdateParameterInPipeline(tx, p.ID, paramName, newParam); err != nil {
			return sdk.WrapError(err, "updateParameterInPipelineHandler: Cannot update parameter %s in pipeline %s", paramName, pipelineName)
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "updateParameterInPipelineHandler: Cannot commit transaction")
		}

		event.PublishPipelineParameterUpdate(key, pipelineName, *oldParam, newParam, getUser(ctx))

		p.Parameter, err = pipeline.GetAllParametersInPipeline(ctx, api.mustDB(), p.ID)
		if err != nil {
			return sdk.WrapError(err, "updateParameterInPipelineHandler: Cannot load pipeline parameters")
		}
		return service.WriteJSON(w, p, http.StatusOK)
	}
}

func (api *API) addParameterInPipelineHandler() service.Handler {
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

		p, err := pipeline.LoadPipeline(api.mustDB(), key, pipelineName, false)
		if err != nil {
			return sdk.WrapError(err, "addParameterInPipelineHandler: Cannot load %s", pipelineName)
		}

		paramInProject, err := pipeline.CheckParameterInPipeline(api.mustDB(), p.ID, paramName)
		if err != nil {
			return sdk.WrapError(err, "addParameterInPipelineHandler: Cannot check if parameter %s is already in the pipeline %s", paramName, pipelineName)
		}
		if paramInProject {
			return sdk.WrapError(sdk.ErrParameterExists, "addParameterInPipelineHandler:Parameter %s is already in the pipeline %s", paramName, pipelineName)
		}

		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WrapError(err, "addParameterInPipelineHandler: Cannot start transaction")
		}
		defer tx.Rollback()

		if !paramInProject {
			if err := pipeline.InsertParameterInPipeline(tx, p.ID, &newParam); err != nil {
				return sdk.WrapError(err, "addParameterInPipelineHandler: Cannot add parameter %s in pipeline %s", paramName, pipelineName)
			}
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "addParameterInPipelineHandler: Cannot commit transaction")
		}

		event.PublishPipelineParameterAdd(key, pipelineName, newParam, getUser(ctx))

		p.Parameter, err = pipeline.GetAllParametersInPipeline(ctx, api.mustDB(), p.ID)
		if err != nil {
			return sdk.WrapError(err, "addParameterInPipelineHandler: Cannot get pipeline parameters")
		}

		return service.WriteJSON(w, p, http.StatusOK)
	}
}
