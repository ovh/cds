package api

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/gorilla/mux"
	"gopkg.in/yaml.v2"

	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/exportentities"
)

func (api *API) postPipelinePreviewHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		format := r.FormValue("format")

		// Get body
		data, errRead := ioutil.ReadAll(r.Body)
		if errRead != nil {
			return sdk.WrapError(sdk.ErrWrongRequest, "postPipelinePreviewHandler> Unable to read body")
		}

		// Compute format
		f, errF := exportentities.GetFormat(format)
		if errF != nil {
			return sdk.WrapError(sdk.ErrWrongRequest, "postPipelinePreviewHandler> Unable to get format : %s", errF)
		}

		rawPayload := map[string]interface{}{}
		var errorParse error
		switch f {
		case exportentities.FormatJSON:
			errorParse = json.Unmarshal(data, &rawPayload)
		case exportentities.FormatYAML:
			errorParse = yaml.Unmarshal(data, &rawPayload)
		}

		if errorParse != nil {
			return sdk.NewError(sdk.ErrWrongRequest, errorParse)
		}

		//Parse the data once to retrieve the version
		var pipelineV1Format bool
		if v, ok := rawPayload["version"]; ok {
			version, okStr := v.(string)
			if !okStr {
				return sdk.WrapError(sdk.ErrWrongRequest, "postPipelinePreviewHandler> Cannot parse pipeline version")
			}
			if version == exportentities.PipelineVersion1 {
				pipelineV1Format = true
			}
		}

		//Depending on the version, we will use different struct
		type pipeliner interface {
			Pipeline() (*sdk.Pipeline, error)
		}

		var payload pipeliner
		// Parse the pipeline
		if pipelineV1Format {
			payload = &exportentities.PipelineV1{}
		} else {
			payload = &exportentities.Pipeline{}
		}

		switch f {
		case exportentities.FormatJSON:
			errorParse = json.Unmarshal(data, payload)
		case exportentities.FormatYAML:
			errorParse = yaml.Unmarshal(data, payload)
		}

		if errorParse != nil {
			return sdk.WrapError(sdk.ErrWrongRequest, "postPipelinePreviewHandler> Cannot parsing: %s", errorParse)
		}

		pip, errP := payload.Pipeline()
		if errP != nil {
			return sdk.WrapError(errP, "postPipelinePreviewHandler> Unable to parse pipeline")
		}

		return WriteJSON(w, pip, http.StatusOK)
	}
}

func (api *API) importPipelineHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars["permProjectKey"]
		format := r.FormValue("format")
		forceUpdate := FormBool(r, "forceUpdate")

		// Load project
		proj, errp := project.Load(api.mustDB(), api.Cache, key, getUser(ctx), project.LoadOptions.Default)
		if errp != nil {
			return sdk.WrapError(errp, "importPipelineHandler> Unable to load project %s", key)
		}

		if err := group.LoadGroupByProject(api.mustDB(), proj); err != nil {
			return sdk.WrapError(err, "importPipelineHandler> Unable to load project permissions %s", key)
		}

		// Get body
		data, errRead := ioutil.ReadAll(r.Body)
		if errRead != nil {
			return sdk.WrapError(sdk.ErrWrongRequest, "importPipelineHandler> Unable to read body")
		}

		// Compute format
		f, errF := exportentities.GetFormat(format)
		if errF != nil {
			return sdk.WrapError(sdk.ErrWrongRequest, "importPipelineHandler> Unable to get format : %s", errF)
		}

		rawPayload := map[string]interface{}{}
		var errorParse error
		switch f {
		case exportentities.FormatJSON:
			errorParse = json.Unmarshal(data, &rawPayload)
		case exportentities.FormatYAML:
			errorParse = yaml.Unmarshal(data, &rawPayload)
		}

		if errorParse != nil {
			return sdk.NewError(sdk.ErrWrongRequest, errorParse)
		}

		//Parse the data once to retrieve the version
		var pipelineV1Format bool
		if v, ok := rawPayload["version"]; ok {
			if v.(string) == exportentities.PipelineVersion1 {
				pipelineV1Format = true
			}
		}

		//Depending on the version, we will use different struct
		type pipeliner interface {
			Pipeline() (*sdk.Pipeline, error)
		}

		var payload pipeliner
		// Parse the pipeline
		if pipelineV1Format {
			payload = &exportentities.PipelineV1{}
		} else {
			payload = &exportentities.Pipeline{}
		}

		switch f {
		case exportentities.FormatJSON:
			errorParse = json.Unmarshal(data, payload)
		case exportentities.FormatYAML:
			errorParse = yaml.Unmarshal(data, payload)
		}

		if errorParse != nil {
			return sdk.WrapError(sdk.ErrWrongRequest, "importPipelineHandler> Cannot parsing: %s", errorParse)
		}

		tx, errBegin := api.mustDB().Begin()
		if errBegin != nil {
			return sdk.WrapError(errBegin, "importPipelineHandler: Cannot start transaction")
		}

		defer tx.Rollback()

		_, allMsg, globalError := pipeline.ParseAndImport(tx, api.Cache, proj, payload, forceUpdate, getUser(ctx))
		msgListString := translate(r, allMsg)

		if globalError != nil {
			myError, ok := globalError.(sdk.Error)
			if ok {
				return WriteJSON(w, msgListString, myError.Status)
			}
			return sdk.WrapError(globalError, "importPipelineHandler> Unable import pipeline")
		}

		if err := project.UpdateLastModified(tx, api.Cache, getUser(ctx), proj, sdk.ProjectPipelineLastModificationType); err != nil {
			return sdk.WrapError(err, "importPipelineHandler> Unable to update project")
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "importPipelineHandler> Cannot commit transaction")
		}

		return WriteJSON(w, msgListString, http.StatusOK)
	}
}
