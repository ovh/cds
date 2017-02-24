package main

import (
	"net/http"
	"strconv"

	"github.com/go-gorp/gorp"
	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/context"
	"github.com/ovh/cds/engine/api/environment"
	"github.com/ovh/cds/engine/api/permission"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/trigger"
	"github.com/ovh/cds/engine/log"
	"github.com/ovh/cds/sdk"
)

func addTriggerHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	vars := mux.Vars(r)
	project := vars["key"]

	// Unmarshal args
	var t sdk.PipelineTrigger
	if err := UnmarshalBody(r, &t); err != nil {
		return err
	}

	// load source ids
	if t.SrcApplication.ID == 0 {
		a, errSrcApp := application.LoadByName(db, project, t.SrcApplication.Name, c.User)
		if errSrcApp != nil {
			log.Warning("addTriggersHandler> cannot load src application: %s\n", errSrcApp)
			return errSrcApp
		}
		t.SrcApplication.ID = a.ID
	}
	if !permission.AccessToApplication(t.SrcApplication.ID, c.User, permission.PermissionReadWriteExecute) {
		log.Warning("addTriggersHandler> You don't have enought right on this application %s", t.SrcApplication.Name)
		return sdk.ErrForbidden
	}

	if t.SrcPipeline.ID == 0 {
		p, errSrcPip := pipeline.LoadPipeline(db, project, t.SrcPipeline.Name, false)
		if errSrcPip != nil {
			log.Warning("addTriggersHandler> cannot load src pipeline: %s\n", errSrcPip)
			return errSrcPip
		}
		t.SrcPipeline.ID = p.ID
	}
	if !permission.AccessToPipeline(sdk.DefaultEnv.ID, t.SrcPipeline.ID, c.User, permission.PermissionReadWriteExecute) {
		log.Warning("addTriggersHandler> You don't have enought right on this pipeline %s", t.SrcPipeline.Name)
		return sdk.ErrForbidden

	}

	if t.SrcEnvironment.ID == 0 && t.SrcEnvironment.Name != "" && t.SrcEnvironment.Name != sdk.DefaultEnv.Name {
		e, errSrcEnv := environment.LoadEnvironmentByName(db, project, t.SrcEnvironment.Name)
		if errSrcEnv != nil {
			log.Warning("addTriggersHandler> cannot load src environment: %s\n", errSrcEnv)
			return errSrcEnv
		}
		t.SrcEnvironment.ID = e.ID
	} else if t.SrcEnvironment.ID == 0 {
		t.SrcEnvironment = sdk.DefaultEnv
	}
	if !permission.AccessToEnvironment(t.SrcEnvironment.ID, c.User, permission.PermissionReadWriteExecute) {
		log.Warning("addTriggersHandler> No enought right on this environment %s: \n", t.SrcEnvironment.Name)
		return sdk.ErrForbidden

	}

	// load destination ids
	if t.DestApplication.ID == 0 {
		a, errDestApp := application.LoadByName(db, project, t.DestApplication.Name, c.User)
		if errDestApp != nil {
			log.Warning("addTriggersHandler> cannot load dst application: %s\n", errDestApp)
			return errDestApp
		}
		t.DestApplication.ID = a.ID
	}
	if !permission.AccessToApplication(t.DestApplication.ID, c.User, permission.PermissionReadWriteExecute) {
		log.Warning("addTriggersHandler> You don't have enought right on this application %s", t.DestApplication.Name)
		return sdk.ErrForbidden
	}

	if t.DestPipeline.ID == 0 {
		p, errDestPip := pipeline.LoadPipeline(db, project, t.DestPipeline.Name, false)
		if errDestPip != nil {
			log.Warning("addTriggersHandler> cannot load dst pipeline: %s\n", errDestPip)
			return errDestPip
		}
		t.DestPipeline.ID = p.ID
	}
	if !permission.AccessToPipeline(sdk.DefaultEnv.ID, t.DestPipeline.ID, c.User, permission.PermissionReadWriteExecute) {
		log.Warning("addTriggersHandler> You don't have enought right on this pipeline %s", t.DestPipeline.Name)
		return sdk.ErrForbidden

	}

	if t.DestEnvironment.ID == 0 && t.DestEnvironment.Name != "" && t.DestEnvironment.Name != sdk.DefaultEnv.Name {
		e, errDestEnv := environment.LoadEnvironmentByName(db, project, t.DestEnvironment.Name)
		if errDestEnv != nil {
			log.Warning("addTriggersHandler> cannot load dst environment: %s\n", errDestEnv)
			return errDestEnv
		}
		t.DestEnvironment.ID = e.ID
	} else if t.DestEnvironment.ID == 0 {
		t.DestEnvironment = sdk.DefaultEnv
	}

	if !permission.AccessToEnvironment(t.DestEnvironment.ID, c.User, permission.PermissionReadWriteExecute) {
		log.Warning("addTriggersHandler> No enought right on this environment %s: \n", t.DestEnvironment.Name)
		return sdk.ErrForbidden

	}

	tx, errBegin := db.Begin()
	if errBegin != nil {
		return errBegin

	}
	defer tx.Rollback()

	if err := trigger.InsertTrigger(tx, &t); err != nil {
		log.Warning("addTriggerHandler> cannot insert trigger: %s\n", err)
		return err

	}

	// Update src application
	if err := application.UpdateLastModified(tx, &t.SrcApplication, c.User); err != nil {
		log.Warning("addTriggerHandler> cannot update loast modified date on src application: %s\n", err)
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	var errWorkflow error
	t.SrcApplication.Workflows, errWorkflow = application.LoadCDTree(db, project, t.SrcApplication.Name, c.User)
	if errWorkflow != nil {
		log.Warning("addTriggerHandler> cannot load updated workflow: %s\n", errWorkflow)
		return errWorkflow
	}

	return WriteJSON(w, r, t.SrcApplication, http.StatusOK)
}

func getTriggerHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	vars := mux.Vars(r)
	striggerID := vars["id"]

	triggerID, errParse := strconv.ParseInt(striggerID, 10, 64)
	if errParse != nil {
		log.Warning("getTriggerHandler> TriggerId %s should be an int: %s\n", striggerID, errParse)
		return sdk.ErrInvalidID
	}

	t, errTrig := trigger.LoadTrigger(db, triggerID)
	if errTrig != nil {
		log.Warning("getTriggerHandler> Cannot load trigger %d: %s\n", triggerID, errTrig)
		return errTrig
	}

	return WriteJSON(w, r, t, http.StatusOK)
}

func getTriggersHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	vars := mux.Vars(r)
	project := vars["key"]
	app := vars["permApplicationName"]
	pip := vars["permPipelineKey"]

	if err := r.ParseForm(); err != nil {
		log.Warning("getTriggersHandler> Cannot parse form: %s\n", err)
		return sdk.ErrUnknownError

	}
	env := r.Form.Get("env")

	a, errApp := application.LoadByName(db, project, app, c.User)
	if errApp != nil {
		log.Warning("getTriggersHandler> cannot load application: %s\n", errApp)
		return errApp
	}

	p, errPip := pipeline.LoadPipeline(db, project, pip, false)
	if errPip != nil {
		log.Warning("getTriggersHandler> cannot load pipeline: %s\n", errPip)
		return errPip
	}

	var envID int64
	if env != "" && env != sdk.DefaultEnv.Name {
		e, errEnv := environment.LoadEnvironmentByName(db, project, env)
		if errEnv != nil {
			log.Warning("getTriggersHandler> cannot load environment: %s\n", errEnv)
			return errEnv
		}
		envID = e.ID

		if !permission.AccessToEnvironment(e.ID, c.User, permission.PermissionRead) {
			log.Warning("getTriggersHandler> No enought right on this environment %s: \n", e.Name)
			return sdk.ErrForbidden

		}
	}

	triggers, errTri := trigger.LoadTriggers(db, a.ID, p.ID, envID)
	if errTri != nil {
		log.Warning("getTriggersHandler> cannot load triggers: %s\n", errTri)
		return errTri
	}

	return WriteJSON(w, r, triggers, http.StatusOK)
}

func deleteTriggerHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	vars := mux.Vars(r)
	projectKey := vars["key"]
	triggerIDS := vars["id"]

	triggerID, errParse := strconv.ParseInt(triggerIDS, 10, 64)
	if errParse != nil {
		log.Warning("deleteTriggerHandler> invalid id (%s)\n", errParse)
		return sdk.ErrInvalidID
	}

	t, errTrigger := trigger.LoadTrigger(db, triggerID)
	if errTrigger != nil {
		log.Warning("deleteTriggerHandler> Cannot load trigger: %s\n", errTrigger)
		return errTrigger
	}

	tx, errBegin := db.Begin()
	if errBegin != nil {
		log.Warning("deleteTriggerHandler> Cannot start transaction: %s\n", errBegin)
		return errBegin
	}
	defer tx.Rollback()

	if err := trigger.DeleteTrigger(tx, triggerID); err != nil {
		log.Warning("deleteTriggerHandler> cannot delete trigger: %s\n", err)
		return err
	}

	if err := application.UpdateLastModified(tx, &t.SrcApplication, c.User); err != nil {
		log.Warning("deleteTriggerHandler> cannot update src application last modified date: %s\n", err)
		return err

	}

	if err := tx.Commit(); err != nil {
		log.Warning("deleteTriggerHandler> cannot commit transaction: %s\n", err)
		return err

	}

	var errWorkflow error
	t.SrcApplication.Workflows, errWorkflow = application.LoadCDTree(db, projectKey, t.SrcApplication.Name, c.User)
	if errWorkflow != nil {
		log.Warning("deleteTriggerHandler> cannot load updated workflow: %s\n", errWorkflow)
		return errWorkflow
	}

	return WriteJSON(w, r, t.SrcApplication, http.StatusOK)
}

func updateTriggerHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	vars := mux.Vars(r)
	projectKey := vars["key"]
	triggerIDS := vars["id"]

	triggerID, errParse := strconv.ParseInt(triggerIDS, 10, 64)
	if errParse != nil {
		log.Warning("updateTriggerHandler> invalid id (%s)\n", errParse)
		return sdk.ErrInvalidID

	}

	var t sdk.PipelineTrigger
	if err := UnmarshalBody(r, &t); err != nil {
		return err
	}

	if t.SrcApplication.ID == 0 || t.DestApplication.ID == 0 ||
		t.SrcPipeline.ID == 0 || t.DestPipeline.ID == 0 {
		log.Warning("updateTriggerHandler> IDs should not be zero\n")
		return sdk.ErrWrongRequest

	}

	tx, errBegin := db.Begin()
	if errBegin != nil {
		log.Warning("updateTriggerHandler> cannot start transaction: %s\n", errBegin)
		return errBegin

	}
	defer tx.Rollback()

	t.ID = triggerID
	if err := trigger.UpdateTrigger(tx, &t); err != nil {
		log.Warning("updateTriggerHandler> cannot update trigger: %s\n", err)
		return err
	}

	if err := application.UpdateLastModified(tx, &t.SrcApplication, c.User); err != nil {
		log.Warning("updateTriggerHandler> cannot update src application last modified date: %s\n", err)
		return err
	}

	if err := tx.Commit(); err != nil {
		log.Warning("updateTriggerHandler> cannot commit transaction: %s\n", err)
		return err
	}

	var errWorkflow error
	t.SrcApplication.Workflows, errWorkflow = application.LoadCDTree(db, projectKey, t.SrcApplication.Name, c.User)
	if errWorkflow != nil {
		log.Warning("updateTriggerHandler> cannot load updated workflow: %s\n", errWorkflow)
		return errWorkflow
	}

	return WriteJSON(w, r, t.SrcApplication, http.StatusOK)
}

func getTriggersAsSourceHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	vars := mux.Vars(r)
	project := vars["key"]
	app := vars["permApplicationName"]
	pip := vars["permPipelineKey"]

	if err := r.ParseForm(); err != nil {
		log.Warning("getTriggersAsSourceHandler> Cannot parse form: %s\n", err)
		return sdk.ErrWrongRequest
	}
	env := r.Form.Get("env")

	a, errApp := application.LoadByName(db, project, app, c.User)
	if errApp != nil {
		log.Warning("getTriggersAsSourceHandler> cannot load application: %s\n", errApp)
		return errApp
	}

	p, errPip := pipeline.LoadPipeline(db, project, pip, false)
	if errPip != nil {
		log.Warning("getTriggersAsSourceHandler> cannot load pipeline: %s\n", errPip)
		return errPip
	}

	var envID int64
	if env != "" && env != sdk.DefaultEnv.Name {
		e, errEnv := environment.LoadEnvironmentByName(db, project, env)
		if errEnv != nil {
			log.Warning("getTriggersAsSourceHandler> cannot load environment: %s\n", errEnv)
			return errEnv
		}
		envID = e.ID

		if !permission.AccessToEnvironment(e.ID, c.User, permission.PermissionRead) {
			log.Warning("getTriggersAsSourceHandler> No enought right on this environment %s: \n", e.Name)
			return sdk.ErrForbidden
		}
	}

	triggers, errTri := trigger.LoadTriggersAsSource(db, a.ID, p.ID, envID)
	if errTri != nil {
		log.Warning("getTriggersAsSourceHandler> cannot load triggers: %s\n", errTri)
		return errTri
	}

	return WriteJSON(w, r, triggers, http.StatusOK)
}
