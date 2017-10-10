package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-gorp/gorp"
	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/hook"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/workflowv0"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func processStashHook(w http.ResponseWriter, r *http.Request, data []byte) hook.ReceivedHook {
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

	return rh
}

func processGitlabHook(w http.ResponseWriter, r *http.Request, data []byte) (hook.ReceivedHook, error) {

	type gitlabEvent struct {
		ObjectKind  string `json:"object_kind"`
		Ref         string `json:"ref"`
		UserName    string `json:"user_name"`
		CheckoutSha string `json:"checkout_sha"`
	}

	var ge gitlabEvent
	if err := json.Unmarshal(data, &ge); err != nil {
		return hook.ReceivedHook{}, err
	}

	ge.Ref = strings.TrimPrefix(ge.Ref, "refs/heads/")

	rh := hook.ReceivedHook{
		URL:        *r.URL,
		Data:       data,
		ProjectKey: r.FormValue("project"),
		Repository: r.FormValue("name"),
		Branch:     ge.Ref,
		Hash:       ge.CheckoutSha,
		Author:     ge.UserName,
		Message:    ge.ObjectKind,
		UID:        r.FormValue("uid"),
	}

	return rh, nil
}

func (api *API) receiveHookHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		// Get body
		data, err := ioutil.ReadAll(r.Body)
		if err != nil {
			return sdk.ErrWrongRequest

		}

		if err = r.ParseForm(); err != nil {
			return sdk.WrapError(err, "receiveHook> cannot parse query params")
		}

		var rh hook.ReceivedHook
		if r.Header.Get("X-Gitlab-Event") != "" {
			rh, err = processGitlabHook(w, r, data)
			if err != nil {
				return err
			}
		} else {
			rh = processStashHook(w, r, data)
		}

		db := api.DBConnectionFactory.GetDBMap()
		if db == nil {
			hook.Recovery(api.Cache, rh, fmt.Errorf("database not available"))
			return err
		}

		if err := processHook(api.DBConnectionFactory.GetDBMap, api.Cache, rh); err != nil {
			hook.Recovery(api.Cache, rh, err)
			return err

		}

		return nil
	}
}

func (api *API) addHookHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		projectKey := vars["key"]
		appName := vars["permApplicationName"]

		var h sdk.Hook
		if err := UnmarshalBody(r, &h); err != nil {
			return err
		}
		h.Enabled = true

		// Insert hook in database
		if err := hook.InsertHook(api.mustDB(), &h); err != nil {
			return sdk.WrapError(err, "addHook: cannot insert hook in db")

		}

		app, errA := application.LoadByName(api.mustDB(), api.Cache, projectKey, appName, getUser(ctx), application.LoadOptions.WithHooks)
		if errA != nil {
			return sdk.WrapError(errA, "addHook: Cannot load application")
		}
		var errW error
		app.Workflows, errW = workflowv0.LoadCDTree(api.mustDB(), api.Cache, projectKey, appName, getUser(ctx), "", "", 0)
		if errW != nil {
			return sdk.WrapError(errA, "addHook: Cannot load workflow")
		}

		return WriteJSON(w, r, app, http.StatusOK)
	}
}

func (api *API) updateHookHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		projectKey := vars["key"]
		appName := vars["permApplicationName"]

		var h sdk.Hook
		if err := UnmarshalBody(r, &h); err != nil {
			return sdk.WrapError(err, "updateHookHandler")
		}

		app, errA := application.LoadByName(api.mustDB(), api.Cache, projectKey, appName, getUser(ctx), application.LoadOptions.WithHooks)
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
		if err := hook.UpdateHook(api.mustDB(), h); err != nil {
			return sdk.WrapError(err, "updateHookHandler: cannot update hook")
		}

		var errW error
		app.Workflows, errW = workflowv0.LoadCDTree(api.mustDB(), api.Cache, projectKey, app.Name, getUser(ctx), "", "", 0)
		if errW != nil {
			return sdk.WrapError(errW, "updateHookHandler: Cannot load workflow")
		}

		return WriteJSON(w, r, app, http.StatusOK)
	}
}

func (api *API) getApplicationHooksHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		projectName := vars["key"]
		appName := vars["permApplicationName"]

		a, err := application.LoadByName(api.mustDB(), api.Cache, projectName, appName, getUser(ctx), application.LoadOptions.WithHooks)
		if err != nil {
			return sdk.WrapError(err, "getApplicationHooksHandler> cannot load application %s/%s", projectName, appName)
		}

		return WriteJSON(w, r, a.Hooks, http.StatusOK)
	}
}

func (api *API) getHooksHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		projectName := vars["key"]
		appName := vars["permApplicationName"]
		pipelineName := vars["permPipelineKey"]

		p, err := pipeline.LoadPipeline(api.mustDB(), projectName, pipelineName, false)
		if err != nil {
			if err != sdk.ErrPipelineNotFound {
				log.Warning("getHooks> cannot load pipeline %s/%s: %s\n", projectName, pipelineName, err)
			}
			return err
		}

		a, err := application.LoadByName(api.mustDB(), api.Cache, projectName, appName, getUser(ctx))
		if err != nil {
			return sdk.WrapError(err, "getHooks> cannot load application %s/%s", projectName, appName)
		}

		hooks, err := hook.LoadPipelineHooks(api.mustDB(), p.ID, a.ID)
		if err != nil {
			return sdk.WrapError(err, "getHooks> cannot load hooks")
		}

		return WriteJSON(w, r, hooks, http.StatusOK)
	}
}

func (api *API) deleteHookHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		idS := vars["id"]

		id, err := strconv.ParseInt(idS, 10, 64)
		if err != nil {
			return sdk.ErrWrongRequest

		}

		_, err = hook.LoadHook(api.mustDB(), id)
		if err != nil {
			return sdk.WrapError(err, "deleteHook> cannot load hook")

		}

		err = hook.DeleteHook(api.mustDB(), id)
		if err != nil {
			return sdk.WrapError(err, "deleteHook> cannot delete hook")

		}
		return nil
	}
}

//hookRecoverer is the go-routine which catches on-error hook
func hookRecoverer(c context.Context, DBFunc func() *gorp.DbMap, store cache.Store) {
	tick := time.NewTicker(10 * time.Second).C
	for {
		select {
		case <-c.Done():
			if c.Err() != nil {
				log.Error("Exiting hookRecoverer: %v", c.Err())
				return
			}
		case <-tick:
			h := hook.ReceivedHook{}
			store.DequeueWithContext(c, "hook:recovery", &h)
			if err := c.Err(); err != nil {
				log.Error("Exiting hookRecoverer: %v", err)
				return
			}
			if h.Repository != "" {
				if err := processHook(DBFunc, store, h); err != nil {
					hook.Recovery(store, h, err)
				}
			}
		}
	}
}

//processHook is the core function for hook processing
func processHook(DBFunc func() *gorp.DbMap, store cache.Store, h hook.ReceivedHook) error {
	db := DBFunc()
	if db == nil {
		return fmt.Errorf("database not available")
	}

	// Logging stuff
	if err := hook.InsertReceivedHook(db, h.URL.String(), string(h.Data)); err != nil {
		return sdk.WrapError(err, "processHook> cannot insert received hook in db")
	}

	// Actual search of hook binding
	hooks, err := hook.LoadHooks(db, h.ProjectKey, h.Repository)
	if err != nil {
		return sdk.WrapError(err, "processHook> cannot load hook for %s/%s", h.ProjectKey, h.Repository)
	}

	// If branch is DELETE'd, remove all builds related to this branch
	if h.Message == "DELETE" {
		log.Warning("processHook> Removing builds in %s/%s on branch %s\n", h.ProjectKey, h.Repository, h.Branch)
		if err := hook.DeleteBranchBuilds(db, hooks, h.Branch); err != nil {
			return err
		}
		return nil
	}

	log.Debug("Executing %d hooks for %s/%s on branch %s\n", len(hooks), h.ProjectKey, h.Repository, h.Branch)
	found := false

	for i := range hooks {
		//begin a tx
		tx, err := db.Begin()
		if err != nil {
			return err
		}
		defer tx.Rollback()

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
			return sdk.WrapError(err, "processHook> Cannot load pipeline")
		}

		// get Project
		// Load project
		projectData, err := project.LoadByPipelineID(tx, store, nil, p.ID)
		if err != nil {
			return sdk.WrapError(err, "processHook> Cannot load project for pipeline %s", p.Name)
		}

		projectsVar, err := project.GetAllVariableInProject(tx, projectData.ID)
		if err != nil {
			return sdk.WrapError(err, "processHook> Cannot load project variable")
		}
		projectData.Variable = projectsVar

		pb, err := application.TriggerPipeline(tx, store, hooks[i], h.Branch, h.Hash, h.Author, p, projectData)
		if err != nil {
			return sdk.WrapError(err, "processHook> cannot trigger pipeline %d", hooks[i].Pipeline.ID)
		}
		if pb != nil {
			log.Debug("processHook> Triggered %s/%s/%s", h.ProjectKey, h.Repository, h.Branch)
		} else {
			log.Info("processHook> Did not trigger %s/%s/%s", h.ProjectKey, h.Repository, h.Branch)
		}

		if err := tx.Commit(); err != nil {
			log.Error("processHook> Cannot commit tx; %s", err)
			return err
		}

		go func(h *sdk.Hook) {
			app, errapp := application.LoadByID(DBFunc(), store, h.ApplicationID, nil, application.LoadOptions.WithRepositoryManager)
			if errapp != nil {
				log.Warning("processHook> Unable to load application %s", errapp)
			}

			if _, err := pipeline.UpdatePipelineBuildCommits(DBFunc(), projectData, p, app, &sdk.DefaultEnv, pb); err != nil {
				log.Warning("processHook> Unable to update pipeline build commits: %s", err)
			}
		}(&hooks[i])
	}

	if !found {
		return sdk.WrapError(sdk.ErrUnauthorized, "processHook> Bad uid for hook [%s/%s], got uid='%s'", h.ProjectKey, h.Repository, h.UID)
	}

	return nil
}
