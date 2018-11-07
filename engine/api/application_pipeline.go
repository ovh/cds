package api

import (
	"context"
	"io/ioutil"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/workflowv0"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
)

// Deprecated
func (api *API) attachPipelineToApplicationHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars["key"]
		appName := vars["permApplicationName"]
		pipelineName := vars["permPipelineKey"]

		pipeline, err := pipeline.LoadPipeline(api.mustDB(), key, pipelineName, true)
		if err != nil {
			return sdk.WrapError(sdk.ErrNotFound, "addPipelineInApplicationHandler> Cannot load pipeline %s: %s", appName, err)
		}

		app, err := application.LoadByName(api.mustDB(), api.Cache, key, appName, getUser(ctx), application.LoadOptions.Default)
		if err != nil {
			return sdk.WrapError(sdk.ErrNotFound, "addPipelineInApplicationHandler> Cannot load application %s: %s", appName, err)
		}

		if _, err := application.AttachPipeline(api.mustDB(), app.ID, pipeline.ID); err != nil {
			return sdk.WrapError(err, "Cannot attach pipeline %s to application %s", pipelineName, appName)
		}
		return service.WriteJSON(w, app, http.StatusOK)
	}
}

// Deprecated
func (api *API) attachPipelinesToApplicationHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars["key"]
		appName := vars["permApplicationName"]

		var pipelines []string
		if err := service.UnmarshalBody(r, &pipelines); err != nil {
			return err
		}

		app, err := application.LoadByName(api.mustDB(), api.Cache, key, appName, getUser(ctx), application.LoadOptions.Default)
		if err != nil {
			return sdk.WrapError(err, "attachPipelinesToApplicationHandler: Cannot load application %s", appName)
		}

		tx, errBegin := api.mustDB().Begin()
		if errBegin != nil {
			return sdk.WrapError(errBegin, "attachPipelinesToApplicationHandler: Cannot begin transaction")
		}

		for _, pipName := range pipelines {
			pip, err := pipeline.LoadPipeline(tx, key, pipName, true)
			if err != nil {
				return sdk.WrapError(err, "attachPipelinesToApplicationHandler: Cannot load pipeline %s", pipName)
			}

			id, errA := application.AttachPipeline(tx, app.ID, pip.ID)
			if errA != nil {
				return sdk.WrapError(errA, "attachPipelinesToApplicationHandler: Cannot attach pipeline %s to application %s", pipName, appName)
			}

			app.Pipelines = append(app.Pipelines, sdk.ApplicationPipeline{
				Pipeline: *pip,
				ID:       id,
			})
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "attachPipelinesToApplicationHandler: Cannot commit transaction")
		}

		var errW error
		app.Workflows, errW = workflowv0.LoadCDTree(api.mustDB(), api.Cache, key, app.Name, getUser(ctx), "", "", 0)
		if errW != nil {
			return sdk.WrapError(errW, "attachPipelinesToApplicationHandler: Cannot load application workflow")
		}

		return service.WriteJSON(w, app, http.StatusOK)
	}
}

func (api *API) updatePipelinesToApplicationHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars["key"]
		appName := vars["permApplicationName"]

		var appPipelines []sdk.ApplicationPipeline
		if err := service.UnmarshalBody(r, &appPipelines); err != nil {
			return err
		}

		app, err := application.LoadByName(api.mustDB(), api.Cache, key, appName, getUser(ctx), application.LoadOptions.Default)
		if err != nil {
			return sdk.WrapError(sdk.ErrApplicationNotFound, "updatePipelinesToApplicationHandler: Cannot load application %s", appName)
		}

		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WrapError(sdk.ErrUnknownError, "updatePipelinesToApplicationHandler: Cannot start transaction")
		}
		defer tx.Rollback()

		for _, appPip := range appPipelines {
			err = application.UpdatePipelineApplication(tx, api.Cache, app, appPip.Pipeline.ID, appPip.Parameters, getUser(ctx))
			if err != nil {
				return sdk.WrapError(sdk.ErrUnknownError, "updatePipelinesToApplicationHandler: Cannot update  application pipeline  %s/%s parameters", appName, appPip.Pipeline.Name)
			}
		}
		err = tx.Commit()
		if err != nil {
			return sdk.WrapError(sdk.ErrUnknownError, "updatePipelinesToApplicationHandler: Cannot commit transaction")
		}

		return service.WriteJSON(w, app, http.StatusOK)
	}
}

// DEPRECATED
func (api *API) updatePipelineToApplicationHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars["key"]
		appName := vars["permApplicationName"]
		pipelineName := vars["permPipelineKey"]

		pipeline, err := pipeline.LoadPipeline(api.mustDB(), key, pipelineName, false)
		if err != nil {
			return sdk.WrapError(sdk.ErrNotFound, "updatePipelineToApplicationHandler: Cannot load pipeline %s", appName)
		}

		app, err := application.LoadByName(api.mustDB(), api.Cache, key, appName, getUser(ctx))
		if err != nil {
			return sdk.WrapError(sdk.ErrNotFound, "updatePipelineToApplicationHandler: Cannot load application %s", appName)

		}

		// Get args in body
		data, err := ioutil.ReadAll(r.Body)
		if err != nil {
			return sdk.ErrWrongRequest
		}

		err = application.UpdatePipelineApplicationString(api.mustDB(), api.Cache, app, pipeline.ID, string(data), getUser(ctx))
		if err != nil {
			return sdk.WrapError(err, "updatePipelineToApplicationHandler: Cannot update application %s pipeline %s", appName, pipelineName)
		}

		return service.WriteJSON(w, app, http.StatusOK)
	}
}

func (api *API) getPipelinesInApplicationHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars["key"]
		appName := vars["permApplicationName"]

		pipelines, err := application.GetAllPipelines(api.mustDB(), key, appName)
		if err != nil {
			return sdk.WrapError(sdk.ErrNotFound, "getPipelinesInApplicationHandler: Cannot load pipelines for application %s", appName)
		}

		return service.WriteJSON(w, pipelines, http.StatusOK)
	}
}

func (api *API) removePipelineFromApplicationHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars["key"]
		appName := vars["permApplicationName"]
		pipelineName := vars["permPipelineKey"]

		a, errA := application.LoadByName(api.mustDB(), api.Cache, key, appName, getUser(ctx), application.LoadOptions.WithPipelines)
		if errA != nil {
			return sdk.WrapError(errA, "removePipelineFromApplicationHandler> Cannot load application")
		}

		tx, errB := api.mustDB().Begin()
		if errB != nil {
			return sdk.WrapError(errB, "removePipelineFromApplicationHandler> Cannot start tx")
		}
		defer tx.Rollback()

		if err := application.RemovePipeline(tx, key, appName, pipelineName); err != nil {
			return sdk.WrapError(err, "removePipelineFromApplicationHandler: Cannot detach pipeline %s from %s", pipelineName, appName)
		}

		// Remove pipeline from struct
		var indexPipeline int
		for i, appPip := range a.Pipelines {
			if appPip.Pipeline.Name == pipelineName {
				indexPipeline = i
				break
			}
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "Cannot commit tx")
		}

		var errW error
		a.Workflows, errW = workflowv0.LoadCDTree(api.mustDB(), api.Cache, key, a.Name, getUser(ctx), "", "", 0)
		if errW != nil {
			return sdk.WrapError(errW, "removePipelineFromApplicationHandler> Cannot load workflow")
		}

		a.Pipelines = append(a.Pipelines[:indexPipeline], a.Pipelines[indexPipeline+1:]...)

		return service.WriteJSON(w, a, http.StatusOK)
	}
}

// Deprecated
func (api *API) getUserNotificationApplicationPipelineHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		return nil
	}
}

// Deprecated
func (api *API) deleteUserNotificationApplicationPipelineHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		return nil
	}
}

// Deprecated
func (api *API) addNotificationsHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		return nil
	}
}

// Deprecated
func (api *API) updateUserNotificationApplicationPipelineHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		return nil
	}
}
