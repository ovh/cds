package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/context"
	"github.com/ovh/cds/engine/api/database"
	"github.com/ovh/cds/engine/api/hook"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/log"
	"github.com/ovh/cds/sdk"
)

func receiveHook(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {
	// Get body
	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if err = r.ParseForm(); err != nil {
		log.Warning("receiveHook> cannot parse query params: %s\n", err)
		WriteError(w, r, err)
	}

	rh := hook.ReceivedHook{
		URL:        *r.URL,
		Data:       data,
		ProjectKey: r.FormValue("project"),
		Repository: r.FormValue("name"),
		Branch:     r.FormValue("branch"),
		Hash:       r.FormValue("hash"),
		Author:     r.FormValue("author"),
		Message:    r.FormValue("message"),
		UID:        r.FormValue("uid"),
	}

	if db == nil {
		hook.Recovery(rh, fmt.Errorf("database not available"))
		WriteError(w, r, err)
		return
	}

	if err := processHook(rh); err != nil {
		hook.Recovery(rh, err)
		WriteError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func addHook(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {
	// Get body
	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Warning("addHook: Cannot read body: %s\n", err)
		WriteError(w, r, err)
		return
	}

	var h sdk.Hook
	err = json.Unmarshal(data, &h)
	if err != nil {
		log.Warning("addHook: Cannot unmarshal body: %s\n", err)
		WriteError(w, r, err)
		return
	}

	h.Enabled = true

	// Insert hook in database
	err = hook.InsertHook(db, &h)
	if err != nil {
		log.Warning("addHook: cannot insert hook in db: %s\n", err)
		WriteError(w, r, err)
		return
	}

	WriteJSON(w, r, h, http.StatusOK)
}

func updateHookHandler(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {

	// Get body
	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Warning("updateHookHandler: Cannot read body: %s\n", err)
		WriteError(w, r, err)
		return
	}

	var h sdk.Hook
	err = json.Unmarshal(data, &h)
	if err != nil {
		log.Warning("updateHookHandler: Cannot unmarshal body: %s\n", err)
		WriteError(w, r, err)
		return
	}

	// Update hook in database
	err = hook.UpdateHook(db, h)
	if err != nil {
		log.Warning("updateHookHandler: cannot update hook in db: %s\n", err)
		WriteError(w, r, err)
		return
	}
}

func getApplicationHooksHandler(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {
	vars := mux.Vars(r)
	projectName := vars["key"]
	appName := vars["permApplicationName"]

	a, err := application.LoadApplicationByName(db, projectName, appName)
	if err != nil {
		log.Warning("getApplicationHooksHandler> cannot load application %s/%s: %s\n", projectName, appName, err)
		WriteError(w, r, err)
		return
	}

	hooks, err := hook.LoadApplicationHooks(db, a.ID)
	if err != nil {
		log.Warning("getApplicationHooksHandler> cannot load hooks: %s\n", err)
		WriteError(w, r, err)
		return
	}

	WriteJSON(w, r, hooks, http.StatusOK)
}

func getHooks(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {
	vars := mux.Vars(r)
	projectName := vars["key"]
	appName := vars["permApplicationName"]
	pipelineName := vars["permPipelineKey"]

	p, err := pipeline.LoadPipeline(db, projectName, pipelineName, false)
	if err != nil {
		if err != sdk.ErrPipelineNotFound {
			log.Warning("getHooks> cannot load pipeline %s/%s: %s\n", projectName, pipelineName, err)
		}
		WriteError(w, r, err)
		return
	}

	a, err := application.LoadApplicationByName(db, projectName, appName)
	if err != nil {
		log.Warning("getHooks> cannot load application %s/%s: %s\n", projectName, appName, err)
		WriteError(w, r, err)
		return
	}

	hooks, err := hook.LoadPipelineHooks(db, p.ID, a.ID)
	if err != nil {
		log.Warning("getHooks> cannot load hooks: %s\n", err)
		WriteError(w, r, err)
		return
	}

	WriteJSON(w, r, hooks, http.StatusOK)
}

func deleteHook(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {
	vars := mux.Vars(r)
	idS := vars["id"]

	id, err := strconv.ParseInt(idS, 10, 64)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	_, err = hook.LoadHook(db, id)
	if err != nil {
		log.Warning("deleteHook> cannot load hook: %s\n", err)
		WriteError(w, r, err)
		return
	}

	err = hook.DeleteHook(db, id)
	if err != nil {
		log.Warning("deleteHook> cannot delete hook: %s\n", err)
		WriteError(w, r, err)
		return
	}
}

//hookRecoverer is the go-routine which catches on-error hook
func hookRecoverer() {
	for {
		h := hook.ReceivedHook{}
		cache.Dequeue("hook:recovery", &h)
		if h.Repository != "" {
			if err := processHook(h); err != nil {
				hook.Recovery(h, err)
			}
		}
		time.Sleep(10 * time.Second)
	}
}

//processHook is the core function for hook processing
func processHook(h hook.ReceivedHook) error {
	db := database.DB()
	if db == nil {
		return fmt.Errorf("database not available")
	}

	// Logging stuff
	if err := hook.InsertReceivedHook(db, h.URL.String(), string(h.Data)); err != nil {
		log.Warning("processHook> cannot insert received hook in db: %s\n", err)
		return err
	}

	// Actual search of hook binding
	hooks, err := hook.LoadHooks(db, h.ProjectKey, h.Repository)
	if err != nil {
		log.Warning("processHook> cannot load hook for %s/%s: %s\n", h.ProjectKey, h.Repository, err)
		return err
	}

	// If branch is DELETE'd, remove all builds related to this branch
	if h.Message == "DELETE" {
		log.Warning("processHook> Removing builds in %s/%s on branch %s\n", h.ProjectKey, h.Repository, h.Branch)
		if err := hook.DeleteBranchBuilds(db, hooks, h.Branch); err != nil {
			return err
		}
		return nil
	}

	log.Info("Executing %d hooks for %s/%s on branch %s\n", len(hooks), h.ProjectKey, h.Repository, h.Branch)
	found := false
	//begin a tx
	tx, err := db.Begin()
	defer tx.Rollback()

	for i := range hooks {
		if !hooks[i].Enabled {
			continue
		}

		if hooks[i].UID != h.UID {
			continue
		}

		found = true

		// create pipeline object
		p, err := pipeline.LoadPipelineByID(tx, hooks[i].Pipeline.ID)
		if err != nil {
			log.Warning("processHook> Cannot load pipeline: %s\n", err)
			return err
		}

		// get Project
		// Load project
		projectData, err := project.LoadProjectByPipelineID(tx, p.ID)
		if err != nil {
			log.Warning("processHook> Cannot load project for pipeline %s: %s\n", p.Name, err)
			return err
		}

		projectsVar, err := project.GetAllVariableInProject(tx, projectData.ID)
		if err != nil {
			log.Warning("processHook> Cannot load project variable: %s\n", err)
			return err
		}
		projectData.Variable = projectsVar

		ok, err := application.TriggerPipeline(tx, hooks[i], h.Branch, h.Hash, h.Author, p, projectData)
		if err != nil {
			log.Warning("processHook> cannot trigger pipeline %d: %s\n", hooks[i].Pipeline.ID, err)
			return err
		}
		if ok {
			log.Debug("processHook> Triggered %s/%s/%s", h.ProjectKey, h.Repository, h.Branch)
		} else {
			log.Notice("processHook> Did not trigger %s/%s/%s", h.ProjectKey, h.Repository, h.Branch)
		}
	}

	if err := tx.Commit(); err != nil {
		log.Critical("processHook> Cannot commit tx; %s", err)
		return err
	}

	if !found {
		log.Warning("processHook> Bad uid for hook [%s/%s], got uid='%s'", h.ProjectKey, h.Repository, h.UID)
		return sdk.ErrUnauthorized
	}

	return nil
}
