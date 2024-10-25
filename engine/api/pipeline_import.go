package api

import (
	"context"
	"io"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/event"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/exportentities"
)

func (api *API) postPipelinePreviewHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			return sdk.NewErrorWithStack(err, sdk.NewErrorFrom(sdk.ErrWrongRequest, "unable to read body"))
		}

		contentType := r.Header.Get("Content-Type")
		if contentType == "" {
			contentType = http.DetectContentType(body)
		}
		format, err := exportentities.GetFormatFromContentType(contentType)
		if err != nil {
			return err
		}

		var data exportentities.PipelineV1
		if err := exportentities.Unmarshal(body, format, &data); err != nil {
			return err
		}

		pip, err := data.Pipeline()
		if err != nil {
			return sdk.WrapError(err, "unable to parse pipeline")
		}

		return service.WriteJSON(w, pip, http.StatusOK)
	}
}

func (api *API) importPipelineHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars[permProjectKey]
		force := service.FormBool(r, "force")

		body, err := io.ReadAll(r.Body)
		if err != nil {
			return sdk.NewError(sdk.ErrWrongRequest, sdk.WrapError(err, "unable to read body"))
		}

		contentType := r.Header.Get("Content-Type")
		if contentType == "" {
			contentType = http.DetectContentType(body)
		}
		format, err := exportentities.GetFormatFromContentType(contentType)
		if err != nil {
			return err
		}

		// Load project
		proj, err := project.Load(ctx, api.mustDB(), key,
			project.LoadOptions.Default,
			project.LoadOptions.WithGroups,
		)
		if err != nil {
			return sdk.WrapError(err, "unable to load project %s", key)
		}

		data, err := exportentities.ParsePipeline(format, body)
		if err != nil {
			return err
		}

		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WrapError(err, "cannot start transaction")
		}
		defer tx.Rollback() // nolint

		pip, allMsg, globalError := pipeline.ParseAndImport(ctx, tx, api.Cache, *proj, data, getUserConsumer(ctx),
			pipeline.ImportOptions{Force: force})
		msgListString := translate(allMsg)
		if globalError != nil {
			globalError = sdk.WrapError(globalError, "unable to import pipeline")
			if sdk.ErrorIsUnknown(globalError) {
				return globalError
			}
			sdkErr := sdk.ExtractHTTPError(globalError)
			return service.WriteJSON(w, append(msgListString, sdkErr.Error()), sdkErr.Status)
		}

		if err := tx.Commit(); err != nil {
			return sdk.WithStack(err)
		}

		event.PublishPipelineAdd(ctx, proj.Key, *pip, getUserConsumer(ctx))

		return service.WriteJSON(w, msgListString, http.StatusOK)
	}
}

func (api *API) putImportPipelineHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars[permProjectKey]
		pipelineName := vars["pipelineKey"]

		body, err := io.ReadAll(r.Body)
		if err != nil {
			return sdk.NewErrorWithStack(err, sdk.NewErrorFrom(sdk.ErrWrongRequest, "unable to read body"))
		}

		contentType := r.Header.Get("Content-Type")
		if contentType == "" {
			contentType = http.DetectContentType(body)
		}
		format, err := exportentities.GetFormatFromContentType(contentType)
		if err != nil {
			return err
		}

		proj, err := project.Load(ctx, api.mustDB(), key,
			project.LoadOptions.Default,
			project.LoadOptions.WithGroups,
		)
		if err != nil {
			return sdk.WrapError(err, "unable to load project %s", key)
		}

		data, err := exportentities.ParsePipeline(format, body)
		if err != nil {
			return err
		}

		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WrapError(err, "cannot start transaction")
		}
		defer tx.Rollback() // nolint

		pip, allMsg, err := pipeline.ParseAndImport(ctx, tx, api.Cache, *proj, data, getUserConsumer(ctx), pipeline.ImportOptions{Force: true, PipelineName: pipelineName})
		msgListString := translate(allMsg)
		if err != nil {
			return sdk.ErrorWithFallback(err, sdk.ErrWrongRequest, "unable to import pipeline")
		}

		if err := tx.Commit(); err != nil {
			return sdk.WithStack(err)
		}

		event.PublishPipelineAdd(ctx, proj.Key, *pip, getUserConsumer(ctx))

		return service.WriteJSON(w, msgListString, http.StatusOK)
	}
}
