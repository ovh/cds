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
	"github.com/ovh/cds/engine/api/repositoriesmanager"
	"github.com/ovh/cds/engine/api/workflow"
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
		After       string `json:"after"`
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

	// Branch deleted
	if ge.After == "0000000000000000000000000000000000000000" {
		rh.Message = "DELETE"
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

		go func() {
			db := api.DBConnectionFactory.GetDBMap()
			if db == nil {
				err := fmt.Errorf("database not available")
				hook.Recovery(api.Cache, rh, err)
				log.Error("receiveHookHandler> Error, try to recover...: %v", err)
				return
			}

			if err := processHook(api.DBConnectionFactory.GetDBMap, api.Cache, rh); err != nil {
				log.Error("receiveHookHandler> Error, try to recover...: %v", err)
				hook.Recovery(api.Cache, rh, err)
			}
		}()

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
		return hook.DeleteBranchBuilds(db, store, hooks, h.Branch)
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
			app, errapp := application.LoadByID(DBFunc(), store, h.ApplicationID, nil)
			if errapp != nil {
				log.Warning("processHook> Unable to load application %s", errapp)
			}

			if _, err := pipeline.UpdatePipelineBuildCommits(DBFunc(), store, projectData, p, app, &sdk.DefaultEnv, pb); err != nil {
				log.Warning("processHook> Unable to update pipeline build commits: %s", err)
			}
		}(&hooks[i])
	}

	if !found {
		return sdk.WrapError(sdk.ErrUnauthorized, "processHook> Bad uid for hook [%s/%s], got uid='%s'", h.ProjectKey, h.Repository, h.UID)
	}

	return nil
}

func (api *API) getHookPollingVCSEvents() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		uid := vars["uid"]
		vcsServerParam := vars["vcsServer"]
		lastExec := time.Now()
		workflowID, errV := requestVarInt(r, "workflowID")
		if errV != nil {
			return sdk.WrapError(sdk.ErrWrongRequest, "getHookPollingVCSEvents> cannot convert workflowID to int %s", errV)
		}

		if r.Header.Get("X-CDS-Last-Execution") != "" {
			if ts, err := strconv.ParseInt(r.Header.Get("X-CDS-Last-Execution"), 10, 64); err == nil {
				lastExec = time.Unix(0, ts)
			}
		}

		h, errL := workflow.LoadHookByUUID(api.mustDB(), uid)
		if errL != nil {
			return sdk.WrapError(errL, "getHookPollingVCSEvents> cannot load hook")
		}
		if h == nil {
			return sdk.ErrNotFound
		}

		proj, errProj := project.Load(api.mustDB(), api.Cache, h.Config["project"].Value, nil)
		if errProj != nil {
			return sdk.WrapError(errProj, "getHookPollingVCSEvents> cannot load project")
		}

		//get the client for the repositories manager
		vcsServer := repositoriesmanager.GetProjectVCSServer(proj, vcsServerParam)
		client, errR := repositoriesmanager.AuthorizedClient(api.mustDB(), api.Cache, vcsServer)
		if errR != nil {
			return sdk.WrapError(errR, "getHookPollingVCSEvents> Unable to get client for %s %s", proj.Key, vcsServerParam)
		}

		//Check if the polling if disabled
		if info, err := repositoriesmanager.GetPollingInfos(client); err != nil {
			return err
		} else if info.PollingDisabled || !info.PollingSupported {
			log.Info("getHookPollingVCSEvents> %s polling is disabled", vcsServer.Name)
			return WriteJSON(w, r, nil, http.StatusOK)
		}

		events, pollingDelay, err := client.GetEvents(h.Config["repoFullName"].Value, lastExec)
		if err != nil && err.Error() != "No new events" {
			return sdk.WrapError(err, "Polling> Unable to get events for %s %s", proj.Key, vcsServerParam)
		}
		pushEvents, err := client.PushEvents(h.Config["repoFullName"].Value, events)
		if err != nil {
			return sdk.WrapError(err, "getHookPollingVCSEvent> ")
		}

		pullRequestEvents, err := client.PullRequestEvents(h.Config["repoFullName"].Value, events)
		if err != nil {
			return sdk.WrapError(err, "getHookPollingVCSEvent> ")
		}

		repoEvents := sdk.RepositoryEvents{}
		for _, pushEvent := range pushEvents {
			exist, errB := workflow.BuildExist(api.mustDB(), h.Config["project"].Value, workflowID, pushEvent.Commit.Hash)
			if errB != nil {
				return errB
			}
			if !exist {
				repoEvents.PushEvents = append(repoEvents.PushEvents, pushEvent)
			}
		}

		for _, pullRequestEvent := range pullRequestEvents {
			exist, errB := workflow.BuildExist(api.mustDB(), h.Config["project"].Value, workflowID, pullRequestEvent.Head.Commit.Hash)
			if errB != nil {
				return errB
			}
			if !exist {
				repoEvents.PullRequestEvents = append(repoEvents.PullRequestEvents, pullRequestEvent)
			}
		}

		w.Header().Add("X-CDS-Poll-Interval", fmt.Sprintf("%.0f", pollingDelay.Seconds()))

		return WriteJSON(w, r, repoEvents, http.StatusOK)
	}
}
