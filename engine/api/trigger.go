package main

import (
	"database/sql"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strconv"

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

func addTriggerHandler(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {
	vars := mux.Vars(r)
	project := vars["key"]

	// Get args in body
	data, errRead := ioutil.ReadAll(r.Body)
	if errRead != nil {
		log.Warning("addTriggerHandler> cannot read body: %s\n", errRead)
		WriteError(w, r, sdk.ErrWrongRequest)
		return
	}

	// Unmarshal args
	var t sdk.PipelineTrigger
	if err := json.Unmarshal(data, &t); err != nil {
		log.Warning("addTriggerHandler> cannot unmarshal body:  %s\n", err)
		WriteError(w, r, sdk.ErrWrongRequest)
		return
	}

	// load source ids
	if t.SrcApplication.ID == 0 {
		a, errSrcApp := application.LoadApplicationByName(db, project, t.SrcApplication.Name)
		if errSrcApp != nil {
			log.Warning("addTriggersHandler> cannot load src application: %s\n", errSrcApp)
			WriteError(w, r, errSrcApp)
			return
		}
		t.SrcApplication.ID = a.ID
	}
	if !permission.AccessToApplication(t.SrcApplication.ID, c.User, permission.PermissionReadWriteExecute) {
		log.Warning("addTriggersHandler> You don't have enought right on this application %s", t.SrcApplication.Name)
		WriteError(w, r, sdk.ErrForbidden)
		return
	}

	if t.SrcPipeline.ID == 0 {
		p, errSrcPip := pipeline.LoadPipeline(db, project, t.SrcPipeline.Name, false)
		if errSrcPip != nil {
			log.Warning("addTriggersHandler> cannot load src pipeline: %s\n", errSrcPip)
			WriteError(w, r, errSrcPip)
			return
		}
		t.SrcPipeline.ID = p.ID
	}
	if !permission.AccessToPipeline(sdk.DefaultEnv.ID, t.SrcPipeline.ID, c.User, permission.PermissionReadWriteExecute) {
		log.Warning("addTriggersHandler> You don't have enought right on this pipeline %s", t.SrcPipeline.Name)
		WriteError(w, r, sdk.ErrForbidden)
		return
	}

	if t.SrcEnvironment.ID == 0 && t.SrcEnvironment.Name != "" && t.SrcEnvironment.Name != sdk.DefaultEnv.Name {
		e, errSrcEnv := environment.LoadEnvironmentByName(db, project, t.SrcEnvironment.Name)
		if errSrcEnv != nil {
			log.Warning("addTriggersHandler> cannot load src environment: %s\n", errSrcEnv)
			WriteError(w, r, errSrcEnv)
			return
		}
		t.SrcEnvironment.ID = e.ID
	} else if t.SrcEnvironment.ID == 0 {
		t.SrcEnvironment = sdk.DefaultEnv
	}
	if t.SrcEnvironment.ID != sdk.DefaultEnv.ID && !permission.AccessToEnvironment(t.SrcEnvironment.ID, c.User, permission.PermissionReadWriteExecute) {
		log.Warning("addTriggersHandler> No enought right on this environment %s: \n", t.SrcEnvironment.Name)
		WriteError(w, r, sdk.ErrForbidden)
		return
	}

	// load destination ids
	if t.DestApplication.ID == 0 {
		a, errDestApp := application.LoadApplicationByName(db, project, t.DestApplication.Name)
		if errDestApp != nil {
			log.Warning("addTriggersHandler> cannot load dst application: %s\n", errDestApp)
			WriteError(w, r, errDestApp)
			return
		}
		t.DestApplication.ID = a.ID
	}
	if !permission.AccessToApplication(t.DestApplication.ID, c.User, permission.PermissionReadWriteExecute) {
		log.Warning("addTriggersHandler> You don't have enought right on this application %s", t.DestApplication.Name)
		WriteError(w, r, sdk.ErrForbidden)
		return
	}

	if t.DestPipeline.ID == 0 {
		p, errDestPip := pipeline.LoadPipeline(db, project, t.DestPipeline.Name, false)
		if errDestPip != nil {
			log.Warning("addTriggersHandler> cannot load dst pipeline: %s\n", errDestPip)
			WriteError(w, r, errDestPip)
			return
		}
		t.DestPipeline.ID = p.ID
	}
	if !permission.AccessToPipeline(sdk.DefaultEnv.ID, t.DestPipeline.ID, c.User, permission.PermissionReadWriteExecute) {
		log.Warning("addTriggersHandler> You don't have enought right on this pipeline %s", t.DestPipeline.Name)
		WriteError(w, r, sdk.ErrForbidden)
		return
	}

	if t.DestEnvironment.ID == 0 && t.DestEnvironment.Name != "" && t.DestEnvironment.Name != sdk.DefaultEnv.Name {
		e, errDestEnv := environment.LoadEnvironmentByName(db, project, t.DestEnvironment.Name)
		if errDestEnv != nil {
			log.Warning("addTriggersHandler> cannot load dst environment: %s\n", errDestEnv)
			WriteError(w, r, errDestEnv)
			return
		}
		t.DestEnvironment.ID = e.ID
	} else if t.DestEnvironment.ID == 0 {
		t.DestEnvironment = sdk.DefaultEnv
	}

	if t.DestEnvironment.ID != sdk.DefaultEnv.ID && !permission.AccessToEnvironment(t.DestEnvironment.ID, c.User, permission.PermissionReadWriteExecute) {
		log.Warning("addTriggersHandler> No enought right on this environment %s: \n", t.DestEnvironment.Name)
		WriteError(w, r, sdk.ErrForbidden)
		return
	}

	tx, errBegin := db.Begin()
	if errBegin != nil {
		WriteError(w, r, errBegin)
		return
	}
	defer tx.Rollback()

	if err := trigger.InsertTrigger(tx, &t); err != nil {
		log.Warning("addTriggerHandler> cannot insert trigger: %s\n", err)
		WriteError(w, r, err)
		return
	}

	// Update src application
	if err := application.UpdateLastModified(tx, &t.SrcApplication); err != nil {
		log.Warning("addTriggerHandler> cannot update loast modified date on src application: %s\n", err)
		WriteError(w, r, err)
		return
	}

	if err := tx.Commit(); err != nil {
		WriteError(w, r, err)
		return
	}

	var errWorkflow error
	t.SrcApplication.Workflows, errWorkflow = application.LoadCDTree(db, project, t.SrcApplication.Name, c.User)
	if errWorkflow != nil {
		log.Warning("addTriggerHandler> cannot load updated workflow: %s\n", errWorkflow)
		WriteError(w, r, errWorkflow)
		return
	}

	WriteJSON(w, r, t.SrcApplication, http.StatusOK)
}

func getTriggerHandler(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {
	vars := mux.Vars(r)
	striggerID := vars["id"]

	triggerID, errParse := strconv.ParseInt(striggerID, 10, 64)
	if errParse != nil {
		log.Warning("getTriggerHandler> TriggerId %s should be an int: %s\n", striggerID, errParse)
		WriteError(w, r, sdk.ErrInvalidID)
		return
	}

	t, errTrig := trigger.LoadTrigger(db, triggerID)
	if errTrig != nil {
		log.Warning("getTriggerHandler> Cannot load trigger %d: %s\n", triggerID, errTrig)
		WriteError(w, r, errTrig)
		return
	}

	WriteJSON(w, r, t, http.StatusOK)
}

func getTriggersHandler(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {
	vars := mux.Vars(r)
	project := vars["key"]
	app := vars["permApplicationName"]
	pip := vars["permPipelineKey"]

	if err := r.ParseForm(); err != nil {
		log.Warning("getTriggersHandler> Cannot parse form: %s\n", err)
		WriteError(w, r, sdk.ErrUnknownError)
		return
	}
	env := r.Form.Get("env")

	a, errApp := application.LoadApplicationByName(db, project, app)
	if errApp != nil {
		log.Warning("getTriggersHandler> cannot load application: %s\n", errApp)
		WriteError(w, r, errApp)
		return
	}

	p, errPip := pipeline.LoadPipeline(db, project, pip, false)
	if errPip != nil {
		log.Warning("getTriggersHandler> cannot load pipeline: %s\n", errPip)
		WriteError(w, r, errPip)
		return
	}

	var envID int64
	if env != "" && env != sdk.DefaultEnv.Name {
		e, errEnv := environment.LoadEnvironmentByName(db, project, env)
		if errEnv != nil {
			log.Warning("getTriggersHandler> cannot load environment: %s\n", errEnv)
			WriteError(w, r, errEnv)
			return
		}
		envID = e.ID

		if e.ID != sdk.DefaultEnv.ID && !permission.AccessToEnvironment(e.ID, c.User, permission.PermissionRead) {
			log.Warning("getTriggersHandler> No enought right on this environment %s: \n", e.Name)
			WriteError(w, r, sdk.ErrForbidden)
			return
		}
	}

	triggers, errTri := trigger.LoadTriggers(db, a.ID, p.ID, envID)
	if errTri != nil {
		log.Warning("getTriggersHandler> cannot load triggers: %s\n", errTri)
		WriteError(w, r, errTri)
		return
	}

	WriteJSON(w, r, triggers, http.StatusOK)
}

func deleteTriggerHandler(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {
	vars := mux.Vars(r)
	projectKey := vars["key"]
	triggerIDS := vars["id"]

	triggerID, errParse := strconv.ParseInt(triggerIDS, 10, 64)
	if errParse != nil {
		log.Warning("deleteTriggerHandler> invalid id (%s)\n", errParse)
		WriteError(w, r, sdk.ErrInvalidID)
		return
	}

	t, errTrigger := trigger.LoadTrigger(db, triggerID)
	if errTrigger != nil {
		log.Warning("deleteTriggerHandler> Cannot load trigger: %s\n", errTrigger)
		WriteError(w, r, errTrigger)
		return
	}

	tx, errBegin := db.Begin()
	if errBegin != nil {
		log.Warning("deleteTriggerHandler> Cannot start transaction: %s\n", errBegin)
		WriteError(w, r, errBegin)
		return
	}
	defer tx.Rollback()

	if err := trigger.DeleteTrigger(tx, triggerID); err != nil {
		log.Warning("deleteTriggerHandler> cannot delete trigger: %s\n", err)
		WriteError(w, r, err)
		return
	}

	if err := application.UpdateLastModified(tx, &t.SrcApplication); err != nil {
		log.Warning("deleteTriggerHandler> cannot update src application last modified date: %s\n", err)
		WriteError(w, r, err)
		return
	}

	if err := tx.Commit(); err != nil {
		log.Warning("deleteTriggerHandler> cannot commit transaction: %s\n", err)
		WriteError(w, r, err)
		return
	}

	var errWorkflow error
	t.SrcApplication.Workflows, errWorkflow = application.LoadCDTree(db, projectKey, t.SrcApplication.Name, c.User)
	if errWorkflow != nil {
		log.Warning("deleteTriggerHandler> cannot load updated workflow: %s\n", errWorkflow)
		WriteError(w, r, errWorkflow)
		return
	}

	WriteJSON(w, r, t.SrcApplication, http.StatusOK)
}

func updateTriggerHandler(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {
	vars := mux.Vars(r)
	projectKey := vars["key"]
	triggerIDS := vars["id"]

	triggerID, errParse := strconv.ParseInt(triggerIDS, 10, 64)
	if errParse != nil {
		log.Warning("updateTriggerHandler> invalid id (%s)\n", errParse)
		WriteError(w, r, sdk.ErrInvalidID)
		return
	}

	// Get args in body
	data, errRead := ioutil.ReadAll(r.Body)
	if errRead != nil {
		log.Warning("updateTriggerHandler> cannot read body: %s\n", errRead)
		WriteError(w, r, sdk.ErrWrongRequest)
		return
	}

	var t sdk.PipelineTrigger
	if err := json.Unmarshal(data, &t); err != nil {
		log.Warning("updateTriggerHandler> cannot unmarshal trigger: %s\n", err)
		WriteError(w, r, sdk.ErrWrongRequest)
		return
	}

	if t.SrcApplication.ID == 0 || t.DestApplication.ID == 0 ||
		t.SrcPipeline.ID == 0 || t.DestPipeline.ID == 0 {
		log.Warning("updateTriggerHandler> IDs should not be zero\n")
		WriteError(w, r, sdk.ErrWrongRequest)
		return
	}

	tx, errBegin := db.Begin()
	if errBegin != nil {
		log.Warning("updateTriggerHandler> cannot start transaction: %s\n", errBegin)
		WriteError(w, r, errBegin)
		return
	}
	defer tx.Rollback()

	t.ID = triggerID
	if err := trigger.UpdateTrigger(tx, &t); err != nil {
		log.Warning("updateTriggerHandler> cannot update trigger: %s\n", err)
		WriteError(w, r, err)
		return
	}

	if err := application.UpdateLastModified(tx, &t.SrcApplication); err != nil {
		log.Warning("updateTriggerHandler> cannot update src application last modified date: %s\n", err)
		WriteError(w, r, err)
		return
	}

	if err := tx.Commit(); err != nil {
		log.Warning("updateTriggerHandler> cannot commit transaction: %s\n", err)
		WriteError(w, r, err)
		return
	}

	var errWorkflow error
	t.SrcApplication.Workflows, errWorkflow = application.LoadCDTree(db, projectKey, t.SrcApplication.Name, c.User)
	if errWorkflow != nil {
		log.Warning("updateTriggerHandler> cannot load updated workflow: %s\n", errWorkflow)
		WriteError(w, r, errWorkflow)
		return
	}

	WriteJSON(w, r, t.SrcApplication, http.StatusOK)
}

func getTriggersAsSourceHandler(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {
	vars := mux.Vars(r)
	project := vars["key"]
	app := vars["permApplicationName"]
	pip := vars["permPipelineKey"]

	if err := r.ParseForm(); err != nil {
		log.Warning("getTriggersAsSourceHandler> Cannot parse form: %s\n", err)
		WriteError(w, r, sdk.ErrWrongRequest)
		return
	}
	env := r.Form.Get("env")

	a, errApp := application.LoadApplicationByName(db, project, app)
	if errApp != nil {
		log.Warning("getTriggersAsSourceHandler> cannot load application: %s\n", errApp)
		WriteError(w, r, errApp)
		return
	}

	p, errPip := pipeline.LoadPipeline(db, project, pip, false)
	if errPip != nil {
		log.Warning("getTriggersAsSourceHandler> cannot load pipeline: %s\n", errPip)
		WriteError(w, r, errPip)
		return
	}

	var envID int64
	if env != "" && env != sdk.DefaultEnv.Name {
		e, errEnv := environment.LoadEnvironmentByName(db, project, env)
		if errEnv != nil {
			log.Warning("getTriggersAsSourceHandler> cannot load environment: %s\n", errEnv)
			WriteError(w, r, errEnv)
			return
		}
		envID = e.ID

		if e.ID != sdk.DefaultEnv.ID && !permission.AccessToEnvironment(e.ID, c.User, permission.PermissionRead) {
			log.Warning("getTriggersAsSourceHandler> No enought right on this environment %s: \n", e.Name)
			WriteError(w, r, sdk.ErrForbidden)
			return
		}
	}

	triggers, errTri := trigger.LoadTriggersAsSource(db, a.ID, p.ID, envID)
	if errTri != nil {
		log.Warning("getTriggersAsSourceHandler> cannot load triggers: %s\n", errTri)
		WriteError(w, r, errTri)
		return
	}

	WriteJSON(w, r, triggers, http.StatusOK)
}
