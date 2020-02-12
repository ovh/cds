package api

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/gorilla/mux"
	"gopkg.in/yaml.v2"

	"github.com/ovh/cds/engine/api/event"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/exportentities"
)

func (api *API) postPipelinePreviewHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		format := r.FormValue("format")

		// Get body
		data, errRead := ioutil.ReadAll(r.Body)
		if errRead != nil {
			return sdk.NewError(sdk.ErrWrongRequest, sdk.WrapError(errRead, "Unable to read body"))
		}

		// Compute format
		f, errF := exportentities.GetFormat(format)
		if errF != nil {
			return sdk.NewError(sdk.ErrWrongRequest, errF)
		}

		var payload exportentities.PipelineV1
		var errorParse error
		switch f {
		case exportentities.FormatJSON:
			errorParse = json.Unmarshal(data, &payload)
		case exportentities.FormatYAML:
			errorParse = yaml.Unmarshal(data, &payload)
		}
		if errorParse != nil {
			return sdk.NewError(sdk.ErrWrongRequest, errorParse)
		}

		pip, errP := payload.Pipeline()
		if errP != nil {
			return sdk.WrapError(errP, "Unable to parse pipeline")
		}

		return service.WriteJSON(w, pip, http.StatusOK)
	}
}

func (api *API) importPipelineHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars[permProjectKey]
		format := r.FormValue("format")
		forceUpdate := FormBool(r, "forceUpdate")

		// Load project
		proj, errp := project.Load(api.mustDB(), api.Cache, key,
			project.LoadOptions.Default,
			project.LoadOptions.WithGroups,
		)
		if errp != nil {
			return sdk.WrapError(errp, "Unable to load project %s", key)
		}

		// get request body
		data, errRead := ioutil.ReadAll(r.Body)
		if errRead != nil {
			return sdk.NewError(sdk.ErrWrongRequest, sdk.WrapError(errRead, "Unable to read body"))
		}

		payload, err := exportentities.ParsePipeline(format, data)
		if err != nil {
			return err
		}

		tx, errBegin := api.mustDB().Begin()
		if errBegin != nil {
			return sdk.WrapError(errBegin, "Cannot start transaction")
		}
		defer tx.Rollback() // nolint

		pip, allMsg, globalError := pipeline.ParseAndImport(ctx, tx, api.Cache, proj, payload, getAPIConsumer(ctx),
			pipeline.ImportOptions{Force: forceUpdate})
		msgListString := translate(r, allMsg)
		if globalError != nil {
			globalError = sdk.WrapError(globalError, "Unable to import pipeline")
			if sdk.ErrorIsUnknown(globalError) {
				return globalError
			}
			sdkErr := sdk.ExtractHTTPError(globalError, r.Header.Get("Accept-Language"))
			return service.WriteJSON(w, append(msgListString, sdkErr.Message), sdkErr.Status)
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "Cannot commit transaction")
		}

		event.PublishPipelineAdd(ctx, proj.Key, *pip, getAPIConsumer(ctx))

		return service.WriteJSON(w, msgListString, http.StatusOK)
	}
}

func (api *API) putImportPipelineHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars[permProjectKey]
		pipelineName := vars["pipelineKey"]
		format := r.FormValue("format")

		// Load project
		proj, errp := project.Load(api.mustDB(), api.Cache, key,
			project.LoadOptions.Default,
			project.LoadOptions.WithGroups,
		)
		if errp != nil {
			return sdk.WrapError(errp, "Unable to load project %s", key)
		}

		// Get body
		data, errRead := ioutil.ReadAll(r.Body)
		if errRead != nil {
			return sdk.NewError(sdk.ErrWrongRequest, sdk.WrapError(errRead, "Unable to read body"))
		}

		payload, err := exportentities.ParsePipeline(format, data)
		if err != nil {
			return err
		}

		tx, errBegin := api.mustDB().Begin()
		if errBegin != nil {
			return sdk.WrapError(errBegin, "Cannot start transaction")
		}

		defer func() {
			_ = tx.Rollback()
		}()

		pip, allMsg, globalError := pipeline.ParseAndImport(ctx, tx, api.Cache, proj, payload, getAPIConsumer(ctx), pipeline.ImportOptions{Force: true, PipelineName: pipelineName})
		msgListString := translate(r, allMsg)
		if globalError != nil {
			return sdk.WrapError(sdk.NewError(sdk.ErrInvalidPipeline, globalError), "unable to parse and import pipeline")
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "Cannot commit transaction")
		}

		event.PublishPipelineAdd(ctx, proj.Key, *pip, getAPIConsumer(ctx))

		return service.WriteJSON(w, msgListString, http.StatusOK)
	}
}
