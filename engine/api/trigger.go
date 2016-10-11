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
	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Warning("addTriggerHandler> cannot read body: %s\n", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Unmarshal args
	var t sdk.PipelineTrigger
	err = json.Unmarshal(data, &t)
	if err != nil {
		log.Warning("addTriggerHandler> cannot unmarshal body:  %s\n", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// load source ids
	if t.SrcApplication.ID == 0 {
		a, err := application.LoadApplicationByName(db, project, t.SrcApplication.Name)
		if err != nil {
			log.Warning("addTriggersHandler> cannot load src application: %s\n", err)
			w.WriteHeader(http.StatusBadRequest)
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
		p, err := pipeline.LoadPipeline(db, project, t.SrcPipeline.Name, false)
		if err != nil {
			log.Warning("addTriggersHandler> cannot load src pipeline: %s\n", err)
			w.WriteHeader(http.StatusBadRequest)
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
		e, err := environment.LoadEnvironmentByName(db, project, t.SrcEnvironment.Name)
		if err != nil {
			log.Warning("addTriggersHandler> cannot load src environment: %s\n", err)
			w.WriteHeader(http.StatusBadRequest)
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
		a, err := application.LoadApplicationByName(db, project, t.DestApplication.Name)
		if err != nil {
			log.Warning("addTriggersHandler> cannot load dst application: %s\n", err)
			w.WriteHeader(http.StatusBadRequest)
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
		p, err := pipeline.LoadPipeline(db, project, t.DestPipeline.Name, false)
		if err != nil {
			log.Warning("addTriggersHandler> cannot load dst pipeline: %s\n", err)
			w.WriteHeader(http.StatusBadRequest)
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
		e, err := environment.LoadEnvironmentByName(db, project, t.DestEnvironment.Name)
		if err != nil {
			log.Warning("addTriggersHandler> cannot load dst environment: %s\n", err)
			w.WriteHeader(http.StatusBadRequest)
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

	tx, err := db.Begin()
	if err != nil {
		WriteError(w, r, err)
		return
	}
	defer tx.Rollback()

	err = trigger.InsertTrigger(tx, &t)
	if err != nil {
		log.Warning("addTriggerHandler> cannot insert trigger: %s\n", err)
		WriteError(w, r, err)
		return
	}

	err = tx.Commit()
	if err != nil {
		WriteError(w, r, err)
		return
	}

	WriteJSON(w, r, t, http.StatusCreated)
}

func getTriggerHandler(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {
	vars := mux.Vars(r)
	striggerID := vars["id"]

	triggerID, err := strconv.ParseInt(striggerID, 10, 64)
	if err != nil {
		log.Warning("getTriggerHandler> TriggerId %s should be an int: %s\n", striggerID, err)
		WriteError(w, r, sdk.ErrInvalidID)
		return
	}

	t, err := trigger.LoadTrigger(db, triggerID)
	if err != nil {
		log.Warning("getTriggerHandler> Cannot load trigger %d: %s\n", triggerID, err)
		WriteError(w, r, sdk.ErrUnknownError)
		return
	}

	WriteJSON(w, r, t, http.StatusOK)
}

func getTriggersHandler(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {
	vars := mux.Vars(r)
	project := vars["key"]
	app := vars["permApplicationName"]
	pip := vars["permPipelineKey"]

	err := r.ParseForm()
	if err != nil {
		log.Warning("getTriggersHandler> Cannot parse form: %s\n", err)
		WriteError(w, r, sdk.ErrUnknownError)
		return
	}
	env := r.Form.Get("env")

	a, err := application.LoadApplicationByName(db, project, app)
	if err != nil {
		log.Warning("getTriggersHandler> cannot load application: %s\n", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	p, err := pipeline.LoadPipeline(db, project, pip, false)
	if err != nil {
		log.Warning("getTriggersHandler> cannot load pipeline: %s\n", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	var envID int64
	if env != "" && env != sdk.DefaultEnv.Name {
		e, err := environment.LoadEnvironmentByName(db, project, env)
		if err != nil {
			log.Warning("getTriggersHandler> cannot load environment: %s\n", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		envID = e.ID

		if e.ID != sdk.DefaultEnv.ID && !permission.AccessToEnvironment(e.ID, c.User, permission.PermissionRead) {
			log.Warning("getTriggersHandler> No enought right on this environment %s: \n", e.Name)
			WriteError(w, r, sdk.ErrForbidden)
			return
		}
	}

	triggers, err := trigger.LoadTriggers(db, a.ID, p.ID, envID)
	if err != nil {
		log.Warning("getTriggersHandler> cannot load triggers: %s\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	WriteJSON(w, r, triggers, http.StatusOK)
}

func deleteTriggerHandler(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {
	vars := mux.Vars(r)
	triggerIDS := vars["id"]

	triggerID, err := strconv.ParseInt(triggerIDS, 10, 64)
	if err != nil {
		log.Warning("deleteTriggerHandler> invalid id (%s)\n", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	err = trigger.DeleteTrigger(db, triggerID)
	if err != nil {
		log.Warning("deleteTriggerHandler> cannot delete trigger: %s\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func updateTriggerHandler(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {
	vars := mux.Vars(r)
	triggerIDS := vars["id"]

	triggerID, err := strconv.ParseInt(triggerIDS, 10, 64)
	if err != nil {
		log.Warning("deleteTriggerHandler> invalid id (%s)\n", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Get args in body
	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Warning("updateTriggerHandler> cannot read body: %s\n", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	var t sdk.PipelineTrigger
	err = json.Unmarshal(data, &t)
	if err != nil {
		log.Warning("updateTriggerHandler> cannot unmarshal trigger: %s\n", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	/*
		TODO: remove this, useless now
		// Before updating trigger, replace PasswordPlaceholder by
		// actual variable value
		clear, err := trigger.LoadTrigger(db, t.ID, trigger.WithClearSecrets())
		clear, err := trigger.LoadTrigger(db, t.ID, trigger.WithClearSecrets())
		if err != nil {
			log.Warning("updateTriggerHandler> cannot load trigger: %s\n", err)
			WriteError(w, r, err)
			return
		}
		for i := range t.Parameters {
			if t.Parameters[i].Type != sdk.PasswordParameter || t.Parameters[i].Value != sdk.PasswordPlaceholder {
				continue
			}
			for _, clearP := range clear.Parameters {
				if t.Parameters[i].Name == clearP.Name {
					t.Parameters[i].Value = clearP.Value
				}
			}

		}
	*/

	t.ID = triggerID
	err = trigger.UpdateTrigger(db, t)
	if err != nil {
		log.Warning("updateTriggerHandler> cannot update trigger: %s\n", err)
		WriteError(w, r, err)
		return
	}
}

func getTriggersAsSourceHandler(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {
	vars := mux.Vars(r)
	project := vars["key"]
	app := vars["permApplicationName"]
	pip := vars["permPipelineKey"]

	err := r.ParseForm()
	if err != nil {
		log.Warning("getTriggersAsSourceHandler> Cannot parse form: %s\n", err)
		WriteError(w, r, sdk.ErrUnknownError)
		return
	}
	env := r.Form.Get("env")

	a, err := application.LoadApplicationByName(db, project, app)
	if err != nil {
		log.Warning("getTriggersAsSourceHandler> cannot load application: %s\n", err)
		WriteError(w, r, err)
		return
	}

	p, err := pipeline.LoadPipeline(db, project, pip, false)
	if err != nil {
		log.Warning("getTriggersAsSourceHandler> cannot load pipeline: %s\n", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	var envID int64
	if env != "" && env != sdk.DefaultEnv.Name {
		e, err := environment.LoadEnvironmentByName(db, project, env)
		if err != nil {
			log.Warning("getTriggersAsSourceHandler> cannot load environment: %s\n", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		envID = e.ID

		if e.ID != sdk.DefaultEnv.ID && !permission.AccessToEnvironment(e.ID, c.User, permission.PermissionRead) {
			log.Warning("getTriggersAsSourceHandler> No enought right on this environment %s: \n", e.Name)
			WriteError(w, r, sdk.ErrForbidden)
			return
		}
	}

	triggers, err := trigger.LoadTriggersAsSource(db, a.ID, p.ID, envID)
	if err != nil {
		log.Warning("getTriggersAsSourceHandler> cannot load triggers: %s\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	WriteJSON(w, r, triggers, http.StatusOK)
}
