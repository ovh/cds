package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"

	"github.com/go-gorp/gorp"
	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/context"
	"github.com/ovh/cds/engine/api/hook"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/engine/log"
	"github.com/ovh/cds/sdk"
)

func receiveHook(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	// Get body
	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return sdk.ErrWrongRequest

	}

	if err = r.ParseForm(); err != nil {
		log.Warning("receiveHook> cannot parse query params: %s\n", err)
		return err
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
		return err

	}

	if err := processHook(db, rh); err != nil {
		hook.Recovery(rh, err)
		return err

	}

	return nil
}

func addHook(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	vars := mux.Vars(r)
	projectKey := vars["key"]
	appName := vars["permApplicationName"]

	var h sdk.Hook
	if err := UnmarshalBody(r, &h); err != nil {
		return err
	}
	h.Enabled = true

	// Insert hook in database
	if err := hook.InsertHook(db, &h); err != nil {
		log.Warning("addHook: cannot insert hook in db: %s\n", err)
		return err

	}

	app, errA := application.LoadByName(db, projectKey, appName, c.User, application.LoadOptions.WithHooks)
	if errA != nil {
		return sdk.WrapError(errA, "addHook: Cannot load application")
	}
	var errW error
	app.Workflows, errW = workflow.LoadCDTree(db, projectKey, appName, c.User)
	if errW != nil {
		return sdk.WrapError(errA, "addHook: Cannot load workflow")
	}

	return WriteJSON(w, r, app, http.StatusOK)
}

func updateHookHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	vars := mux.Vars(r)
	projectKey := vars["key"]
	appName := vars["permApplicationName"]

	var h sdk.Hook
	if err := UnmarshalBody(r, &h); err != nil {
		return sdk.WrapError(err, "updateHookHandler")
	}

	app, errA := application.LoadByName(db, projectKey, appName, c.User, application.LoadOptions.WithHooks)
	if errA != nil {
		return sdk.WrapError(errA, "updateHookHandler> Cannot load application")
	}

	found := false
	for _, hookInApp := range app.Hooks {
		if hookInApp.ID == h.ID {
			found = true
			break
		}
	}

	if !found {
		return sdk.WrapError(sdk.ErrNoHook, "updateHookHandler")
	}

	// Update hook in database
	if err := hook.UpdateHook(db, h); err != nil {
		return sdk.WrapError(err, "updateHookHandler: cannot update hook")
	}

	var errW error
	app.Workflows, errW = workflow.LoadCDTree(db, projectKey, app.Name, c.User)
	if errW != nil {
		return sdk.WrapError(errW, "updateHookHandler: Cannot load workflow")
	}

	return WriteJSON(w, r, app, http.StatusOK)
}

func getApplicationHooksHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	vars := mux.Vars(r)
	projectName := vars["key"]
	appName := vars["permApplicationName"]

	a, err := application.LoadByName(db, projectName, appName, c.User, application.LoadOptions.WithHooks)
	if err != nil {
		log.Warning("getApplicationHooksHandler> cannot load application %s/%s: %s\n", projectName, appName, err)
		return err
	}

	return WriteJSON(w, r, a.Hooks, http.StatusOK)
}

func getHooks(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	vars := mux.Vars(r)
	projectName := vars["key"]
	appName := vars["permApplicationName"]
	pipelineName := vars["permPipelineKey"]

	p, err := pipeline.LoadPipeline(db, projectName, pipelineName, false)
	if err != nil {
		if err != sdk.ErrPipelineNotFound {
			log.Warning("getHooks> cannot load pipeline %s/%s: %s\n", projectName, pipelineName, err)
		}
		return err
	}

	a, err := application.LoadByName(db, projectName, appName, c.User)
	if err != nil {
		log.Warning("getHooks> cannot load application %s/%s: %s\n", projectName, appName, err)
		return err
	}

	hooks, err := hook.LoadPipelineHooks(db, p.ID, a.ID)
	if err != nil {
		log.Warning("getHooks> cannot load hooks: %s\n", err)
		return err
	}

	return WriteJSON(w, r, hooks, http.StatusOK)
}

func deleteHook(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	vars := mux.Vars(r)
	idS := vars["id"]

	id, err := strconv.ParseInt(idS, 10, 64)
	if err != nil {
		return sdk.ErrWrongRequest

	}

	_, err = hook.LoadHook(db, id)
	if err != nil {
		log.Warning("deleteHook> cannot load hook: %s\n", err)
		return err

	}

	err = hook.DeleteHook(db, id)
	if err != nil {
		log.Warning("deleteHook> cannot delete hook: %s\n", err)
		return err

	}
	return nil
}

//hookRecoverer is the go-routine which catches on-error hook
func hookRecoverer(DBFunc func() *gorp.DbMap) {
	for {
		h := hook.ReceivedHook{}
		cache.Dequeue("hook:recovery", &h)
		if h.Repository != "" {
			if err := processHook(DBFunc(), h); err != nil {
				hook.Recovery(h, err)
			}
		}
		time.Sleep(10 * time.Second)
	}
}

//processHook is the core function for hook processing
func processHook(db *gorp.DbMap, h hook.ReceivedHook) error {
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
	if err != nil {
		return err
	}
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
		p, err := pipeline.LoadPipelineByID(tx, hooks[i].Pipeline.ID, true)
		if err != nil {
			log.Warning("processHook> Cannot load pipeline: %s\n", err)
			return err
		}

		// get Project
		// Load project
		projectData, err := project.LoadByPipelineID(tx, nil, p.ID)
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
