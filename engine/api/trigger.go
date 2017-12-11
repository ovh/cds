package api

import (
	"context"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/environment"
	"github.com/ovh/cds/engine/api/permission"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/trigger"
	"github.com/ovh/cds/engine/api/workflowv0"
	"github.com/ovh/cds/sdk"
)

func (api *API) addTriggerHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars["key"]

		// Unmarshal args
		var t sdk.PipelineTrigger
		if err := UnmarshalBody(r, &t); err != nil {
			return err
		}

		// load source ids
		if t.SrcApplication.ID == 0 {
			a, errSrcApp := application.LoadByName(api.mustDB(), api.Cache, key, t.SrcApplication.Name, getUser(ctx))
			if errSrcApp != nil {
				return sdk.WrapError(errSrcApp, "addTriggersHandler> cannot load src application")
			}
			t.SrcApplication.ID = a.ID
		}
		if !permission.AccessToApplication(key, t.SrcApplication.Name, getUser(ctx), permission.PermissionReadWriteExecute) {
			return sdk.WrapError(sdk.ErrForbidden, "addTriggersHandler> You don't have enought right on this application %s", t.SrcApplication.Name)
		}

		if t.SrcPipeline.ID == 0 {
			p, errSrcPip := pipeline.LoadPipeline(api.mustDB(), key, t.SrcPipeline.Name, false)
			if errSrcPip != nil {
				return sdk.WrapError(errSrcPip, "addTriggersHandler> cannot load src pipeline")
			}
			t.SrcPipeline.ID = p.ID
		}
		if !permission.AccessToPipeline(key, sdk.DefaultEnv.Name, t.SrcPipeline.Name, getUser(ctx), permission.PermissionReadWriteExecute) {
			return sdk.WrapError(sdk.ErrForbidden, "addTriggersHandler> You don't have enought right on this pipeline %s", t.SrcPipeline.Name)

		}

		if t.SrcEnvironment.ID == 0 && t.SrcEnvironment.Name != "" && t.SrcEnvironment.Name != sdk.DefaultEnv.Name {
			e, errSrcEnv := environment.LoadEnvironmentByName(api.mustDB(), key, t.SrcEnvironment.Name)
			if errSrcEnv != nil {
				return sdk.WrapError(errSrcEnv, "addTriggersHandler> cannot load src environment")
			}
			t.SrcEnvironment.ID = e.ID
		} else if t.SrcEnvironment.ID == 0 {
			t.SrcEnvironment = sdk.DefaultEnv
		}
		if !permission.AccessToEnvironment(key, t.SrcEnvironment.Name, getUser(ctx), permission.PermissionReadWriteExecute) {
			return sdk.WrapError(sdk.ErrForbidden, "addTriggersHandler> No enought right on this environment %s: ", t.SrcEnvironment.Name)

		}

		// load destination ids
		if t.DestApplication.ID == 0 {
			a, errDestApp := application.LoadByName(api.mustDB(), api.Cache, key, t.DestApplication.Name, getUser(ctx))
			if errDestApp != nil {
				return sdk.WrapError(errDestApp, "addTriggersHandler> cannot load dst application")
			}
			t.DestApplication.ID = a.ID
		}
		if !permission.AccessToApplication(key, t.DestApplication.Name, getUser(ctx), permission.PermissionReadWriteExecute) {
			return sdk.WrapError(sdk.ErrForbidden, "addTriggersHandler> You don't have enought right on this application %s", t.DestApplication.Name)
		}

		if t.DestPipeline.ID == 0 {
			p, errDestPip := pipeline.LoadPipeline(api.mustDB(), key, t.DestPipeline.Name, false)
			if errDestPip != nil {
				return sdk.WrapError(errDestPip, "addTriggersHandler> cannot load dst pipeline")
			}
			t.DestPipeline.ID = p.ID
		}
		if !permission.AccessToPipeline(key, sdk.DefaultEnv.Name, t.DestPipeline.Name, getUser(ctx), permission.PermissionReadWriteExecute) {
			return sdk.WrapError(sdk.ErrForbidden, "addTriggersHandler> You don't have enought right on this pipeline %s", t.DestPipeline.Name)

		}

		if t.DestEnvironment.ID == 0 && t.DestEnvironment.Name != "" && t.DestEnvironment.Name != sdk.DefaultEnv.Name {
			e, errDestEnv := environment.LoadEnvironmentByName(api.mustDB(), key, t.DestEnvironment.Name)
			if errDestEnv != nil {
				return sdk.WrapError(errDestEnv, "addTriggersHandler> cannot load dst environment")
			}
			t.DestEnvironment.ID = e.ID
		} else if t.DestEnvironment.ID == 0 {
			t.DestEnvironment = sdk.DefaultEnv
		}

		if !permission.AccessToEnvironment(key, t.DestEnvironment.Name, getUser(ctx), permission.PermissionReadWriteExecute) {
			return sdk.WrapError(sdk.ErrForbidden, "addTriggersHandler> No enought right on this environment %s: ", t.DestEnvironment.Name)

		}

		tx, errBegin := api.mustDB().Begin()
		if errBegin != nil {
			return errBegin

		}
		defer tx.Rollback()

		if err := trigger.InsertTrigger(tx, &t); err != nil {
			return sdk.WrapError(err, "addTriggerHandler> cannot insert trigger")

		}

		// Update src application
		if err := application.UpdateLastModified(tx, api.Cache, &t.SrcApplication, getUser(ctx)); err != nil {
			return sdk.WrapError(err, "addTriggerHandler> cannot update loast modified date on src application")
		}

		if err := tx.Commit(); err != nil {
			return err
		}

		var errWorkflow error
		t.SrcApplication.Workflows, errWorkflow = workflowv0.LoadCDTree(api.mustDB(), api.Cache, key, t.SrcApplication.Name, getUser(ctx), "", "", 0)
		if errWorkflow != nil {
			return sdk.WrapError(errWorkflow, "addTriggerHandler> cannot load updated workflow")
		}

		return WriteJSON(w, r, t.SrcApplication, http.StatusOK)
	}
}

func (api *API) getTriggerHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		striggerID := vars["id"]

		triggerID, errParse := strconv.ParseInt(striggerID, 10, 64)
		if errParse != nil {
			return sdk.WrapError(sdk.ErrInvalidID, "getTriggerHandler> TriggerId %s should be an int", striggerID)
		}

		t, errTrig := trigger.LoadTrigger(api.mustDB(), triggerID)
		if errTrig != nil {
			return sdk.WrapError(errTrig, "getTriggerHandler> Cannot load trigger %d", triggerID)
		}

		return WriteJSON(w, r, t, http.StatusOK)
	}
}

func (api *API) getTriggersHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars["key"]
		app := vars["permApplicationName"]
		pip := vars["permPipelineKey"]

		if err := r.ParseForm(); err != nil {
			return sdk.WrapError(sdk.ErrUnknownError, "getTriggersHandler> Cannot parse form")

		}
		env := r.Form.Get("env")

		a, errApp := application.LoadByName(api.mustDB(), api.Cache, key, app, getUser(ctx))
		if errApp != nil {
			return sdk.WrapError(errApp, "getTriggersHandler> cannot load application")
		}

		p, errPip := pipeline.LoadPipeline(api.mustDB(), key, pip, false)
		if errPip != nil {
			return sdk.WrapError(errPip, "getTriggersHandler> cannot load pipeline")
		}

		var envID int64
		if env != "" && env != sdk.DefaultEnv.Name {
			e, errEnv := environment.LoadEnvironmentByName(api.mustDB(), key, env)
			if errEnv != nil {
				return sdk.WrapError(errEnv, "getTriggersHandler> cannot load environment")
			}
			envID = e.ID

			if !permission.AccessToEnvironment(key, e.Name, getUser(ctx), permission.PermissionRead) {
				return sdk.WrapError(sdk.ErrForbidden, "getTriggersHandler> No enought right on this environment %s: ", e.Name)

			}
		}

		triggers, errTri := trigger.LoadTriggers(api.mustDB(), a.ID, p.ID, envID)
		if errTri != nil {
			return sdk.WrapError(errTri, "getTriggersHandler> cannot load triggers")
		}

		return WriteJSON(w, r, triggers, http.StatusOK)
	}
}

func (api *API) deleteTriggerHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		projectKey := vars["key"]
		triggerIDS := vars["id"]

		triggerID, errParse := strconv.ParseInt(triggerIDS, 10, 64)
		if errParse != nil {
			return sdk.WrapError(sdk.ErrInvalidID, "deleteTriggerHandler> invalid id (%s)", errParse)
		}

		t, errTrigger := trigger.LoadTrigger(api.mustDB(), triggerID)
		if errTrigger != nil {
			return sdk.WrapError(errTrigger, "deleteTriggerHandler> Cannot load trigger")
		}

		tx, errBegin := api.mustDB().Begin()
		if errBegin != nil {
			return sdk.WrapError(errBegin, "deleteTriggerHandler> Cannot start transaction")
		}
		defer tx.Rollback()

		if err := trigger.DeleteTrigger(tx, triggerID); err != nil {
			return sdk.WrapError(err, "deleteTriggerHandler> cannot delete trigger")
		}

		if err := application.UpdateLastModified(tx, api.Cache, &t.SrcApplication, getUser(ctx)); err != nil {
			return sdk.WrapError(err, "deleteTriggerHandler> cannot update src application last modified date")

		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "deleteTriggerHandler> cannot commit transaction")

		}

		var errWorkflow error
		t.SrcApplication.Workflows, errWorkflow = workflowv0.LoadCDTree(api.mustDB(), api.Cache, projectKey, t.SrcApplication.Name, getUser(ctx), "", "", 0)
		if errWorkflow != nil {
			return sdk.WrapError(errWorkflow, "deleteTriggerHandler> cannot load updated workflow")
		}

		return WriteJSON(w, r, t.SrcApplication, http.StatusOK)
	}
}

func (api *API) updateTriggerHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		projectKey := vars["key"]
		triggerIDS := vars["id"]

		triggerID, errParse := strconv.ParseInt(triggerIDS, 10, 64)
		if errParse != nil {
			return sdk.WrapError(sdk.ErrInvalidID, "updateTriggerHandler> invalid id (%s)", errParse)

		}

		var t sdk.PipelineTrigger
		if err := UnmarshalBody(r, &t); err != nil {
			return err
		}

		if t.SrcApplication.ID == 0 || t.DestApplication.ID == 0 ||
			t.SrcPipeline.ID == 0 || t.DestPipeline.ID == 0 {
			return sdk.WrapError(sdk.ErrWrongRequest, "updateTriggerHandler> IDs should not be zero")

		}

		tx, errBegin := api.mustDB().Begin()
		if errBegin != nil {
			return sdk.WrapError(errBegin, "updateTriggerHandler> cannot start transaction")

		}
		defer tx.Rollback()

		t.ID = triggerID
		if err := trigger.UpdateTrigger(tx, &t); err != nil {
			return sdk.WrapError(err, "updateTriggerHandler> cannot update trigger")
		}

		if err := application.UpdateLastModified(tx, api.Cache, &t.SrcApplication, getUser(ctx)); err != nil {
			return sdk.WrapError(err, "updateTriggerHandler> cannot update src application last modified date")
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "updateTriggerHandler> cannot commit transaction")
		}

		var errWorkflow error
		t.SrcApplication.Workflows, errWorkflow = workflowv0.LoadCDTree(api.mustDB(), api.Cache, projectKey, t.SrcApplication.Name, getUser(ctx), "", "", 0)
		if errWorkflow != nil {
			return sdk.WrapError(errWorkflow, "updateTriggerHandler> cannot load updated workflow")
		}

		return WriteJSON(w, r, t.SrcApplication, http.StatusOK)
	}
}

func (api *API) getTriggersAsSourceHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars["key"]
		app := vars["permApplicationName"]
		pip := vars["permPipelineKey"]

		if err := r.ParseForm(); err != nil {
			return sdk.WrapError(sdk.ErrWrongRequest, "getTriggersAsSourceHandler> Cannot parse form")
		}
		env := r.Form.Get("env")

		a, errApp := application.LoadByName(api.mustDB(), api.Cache, key, app, getUser(ctx))
		if errApp != nil {
			return sdk.WrapError(errApp, "getTriggersAsSourceHandler> cannot load application")
		}

		p, errPip := pipeline.LoadPipeline(api.mustDB(), key, pip, false)
		if errPip != nil {
			return sdk.WrapError(errPip, "getTriggersAsSourceHandler> cannot load pipeline")
		}

		var envID int64
		if env != "" && env != sdk.DefaultEnv.Name {
			e, errEnv := environment.LoadEnvironmentByName(api.mustDB(), key, env)
			if errEnv != nil {
				return sdk.WrapError(errEnv, "getTriggersAsSourceHandler> cannot load environment")
			}
			envID = e.ID

			if !permission.AccessToEnvironment(key, e.Name, getUser(ctx), permission.PermissionRead) {
				return sdk.WrapError(sdk.ErrForbidden, "getTriggersAsSourceHandler> No enought right on this environment %s: ", e.Name)
			}
		}

		triggers, errTri := trigger.LoadTriggersAsSource(api.mustDB(), a.ID, p.ID, envID)
		if errTri != nil {
			return sdk.WrapError(errTri, "getTriggersAsSourceHandler> cannot load triggers")
		}

		return WriteJSON(w, r, triggers, http.StatusOK)
	}
}
