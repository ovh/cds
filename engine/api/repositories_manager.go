package api

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/event"
	"github.com/ovh/cds/engine/api/hook"
	"github.com/ovh/cds/engine/api/permission"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/poller"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/repositoriesmanager"
	"github.com/ovh/cds/engine/api/user"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/engine/api/workflowv0"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func (api *API) getRepositoriesManagerHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		rms, err := repositoriesmanager.LoadAll(api.mustDB(), api.Cache)
		if err != nil {
			return sdk.WrapError(err, "getRepositoriesManagerHandler> error")
		}
		return WriteJSON(w, rms, http.StatusOK)
	}
}

func (api *API) getRepositoriesManagerForProjectHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars["permProjectKey"]

		proj, errproj := project.Load(api.mustDB(), api.Cache, key, getUser(ctx))
		if errproj != nil {
			return errproj
		}

		return WriteJSON(w, proj.VCSServers, http.StatusOK)
	}
}

func (api *API) repositoriesManagerAuthorizeHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars["permProjectKey"]
		rmName := vars["name"]

		proj, errP := project.Load(api.mustDB(), api.Cache, key, getUser(ctx))
		if errP != nil {
			return sdk.WrapError(errP, "repositoriesManagerAuthorize> Cannot load project")
		}

		if repositoriesmanager.GetProjectVCSServer(proj, rmName) != nil {
			return sdk.WrapError(errP, "repositoriesManagerAuthorize> Cannot load project")
		}

		vcsServer, errVcsServer := repositoriesmanager.NewVCSServerConsumer(api.mustDB, api.Cache, rmName)
		if errVcsServer != nil {
			return sdk.WrapError(errVcsServer, "repositoriesManagerAuthorize> Cannot start transaction")
		}

		token, url, err := vcsServer.AuthorizeRedirect()
		if err != nil {
			return sdk.WrapError(sdk.ErrNoReposManagerAuth, "repositoriesManagerAuthorize> error with AuthorizeRedirect %s", err)
		}
		log.Info("repositoriesManagerAuthorize> [%s] RequestToken=%s; URL=%s", proj.Key, token, url)

		data := map[string]string{
			"project_key":          proj.Key,
			"last_modified":        strconv.FormatInt(time.Now().Unix(), 10),
			"repositories_manager": rmName,
			"url":           url,
			"request_token": token,
			"username":      getUser(ctx).Username,
		}

		api.Cache.Set(cache.Key("reposmanager", "oauth", token), data)
		return WriteJSON(w, data, http.StatusOK)
	}
}

func (api *API) repositoriesManagerOAuthCallbackHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		cberr := r.FormValue("error")
		errDescription := r.FormValue("error_description")
		errURI := r.FormValue("error_uri")

		if cberr != "" {
			log.Error("Callback Error: %s - %s - %s", cberr, errDescription, errURI)
			return fmt.Errorf("OAuth Error %s", cberr)
		}

		code := r.FormValue("code")
		state := r.FormValue("state")

		data := map[string]string{}

		if !api.Cache.Get(cache.Key("reposmanager", "oauth", state), &data) {
			return sdk.WrapError(sdk.ErrForbidden, "repositoriesManagerAuthorizeCallback> Error")
		}
		projectKey := data["project_key"]
		rmName := data["repositories_manager"]
		username := data["username"]

		u, errU := user.LoadUserWithoutAuth(api.mustDB(), username)
		if errU != nil {
			return sdk.WrapError(errU, "repositoriesManagerAuthorizeCallback> Cannot load user %s", username)
		}

		proj, errP := project.Load(api.mustDB(), api.Cache, projectKey, u)
		if errP != nil {
			return sdk.WrapError(errP, "repositoriesManagerAuthorizeCallback> Cannot load project")
		}

		vcsServer, errVCSServer := repositoriesmanager.NewVCSServerConsumer(api.mustDB, api.Cache, rmName)
		if errVCSServer != nil {
			return sdk.WrapError(errVCSServer, "repositoriesManagerAuthorizeCallback> Cannot load project")
		}

		token, secret, err := vcsServer.AuthorizeToken(state, code)
		if err != nil {
			return sdk.WrapError(sdk.ErrNoReposManagerClientAuth, "repositoriesManagerAuthorizeCallback> Error with AuthorizeToken: %s", err)

		}

		if token == "" || secret == "" {
			return sdk.WrapError(sdk.ErrNoReposManagerClientAuth, "repositoriesManagerAuthorizeCallback> token or secret is empty", err)
		}

		tx, errT := api.mustDB().Begin()
		if errT != nil {
			return sdk.WrapError(errT, "repositoriesManagerAuthorizeCallback> Cannot start transaction")
		}
		defer tx.Rollback()

		vcsServerForProject := &sdk.ProjectVCSServer{
			Name:     rmName,
			Username: username,
			Data: map[string]string{
				"token":  token,
				"secret": secret,
			},
		}

		if err := repositoriesmanager.InsertForProject(tx, proj, vcsServerForProject); err != nil {
			return sdk.WrapError(err, "repositoriesManagerAuthorizeCallback> Error with InsertForProject")
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(errT, "repositoriesManagerAuthorizeCallback> Cannot commit transaction")
		}

		event.PublishAddVCSServer(proj, vcsServerForProject.Name, getUser(ctx))

		//Redirect on UI advanced project page
		url := fmt.Sprintf("%s/project/%s?tab=advanced", api.Config.URL.UI, projectKey)
		http.Redirect(w, r, url, http.StatusTemporaryRedirect)

		return nil
	}
}

func (api *API) repositoriesManagerAuthorizeCallbackHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		projectKey := vars["permProjectKey"]
		rmName := vars["name"]

		var tv map[string]interface{}
		if err := UnmarshalBody(r, &tv); err != nil {
			return err
		}

		var token, verifier string
		if tv["request_token"] != nil {
			token = tv["request_token"].(string)
		}
		if tv["verifier"] != nil {
			verifier = tv["verifier"].(string)
		}

		if token == "" || verifier == "" {
			return sdk.WrapError(sdk.ErrWrongRequest, "repositoriesManagerAuthorizeCallback> Cannot get token nor verifier from data")
		}

		proj, errP := project.Load(api.mustDB(), api.Cache, projectKey, getUser(ctx))
		if errP != nil {
			return sdk.WrapError(errP, "repositoriesManagerAuthorizeCallback> Cannot load project")
		}

		vcsServer, errVCSServer := repositoriesmanager.NewVCSServerConsumer(api.mustDB, api.Cache, rmName)
		if errVCSServer != nil {
			return sdk.WrapError(errVCSServer, "repositoriesManagerAuthorizeCallback> Cannot load project")
		}

		tx, errT := api.mustDB().Begin()
		if errT != nil {
			return sdk.WrapError(errT, "repositoriesManagerAuthorizeCallback> Cannot start transaction")
		}
		defer tx.Rollback()

		token, secret, err := vcsServer.AuthorizeToken(token, verifier)
		if err != nil {
			return sdk.WrapError(sdk.ErrNoReposManagerClientAuth, "repositoriesManagerAuthorizeCallback> Error with AuthorizeToken: %s", err)
		}
		log.Debug("repositoriesManagerAuthorizeCallback> [%s] AccessToken=%s; AccessTokenSecret=%s", projectKey, token, secret)

		vcsServerForProject := &sdk.ProjectVCSServer{
			Name:     rmName,
			Username: getUser(ctx).Username,
			Data: map[string]string{
				"token":  token,
				"secret": secret,
			},
		}

		if err := repositoriesmanager.InsertForProject(tx, proj, vcsServerForProject); err != nil {
			return sdk.WrapError(err, "repositoriesManagerAuthorizeCallback> Error with SaveDataForProject")
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(errT, "repositoriesManagerAuthorizeCallback> Cannot commit transaction")
		}

		event.PublishAddVCSServer(proj, vcsServerForProject.Name, getUser(ctx))

		return WriteJSON(w, proj, http.StatusOK)
	}
}

func (api *API) deleteRepositoriesManagerHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		projectKey := vars["permProjectKey"]
		rmName := vars["name"]

		p, errl := project.Load(api.mustDB(), api.Cache, projectKey, getUser(ctx))
		if errl != nil {
			return sdk.WrapError(errl, "deleteRepositoriesManagerHandler> Cannot load project %s", projectKey)
		}

		// Load the repositories manager from the DB
		vcsServer := repositoriesmanager.GetProjectVCSServer(p, rmName)
		if vcsServer == nil {
			return sdk.ErrRepoNotFound
		}

		tx, errb := api.mustDB().Begin()
		if errb != nil {
			return sdk.WrapError(errb, "deleteRepositoriesManagerHandler> Cannot start transaction")
		}
		defer tx.Rollback()

		if err := repositoriesmanager.DeleteForProject(tx, p, vcsServer); err != nil {
			return sdk.WrapError(err, "deleteRepositoriesManagerHandler> error deleting %s-%s", projectKey, rmName)
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "deleteRepositoriesManagerHandler> Cannot commit transaction")
		}

		event.PublishDeleteVCSServer(p, vcsServer.Name, getUser(ctx))

		return WriteJSON(w, p, http.StatusOK)
	}
}

func (api *API) getReposFromRepositoriesManagerHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		projectKey := vars["permProjectKey"]
		rmName := vars["name"]
		sync := FormBool(r, "synchronize")

		proj, errproj := project.Load(api.mustDB(), api.Cache, projectKey, getUser(ctx))
		if errproj != nil {
			return sdk.WrapError(sdk.ErrNoReposManagerClientAuth, "getReposFromRepositoriesManagerHandler> Cannot get client got %s %s", projectKey, rmName)
		}

		log.Debug("getReposFromRepositoriesManagerHandler> Loading repo for %s", rmName)

		vcsServer := repositoriesmanager.GetProjectVCSServer(proj, rmName)
		if vcsServer == nil {
			return sdk.WrapError(sdk.ErrNoReposManagerClientAuth, "getReposFromRepositoriesManagerHandler> Cannot get client got %s %s", projectKey, rmName)
		}

		log.Debug("getReposFromRepositoriesManagerHandler> Loading repo for %s; ok", vcsServer.Name)

		var errAuthClient error
		client, errAuthClient := repositoriesmanager.AuthorizedClient(api.mustDB(), api.Cache, vcsServer)
		if errAuthClient != nil {
			return sdk.WrapError(sdk.ErrNoReposManagerClientAuth, "getReposFromRepositoriesManagerHandler> Cannot get client got %s %s: %v", projectKey, rmName, errAuthClient)
		}

		cacheKey := cache.Key("reposmanager", "repos", projectKey, rmName)
		if sync {
			api.Cache.Delete(cacheKey)
		}

		var repos []sdk.VCSRepo
		if !api.Cache.Get(cacheKey, &repos) || len(repos) == 0 {
			var errRepos error
			repos, errRepos = client.Repos()
			api.Cache.SetWithTTL(cacheKey, repos, 0)
			if errRepos != nil {
				return sdk.WrapError(errRepos, "getReposFromRepositoriesManagerHandler> Cannot get repos: %v", errRepos)
			}
		}

		return WriteJSON(w, repos, http.StatusOK)
	}
}

func (api *API) getRepoFromRepositoriesManagerHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		// Get project name in URL
		vars := mux.Vars(r)
		projectKey := vars["permProjectKey"]
		rmName := vars["name"]
		repoName := r.FormValue("repo")

		if repoName == "" {
			return sdk.NewError(sdk.ErrWrongRequest, fmt.Errorf("Missing repository name 'repo' as a query parameter"))
		}

		proj, errproj := project.Load(api.mustDB(), api.Cache, projectKey, getUser(ctx))
		if errproj != nil {
			return sdk.WrapError(sdk.ErrNoReposManagerClientAuth, "getReposFromRepositoriesManagerHandler> Cannot get client got %s %s", projectKey, rmName)
		}

		vcsServer := repositoriesmanager.GetProjectVCSServer(proj, rmName)
		if vcsServer == nil {
			return sdk.WrapError(sdk.ErrNoReposManagerClientAuth, "getReposFromRepositoriesManagerHandler> Cannot get client got %s %s", projectKey, rmName)
		}

		client, err := repositoriesmanager.AuthorizedClient(api.mustDB(), api.Cache, vcsServer)
		if err != nil {
			return sdk.WrapError(sdk.ErrNoReposManagerClientAuth, "getRepoFromRepositoriesManagerHandler> Cannot get client got %s %s : %s", projectKey, rmName, err)
		}

		log.Info("getRepoFromRepositoriesManagerHandler> Loading repository on %s", vcsServer.Name)

		repo, err := client.RepoByFullname(repoName)
		if err != nil {
			return sdk.WrapError(err, "getRepoFromRepositoriesManagerHandler> Cannot get repos")
		}
		return WriteJSON(w, repo, http.StatusOK)
	}
}

func (api *API) attachRepositoriesManagerHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		projectKey := vars["key"]
		appName := vars["permApplicationName"]
		rmName := vars["name"]
		fullname := r.FormValue("fullname")
		db := api.mustDB()
		u := getUser(ctx)

		app, err := application.LoadByName(db, api.Cache, projectKey, appName, u)
		if err != nil {
			return sdk.WrapError(err, "attachRepositoriesManager> Cannot load application %s", appName)
		}

		//Load the repositoriesManager for the project
		rm, err := repositoriesmanager.LoadForProject(db, projectKey, rmName)
		if err != nil {
			return sdk.WrapError(sdk.ErrNoReposManager, "attachRepositoriesManager> error loading %s-%s: %s", projectKey, rmName, err)
		}

		//Get an authorized Client
		client, err := repositoriesmanager.AuthorizedClient(db, api.Cache, rm)
		if err != nil {
			return sdk.WrapError(sdk.ErrNoReposManagerClientAuth, "attachRepositoriesManager> Cannot get client got %s %s : %s", projectKey, rmName, err)
		}

		if _, err := client.RepoByFullname(fullname); err != nil {
			return sdk.WrapError(sdk.ErrRepoNotFound, "attachRepositoriesManager> Cannot get repo %s: %s", fullname, err)
		}

		app.VCSServer = rm.Name
		app.RepositoryFullname = fullname

		tx, errT := db.Begin()
		if errT != nil {
			return sdk.WrapError(errT, "attachRepositoriesManager> Cannot start transaction")
		}
		defer tx.Rollback()

		if err := repositoriesmanager.InsertForApplication(tx, app, projectKey); err != nil {
			return sdk.WrapError(err, "attachRepositoriesManager> Cannot insert for application")
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "attachRepositoriesManager> Cannot commit transaction")
		}

		usage, errU := loadApplicationUsage(db, projectKey, appName)
		if errU != nil {
			return sdk.WrapError(errU, "attachRepositoriesManager> Cannot load application usage")
		}

		// Update default payload of linked workflow root
		if len(usage.Workflows) > 0 {
			proj, errP := project.Load(db, api.Cache, projectKey, u)
			if errP != nil {
				return sdk.WrapError(errP, "attachRepositoriesManager> Cannot load project")
			}

			for _, wf := range usage.Workflows {
				rootCtx, errNc := workflow.LoadNodeContext(db, api.Cache, proj, wf.RootID, u, workflow.LoadOptions{})
				if errNc != nil {
					return sdk.WrapError(errNc, "attachRepositoriesManager> Cannot DefaultPayloadToMap")
				}

				if rootCtx.ApplicationID != app.ID {
					continue
				}

				wf.Root = &sdk.WorkflowNode{
					Context: rootCtx,
				}
				payload, errD := rootCtx.DefaultPayloadToMap()
				if errD != nil {
					return sdk.WrapError(errP, "attachRepositoriesManager> Cannot DefaultPayloadToMap")
				}

				if _, ok := payload["git.branch"]; ok && payload["git.repository"] == app.RepositoryFullname {
					continue
				}

				defaultPayload, errPay := workflow.DefaultPayload(db, api.Cache, proj, u, &wf)
				if errPay != nil {
					return sdk.WrapError(errPay, "attachRepositoriesManager> Cannot get defaultPayload")
				}
				wf.Root.Context.DefaultPayload = defaultPayload
				if err := workflow.UpdateNodeContext(db, wf.Root.Context); err != nil {
					return sdk.WrapError(err, "attachRepositoriesManager> Cannot update node context %d", wf.Root.Context.ID)
				}
			}
		}

		event.PublishApplicationRepositoryAdd(projectKey, *app, u)

		return WriteJSON(w, app, http.StatusOK)
	}
}

func (api *API) detachRepositoriesManagerHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		projectKey := vars["key"]
		appName := vars["permApplicationName"]
		rmName := vars["name"]
		db := api.mustDB()
		u := getUser(ctx)

		app, errl := application.LoadByName(db, api.Cache, projectKey, appName, u, application.LoadOptions.WithHooks)
		if errl != nil {
			return sdk.WrapError(errl, "detachRepositoriesManager> error on load project %s", projectKey)
		}

		//Load the repositoriesManager for the project
		rm, err := repositoriesmanager.LoadForProject(db, projectKey, rmName)
		if err != nil {
			return sdk.WrapError(sdk.ErrNoReposManager, "attachRepositoriesManager> error loading %s-%s: %s", projectKey, rmName, err)
		}

		//Get an authorized Client
		client, err := repositoriesmanager.AuthorizedClient(db, api.Cache, rm)
		if err != nil {
			return sdk.WrapError(sdk.ErrNoReposManagerClientAuth, "attachRepositoriesManager> Cannot get client got %s %s : %s", projectKey, rmName, err)
		}

		//Remove all the things in a transaction
		tx, errT := db.Begin()
		if errT != nil {
			return sdk.WrapError(errT, "detachRepositoriesManager> Cannot start transaction")
		}
		defer tx.Rollback()

		if err := repositoriesmanager.DeleteForApplication(tx, app); err != nil {
			return sdk.WrapError(err, "detachRepositoriesManager> Cannot delete for application")
		}

		//TODO: to delete after DEPRECATED workflows are deleted
		for _, h := range app.Hooks {
			s := api.Config.URL.API + hook.HookLink
			link := fmt.Sprintf(s, h.UID, h.Project, h.Repository)

			vcsHook := sdk.VCSHook{
				Name:     rm.Name,
				URL:      link,
				Method:   "GET",
				Workflow: false,
			}

			if err := client.DeleteHook(rm.Name, vcsHook); err != nil {
				log.Warning("detachRepositoriesManager> Cannot delete hook on stash: %s", err)
				//do no return, try to delete the hook in database
			}

			if err := hook.DeleteHook(tx, h.ID); err != nil {
				return sdk.WrapError(err, "detachRepositoriesManager> Cannot get hook")
			}
		}

		// Remove reposmanager poller
		if err := poller.DeleteAll(tx, app.ID); err != nil {
			return sdk.WrapError(err, "detachRepositoriesManager> error on poller.DeleteAll")
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "detachRepositoriesManager> Cannot commit transaction")
		}

		usage, errU := loadApplicationUsage(db, projectKey, appName)
		if errU != nil {
			return sdk.WrapError(errU, "detachRepositoriesManager> Cannot load application usage")
		}

		// Update default payload of linked workflow root
		if len(usage.Workflows) > 0 {
			proj, errP := project.Load(db, api.Cache, projectKey, u)
			if errP != nil {
				return sdk.WrapError(errP, "detachRepositoriesManager> Cannot load project")
			}

			hookToDelete := map[string]sdk.WorkflowNodeHook{}
			for _, wf := range usage.Workflows {
				nodeHooks, err := workflow.LoadHooksByNodeID(db, wf.RootID)
				if err != nil {
					return sdk.WrapError(err, "detachRepositoriesManager> Cannot load node hook by nodeID %d", wf.RootID)
				}

				for _, nodeHook := range nodeHooks {
					if nodeHook.WorkflowHookModel.Name != sdk.RepositoryWebHookModelName && nodeHook.WorkflowHookModel.Name != sdk.GitPollerModelName {
						continue
					}
					hookToDelete[nodeHook.UUID] = nodeHook
				}
			}

			if len(hookToDelete) > 0 {
				txDel, errTx := db.Begin()
				if errTx != nil {
					return sdk.WrapError(errTx, "detachRepositoriesManager> Cannot create delete transaction")
				}
				defer func() {
					_ = txDel.Rollback()
				}()

				for _, nodeHook := range hookToDelete {
					if err := workflow.DeleteHook(txDel, &nodeHook); err != nil {
						return sdk.WrapError(err, "detachRepositoriesManager> Cannot delete hooks")
					}
				}
				if err := workflow.DeleteHookConfiguration(txDel, api.Cache, proj, hookToDelete); err != nil {
					return sdk.WrapError(err, "detachRepositoriesManager> Cannot delete hooks vcs configuration")
				}

				if err := txDel.Commit(); err != nil {
					return sdk.WrapError(err, "detachRepositoriesManager> Cannot commit delete transaction")
				}
			}
		}

		event.PublishApplicationRepositoryDelete(projectKey, appName, app.VCSServer, app.RepositoryFullname, u)

		return WriteJSON(w, app, http.StatusOK)
	}
}

func (api *API) addHookOnRepositoriesManagerHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		projectKey := vars["key"]
		appName := vars["permApplicationName"]
		rmName := vars["name"]

		var data map[string]string
		if err := UnmarshalBody(r, &data); err != nil {
			return err
		}

		repoFullname := data["repository_fullname"]
		pipelineName := data["pipeline_name"]

		proj, errproj := project.Load(api.mustDB(), api.Cache, projectKey, getUser(ctx))
		if errproj != nil {
			return errproj
		}

		app, errla := application.LoadByName(api.mustDB(), api.Cache, projectKey, appName, getUser(ctx))
		if errla != nil {
			return sdk.ErrApplicationNotFound
		}

		pipeline, errl := pipeline.LoadPipeline(api.mustDB(), projectKey, pipelineName, false)
		if errl != nil {
			return sdk.ErrPipelineNotFound
		}

		if !permission.AccessToPipeline(projectKey, sdk.DefaultEnv.Name, pipeline.Name, getUser(ctx), permission.PermissionReadWriteExecute) {
			return sdk.WrapError(sdk.ErrForbidden, "addHookOnRepositoriesManagerHandler> You don't have enought right on this pipeline %s", pipeline.Name)
		}

		//Load the repositoriesManager for the project
		rm := repositoriesmanager.GetProjectVCSServer(proj, rmName)
		if rm == nil {
			return sdk.WrapError(sdk.ErrNoReposManager, "attachRepositoriesManager> error loading %s-%s", projectKey, rmName)
		}

		tx, errb := api.mustDB().Begin()
		if errb != nil {
			return sdk.WrapError(errb, "addHookOnRepositoriesManagerHandler> cannot start transaction")
		}
		defer tx.Rollback()

		if _, err := hook.CreateHook(tx, api.Cache, proj, rmName, repoFullname, app, pipeline); err != nil {
			return sdk.WrapError(err, "addHookOnRepositoriesManagerHandler> cannot create hook")
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "addHookOnRepositoriesManagerHandler> cannot commit transaction")
		}

		var errlah error
		app.Hooks, errlah = hook.LoadApplicationHooks(api.mustDB(), app.ID)
		if errlah != nil {
			return sdk.WrapError(errlah, "addHookOnRepositoriesManagerHandler> cannot load application hooks")
		}

		var errW error
		app.Workflows, errW = workflowv0.LoadCDTree(api.mustDB(), api.Cache, projectKey, app.Name, getUser(ctx), "", "", 0)
		if errW != nil {
			return sdk.WrapError(errW, "addHookOnRepositoriesManagerHandler> Cannot load workflow")
		}

		return WriteJSON(w, app, http.StatusCreated)
	}
}

func (api *API) deleteHookOnRepositoriesManagerHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		projectKey := vars["key"]
		appName := vars["permApplicationName"]
		hookIDString := vars["hookId"]

		hookID, errparse := strconv.ParseInt(hookIDString, 10, 64)
		if errparse != nil {
			return sdk.WrapError(sdk.ErrWrongRequest, "deleteHookOnRepositoriesManagerHandler> Unable to parse hook id")
		}

		proj, errproj := project.Load(api.mustDB(), api.Cache, projectKey, getUser(ctx))
		if errproj != nil {
			return sdk.WrapError(errproj, "deleteHookOnRepositoriesManagerHandler> unable to load project")
		}

		app, errload := application.LoadByName(api.mustDB(), api.Cache, projectKey, appName, getUser(ctx))
		if errload != nil {
			return sdk.WrapError(errload, "deleteHookOnRepositoriesManagerHandler> Application %s/%s not found ", projectKey, appName)
		}

		h, errhook := hook.LoadHook(api.mustDB(), hookID)
		if errhook != nil {
			return sdk.WrapError(errhook, "deleteHookOnRepositoriesManagerHandler> Unable to load hook %d ", hookID)
		}

		tx, errtx := api.mustDB().Begin()
		if errtx != nil {
			return sdk.WrapError(errtx, "deleteHookOnRepositoriesManagerHandler> Unable to start transaction")
		}
		defer tx.Rollback()

		if errdelete := hook.DeleteHook(tx, h.ID); errdelete != nil {
			return sdk.WrapError(errdelete, "deleteHookOnRepositoriesManagerHandler> Unable to delete hook %d", h.ID)
		}

		if errtx := tx.Commit(); errtx != nil {
			return sdk.WrapError(errtx, "deleteHookOnRepositoriesManagerHandler> Unable to commit transaction")
		}

		var errW error
		app.Workflows, errW = workflowv0.LoadCDTree(api.mustDB(), api.Cache, projectKey, app.Name, getUser(ctx), "", "", 0)
		if errW != nil {
			return sdk.WrapError(errW, "deleteHookOnRepositoriesManagerHandler> Unable to load workflow")
		}

		rm := repositoriesmanager.GetProjectVCSServer(proj, app.VCSServer)
		if rm == nil {
			return sdk.ErrNoReposManager
		}

		client, errauth := repositoriesmanager.AuthorizedClient(api.mustDB(), api.Cache, rm)
		if errauth != nil {
			return sdk.WrapError(errauth, "deleteHookOnRepositoriesManagerHandler> Cannot get client %s %s", projectKey, app.VCSServer)
		}

		t := strings.Split(app.RepositoryFullname, "/")
		if len(t) != 2 {
			return sdk.WrapError(sdk.ErrRepoNotFound, "deleteHookOnRepositoriesManagerHandler> Application %s repository fullname is not valid %s", app.Name, app.RepositoryFullname)
		}

		s := api.Config.URL.API + hook.HookLink
		log.Info("Will delete hook %s", h.UID)
		link := fmt.Sprintf(s, h.UID, t[0], t[1])

		vcsHook := sdk.VCSHook{
			Name:     rm.Name,
			URL:      link,
			Method:   "GET",
			Workflow: false,
		}

		if errdelete := client.DeleteHook(app.RepositoryFullname, vcsHook); errdelete != nil {
			return sdk.WrapError(errdelete, "deleteHookOnRepositoriesManagerHandler> Cannot delete hook on stash")
		}

		return WriteJSON(w, app, http.StatusOK)
	}
}
