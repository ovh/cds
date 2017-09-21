package api

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/hook"
	"github.com/ovh/cds/engine/api/permission"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/poller"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/repositoriesmanager"
	"github.com/ovh/cds/engine/api/user"
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
		return WriteJSON(w, r, rms, http.StatusOK)
	}
}

func (api *API) addRepositoriesManagerHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		var args interface{}
		options := map[string]string{}

		if err := UnmarshalBody(r, &args); err != nil {
			return err
		}

		t := args.(map[string]interface{})["type"].(string)
		name := args.(map[string]interface{})["name"].(string)
		url := args.(map[string]interface{})["url"].(string)

		for k, v := range args.(map[string]interface{}) {
			if k != "type" && k != "name" && k != "url" {
				// example: for github, we need client-id here
				options[k] = v.(string)
			}
		}

		if t == "" || name == "" || url == "" {
			return sdk.WrapError(sdk.ErrWrongRequest, "addRepositoriesManagerHandler> Bad request: type=%s name=%s url=%s", t, name, url)
		}

		rm, err := repositoriesmanager.New(sdk.RepositoriesManagerType(t), 0, name, url, options, "", api.Cache)
		if err != nil {
			return sdk.WrapError(err, "addRepositoriesManagerHandler> cannot create %s")
		}
		if err := repositoriesmanager.Insert(api.mustDB(), rm); err != nil {
			return sdk.WrapError(err, "addRepositoriesManagerHandler> cannot insert %s")
		}
		return WriteJSON(w, r, rm, http.StatusCreated)
	}
}

func (api *API) getRepositoriesManagerForProjectHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars["permProjectKey"]

		rms, err := repositoriesmanager.LoadAllForProject(api.mustDB(), key, api.Cache)
		if err != nil {
			return sdk.WrapError(err, "getRepositoriesManagerForProjectHandler> error %s")
		}
		return WriteJSON(w, r, rms, http.StatusOK)
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

		//Load the repositories manager from the DB
		rm, errFind := repositoriesmanager.LoadForProject(api.mustDB(), proj.Key, rmName, api.Cache)
		var lastModified time.Time

		//If we don't find any repositories manager for the project, let's insert it
		if errFind == sql.ErrNoRows {
			var errLoad error
			rm, errLoad = repositoriesmanager.LoadByName(api.mustDB(), rmName, api.Cache)
			if errLoad != nil {
				return sdk.WrapError(sdk.ErrNoReposManager, "repositoriesManagerAuthorize> error while loading repositories manager %s", errLoad)
			}

			tx, err := api.mustDB().Begin()
			if err != nil {
				return sdk.WrapError(err, "repositoriesManagerAuthorize> Cannot start transaction")
			}
			defer tx.Rollback()

			if errI := repositoriesmanager.InsertForProject(tx, rm, proj.Key); errI != nil {
				return sdk.WrapError(errI, "repositoriesManagerAuthorize> error while inserting repositories manager for project %s", proj.Key)
			}

			if err := project.UpdateLastModified(tx, api.Cache, getUser(ctx), proj); err != nil {
				return sdk.WrapError(err, "repositoriesManagerAuthorize> Cannot update project last modified")
			}

			if err := tx.Commit(); err != nil {
				return sdk.WrapError(err, "repositoriesManagerAuthorize> Cannot commit transaction")
			}
		} else if errFind != nil {
			return sdk.WrapError(errFind, "repositoriesManagerAuthorize> error")
		}

		token, url, err := rm.Consumer.AuthorizeRedirect()
		if err != nil {
			return sdk.WrapError(sdk.ErrNoReposManagerAuth, "repositoriesManagerAuthorize> error with AuthorizeRedirect %s", err)
		}
		log.Info("repositoriesManagerAuthorize> [%s] RequestToken=%s; URL=%s", proj.Key, token, url)

		data := map[string]string{
			"project_key":          proj.Key,
			"last_modified":        strconv.FormatInt(lastModified.Unix(), 10),
			"repositories_manager": rmName,
			"url":           url,
			"request_token": token,
			"username":      getUser(ctx).Username,
		}

		api.Cache.Set(cache.Key("reposmanager", "oauth", token), data)
		return WriteJSON(w, r, data, http.StatusOK)
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

		api.Cache.Get(cache.Key("reposmanager", "oauth", state), &data)
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

		//Load the repositories manager from the DB
		rm, err := repositoriesmanager.LoadForProject(api.mustDB(), projectKey, rmName, api.Cache)
		if err != nil {
			log.Warning("repositoriesManagerAuthorizeCallback> error %s\n", err)
			return sdk.ErrNoReposManager

		}

		accessToken, accessTokenSecret, err := rm.Consumer.AuthorizeToken(state, code)
		if err != nil {
			log.Warning("repositoriesManagerAuthorizeCallback> Error with AuthorizeToken: %s", err)
			return sdk.ErrNoReposManagerClientAuth

		}

		log.Info("repositoriesManagerAuthorizeCallback> [%s] AccessToken=%s; AccessTokenSecret=%s", projectKey, accessToken, accessTokenSecret)
		result := map[string]string{
			"project_key":          projectKey,
			"repositories_manager": rmName,
			"access_token":         accessToken,
			"access_token_secret":  accessTokenSecret,
		}

		tx, errT := api.mustDB().Begin()
		if errT != nil {
			return sdk.WrapError(errT, "repositoriesManagerAuthorizeCallback> Cannot start transaction")
		}
		defer tx.Rollback()

		if err := repositoriesmanager.SaveDataForProject(tx, rm, projectKey, result); err != nil {
			return sdk.WrapError(err, "repositoriesManagerAuthorizeCallback> Error with SaveDataForProject")
		}

		if err := project.UpdateLastModified(tx, api.Cache, u, proj); err != nil {
			return sdk.WrapError(err, "repositoriesManagerAuthorizeCallback> Cannot update project last modified date")
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(errT, "repositoriesManagerAuthorizeCallback> Cannot commit transaction")
		}

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

		rm, errl := repositoriesmanager.LoadForProject(api.mustDB(), projectKey, rmName, api.Cache)
		if errl != nil {
			return sdk.WrapError(sdk.ErrNoReposManager, "repositoriesManagerAuthorizeCallback> Cannot find repository manager %s for project %s err:%s", rmName, projectKey, errl)
		}

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

		accessToken, accessTokenSecret, erra := rm.Consumer.AuthorizeToken(token, verifier)
		if erra != nil {
			return sdk.WrapError(sdk.ErrNoReposManagerClientAuth, "repositoriesManagerAuthorizeCallback> Error with AuthorizeToken: %s", erra)
		}

		log.Info("repositoriesManagerAuthorizeCallback> [%s] AccessToken=%s; AccessTokenSecret=%s\n", projectKey, accessToken, accessTokenSecret)
		result := map[string]string{
			"project_key":          projectKey,
			"repositories_manager": rmName,
			"access_token":         accessToken,
			"access_token_secret":  accessTokenSecret,
		}

		if err := repositoriesmanager.SaveDataForProject(api.mustDB(), rm, projectKey, result); err != nil {
			return sdk.WrapError(err, "repositoriesManagerAuthorizeCallback> Error with SaveDataForProject")
		}

		p, err := project.Load(api.mustDB(), api.Cache, projectKey, getUser(ctx), project.LoadOptions.WithRepositoriesManagers)
		if err != nil {
			return sdk.WrapError(err, "repositoriesManagerAuthorizeCallback> Cannot load project %s", projectKey)
		}

		return WriteJSON(w, r, p, http.StatusOK)
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
		rm, errlp := repositoriesmanager.LoadForProject(api.mustDB(), projectKey, rmName, api.Cache)
		if errlp != nil {
			return sdk.WrapError(sdk.ErrNoReposManager, "deleteRepositoriesManagerHandler> error loading %s-%s: %s", projectKey, rmName, errlp)
		}

		tx, errb := api.mustDB().Begin()
		if errb != nil {
			return sdk.WrapError(errb, "deleteRepositoriesManagerHandler> Cannot start transaction")
		}
		defer tx.Rollback()

		if err := repositoriesmanager.DeleteForProject(tx, rm, p); err != nil {
			return sdk.WrapError(err, "deleteRepositoriesManagerHandler> error deleting %s-%s", projectKey, rmName)
		}

		if err := project.UpdateLastModified(tx, api.Cache, getUser(ctx), p); err != nil {
			return sdk.WrapError(err, "deleteRepositoriesManagerHandler> Cannot update project last modified date")
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "deleteRepositoriesManagerHandler> Cannot commit transaction")
		}

		var errla error
		p.ReposManager, errla = repositoriesmanager.LoadAllForProject(api.mustDB(), p.Key, api.Cache)
		if errla != nil {
			return sdk.WrapError(errla, "deleteRepositoriesManagerHandler> Cannot load repos manager for project %s", p.Key)
		}

		return WriteJSON(w, r, p, http.StatusOK)
	}
}

func (api *API) getReposFromRepositoriesManagerHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		projectKey := vars["permProjectKey"]
		rmName := vars["name"]
		sync := FormBool(r, "synchronize")

		client, err := repositoriesmanager.AuthorizedClient(api.mustDB(), projectKey, rmName, api.Cache)
		if err != nil {
			return sdk.WrapError(sdk.ErrNoReposManagerClientAuth, "getReposFromRepositoriesManagerHandler> Cannot get client got %s %s", projectKey, rmName)
		}

		cacheKey := cache.Key("reposmanager", "repos", projectKey, rmName)
		if sync {
			api.Cache.Delete(cacheKey)
		}

		var repos []sdk.VCSRepo
		if !api.Cache.Get(cacheKey, &repos) {
			log.Debug("getReposFromRepositoriesManagerHandler> loading from Stash")
			repos, err = client.Repos()
			api.Cache.SetWithTTL(cacheKey, repos, 0)
		}
		if err != nil {
			return sdk.WrapError(err, "getReposFromRepositoriesManagerHandler> Cannot get repos")

		}
		return WriteJSON(w, r, repos, http.StatusOK)
	}
}

func (api *API) getRepoFromRepositoriesManagerHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		// Get project name in URL
		vars := mux.Vars(r)
		projectKey := vars["permProjectKey"]
		rmName := vars["name"]
		repoName := r.FormValue("repo")

		client, err := repositoriesmanager.AuthorizedClient(api.mustDB(), projectKey, rmName, api.Cache)
		if err != nil {
			return sdk.WrapError(sdk.ErrNoReposManagerClientAuth, "getRepoFromRepositoriesManagerHandler> Cannot get client got %s %s : %s", projectKey, rmName, err)
		}
		repo, err := client.RepoByFullname(repoName)
		if err != nil {
			return sdk.WrapError(err, "getRepoFromRepositoriesManagerHandler> Cannot get repos")
		}
		return WriteJSON(w, r, repo, http.StatusOK)
	}
}

func (api *API) attachRepositoriesManagerHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		projectKey := vars["key"]
		appName := vars["permApplicationName"]
		rmName := vars["name"]
		fullname := r.FormValue("fullname")

		app, err := application.LoadByName(api.mustDB(), api.Cache, projectKey, appName, getUser(ctx))
		if err != nil {
			return sdk.WrapError(err, "attachRepositoriesManager> Cannot load application %s", appName)
		}

		//Load the repositoriesManager for the project
		rm, err := repositoriesmanager.LoadForProject(api.mustDB(), projectKey, rmName, api.Cache)
		if err != nil {
			return sdk.WrapError(sdk.ErrNoReposManager, "attachRepositoriesManager> error loading %s-%s: %s", projectKey, rmName, err)
		}

		//Get an authorized Client
		client, err := repositoriesmanager.AuthorizedClient(api.mustDB(), projectKey, rmName, api.Cache)
		if err != nil {
			return sdk.WrapError(sdk.ErrNoReposManagerClientAuth, "attachRepositoriesManager> Cannot get client got %s %s : %s", projectKey, rmName, err)
		}

		if _, err := client.RepoByFullname(fullname); err != nil {
			return sdk.WrapError(sdk.ErrRepoNotFound, "attachRepositoriesManager> Cannot get repo %s: %s", fullname, err)
		}

		app.RepositoriesManager = rm
		app.RepositoryFullname = fullname

		tx, errT := api.mustDB().Begin()
		if errT != nil {
			return sdk.WrapError(errT, "attachRepositoriesManager> Cannot start transaction")
		}
		defer tx.Rollback()

		if err := repositoriesmanager.InsertForApplication(tx, app, projectKey); err != nil {
			return sdk.WrapError(err, "attachRepositoriesManager> Cannot insert for application")
		}

		if err := application.UpdateLastModified(tx, api.Cache, app, getUser(ctx)); err != nil {
			return sdk.WrapError(err, "attachRepositoriesManager> Cannot update application last modified date")
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "attachRepositoriesManager> Cannot commit transaction")
		}

		return WriteJSON(w, r, app, http.StatusOK)
	}
}

func (api *API) detachRepositoriesManagerHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		projectKey := vars["key"]
		appName := vars["permApplicationName"]
		rmName := vars["name"]

		app, errl := application.LoadByName(api.mustDB(), api.Cache, projectKey, appName, getUser(ctx), application.LoadOptions.WithHooks)
		if errl != nil {
			return sdk.WrapError(errl, "detachRepositoriesManager> error on load project %s", projectKey)
		}

		client, erra := repositoriesmanager.AuthorizedClient(api.mustDB(), projectKey, rmName, api.Cache)
		if erra != nil {
			return sdk.WrapError(sdk.ErrNoReposManagerClientAuth, "detachRepositoriesManager> Cannot get client got %s %s: %s", projectKey, rmName, erra)
		}

		//Remove all the things in a transaction
		tx, errT := api.mustDB().Begin()
		if errT != nil {
			return sdk.WrapError(errT, "detachRepositoriesManager> Cannot start transaction")
		}
		defer tx.Rollback()

		if err := repositoriesmanager.DeleteForApplication(tx, app); err != nil {
			return sdk.WrapError(err, "detachRepositoriesManager> Cannot delete for application")
		}

		for _, h := range app.Hooks {
			s := api.Config.URL.API + hook.HookLink
			link := fmt.Sprintf(s, h.UID, h.Project, h.Repository)

			if err := client.DeleteHook(h.Project+"/"+h.Repository, link); err != nil {
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

		if err := application.UpdateLastModified(tx, api.Cache, app, getUser(ctx)); err != nil {
			return sdk.WrapError(err, "detachRepositoriesManager> Cannot update application last modified date")
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "detachRepositoriesManager> Cannot commit transaction")
		}

		return WriteJSON(w, r, app, http.StatusOK)
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

		app, errla := application.LoadByName(api.mustDB(), api.Cache, projectKey, appName, getUser(ctx))
		if errla != nil {
			return sdk.ErrApplicationNotFound
		}

		pipeline, errl := pipeline.LoadPipeline(api.mustDB(), projectKey, pipelineName, false)
		if errl != nil {
			return sdk.ErrPipelineNotFound
		}

		if !permission.AccessToPipeline(sdk.DefaultEnv.ID, pipeline.ID, getUser(ctx), permission.PermissionReadWriteExecute) {
			return sdk.WrapError(sdk.ErrForbidden, "addHookOnRepositoriesManagerHandler> You don't have enought right on this pipeline %s", pipeline.Name)
		}

		rm, errlp := repositoriesmanager.LoadForProject(api.mustDB(), projectKey, rmName, api.Cache)
		if errlp != nil {
			return sdk.WrapError(sdk.ErrNoReposManager, "addHookOnRepositoriesManagerHandler> error loading %s-%s: %s", projectKey, rmName, errlp)
		}

		b, e := repositoriesmanager.CheckApplicationIsAttached(api.mustDB(), rmName, projectKey, appName)
		if e != nil {
			return sdk.WrapError(e, "addHookOnRepositoriesManagerHandler> Cannot check app (%s,%s,%s)", rmName, projectKey, appName)
		}

		if !b {
			return sdk.ErrNoReposManagerClientAuth
		}

		tx, errb := api.mustDB().Begin()
		if errb != nil {
			return sdk.WrapError(errb, "addHookOnRepositoriesManagerHandler> cannot start transaction")
		}
		defer tx.Rollback()

		if _, err := hook.CreateHook(tx, api.Cache, projectKey, rm, repoFullname, app, pipeline); err != nil {
			return sdk.WrapError(err, "addHookOnRepositoriesManagerHandler> cannot create hook")
		}

		if err := application.UpdateLastModified(tx, api.Cache, app, getUser(ctx)); err != nil {
			return sdk.WrapError(err, "addHookOnRepositoriesManagerHandler> cannot update application last modified date")
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
		app.Workflows, errW = workflowv0.LoadCDTree(api.mustDB(), api.Cache, projectKey, app.Name, getUser(ctx), "", 0)
		if errW != nil {
			return sdk.WrapError(errW, "addHookOnRepositoriesManagerHandler> Cannot load workflow")
		}

		return WriteJSON(w, r, app, http.StatusCreated)
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

		if errupdate := application.UpdateLastModified(tx, api.Cache, app, getUser(ctx)); errupdate != nil {
			return sdk.WrapError(errupdate, "deleteHookOnRepositoriesManagerHandler> Unable to update last modified")
		}

		if errtx := tx.Commit(); errtx != nil {
			return sdk.WrapError(errtx, "deleteHookOnRepositoriesManagerHandler> Unable to commit transaction")
		}

		var errW error
		app.Workflows, errW = workflowv0.LoadCDTree(api.mustDB(), api.Cache, projectKey, app.Name, getUser(ctx), "", 0)
		if errW != nil {
			return sdk.WrapError(errW, "deleteHookOnRepositoriesManagerHandler> Unable to load workflow")
		}

		var errR error
		_, app.RepositoriesManager, errR = repositoriesmanager.LoadFromApplicationByID(api.mustDB(), app.ID, api.Cache)
		if errR != nil {
			return sdk.WrapError(errR, "deleteHookOnRepositoriesManagerHandler> Cannot load repository manager from application %s", appName)
		}

		client, errauth := repositoriesmanager.AuthorizedClient(api.mustDB(), projectKey, app.RepositoriesManager.Name, api.Cache)
		if errauth != nil {
			return sdk.WrapError(errauth, "deleteHookOnRepositoriesManagerHandler> Cannot get client %s %s", projectKey, app.RepositoriesManager.Name)
		}

		t := strings.Split(app.RepositoryFullname, "/")
		if len(t) != 2 {
			return sdk.WrapError(sdk.ErrRepoNotFound, "deleteHookOnRepositoriesManagerHandler> Application %s repository fullname is not valid %s", app.Name, app.RepositoryFullname)
		}

		s := api.Config.URL.API + hook.HookLink
		link := fmt.Sprintf(s, h.UID, t[0], t[1])

		if errdelete := client.DeleteHook(app.RepositoryFullname, link); errdelete != nil {
			return sdk.WrapError(errdelete, "deleteHookOnRepositoriesManagerHandler> Cannot delete hook on stash")
		}

		return WriteJSON(w, r, app, http.StatusOK)
	}
}

func (api *API) addApplicationFromRepositoriesManagerHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		projectKey := vars["permProjectKey"]
		rmName := vars["name"]

		var data map[string]string
		if err := UnmarshalBody(r, &data); err != nil {
			return err
		}

		repoFullname := data["repository_fullname"]
		if repoFullname == "" {
			return sdk.WrapError(sdk.ErrWrongRequest, "addApplicationFromRepositoriesManagerHandler>Repository fullname is mandatory")
		}

		proj, errlp := project.Load(api.mustDB(), api.Cache, projectKey, getUser(ctx))
		if errlp != nil {
			return sdk.WrapError(sdk.ErrInvalidProject, "addApplicationFromRepositoriesManagerHandler: Cannot load %s: %s", projectKey, errlp)
		}

		rm, errlrm := repositoriesmanager.LoadForProject(api.mustDB(), projectKey, rmName, api.Cache)
		if errlrm != nil {
			return sdk.WrapError(sdk.ErrNoReposManager, "addApplicationFromRepositoriesManagerHandler> error loading %s-%s: %s", projectKey, rmName, errlrm)
		}

		client, errac := repositoriesmanager.AuthorizedClient(api.mustDB(), projectKey, rmName, api.Cache)
		if errac != nil {
			return sdk.WrapError(sdk.ErrNoReposManagerClientAuth, "addApplicationFromRepositoriesManagerHandler> Cannot get client got %s %s: %s", projectKey, rmName, errac)
		}

		repo, errlr := client.RepoByFullname(repoFullname)
		if errlr != nil {
			return sdk.WrapError(sdk.ErrRepoNotFound, "addApplicationFromRepositoriesManagerHandler> Cannot get repo: %s", errlr)
		}

		app := sdk.Application{
			Name:       repo.Slug,
			ProjectKey: projectKey,
			Variable: []sdk.Variable{
				sdk.Variable{
					Name:  "repo",
					Type:  sdk.StringVariable,
					Value: repo.SSHCloneURL,
				},
			},
		}

		tx, errb := api.mustDB().Begin()
		if errb != nil {
			return sdk.WrapError(errb, "addApplicationFromRepositoriesManagerHandler> Cannot start transaction")
		}

		defer tx.Rollback()

		//Insert application in database
		if err := application.Insert(tx, api.Cache, proj, &app, getUser(ctx)); err != nil {
			return sdk.WrapError(err, "addApplicationFromRepositoriesManagerHandler> Cannot insert pipeline")
		}

		//Fetch groups from project
		if err := group.LoadGroupByProject(tx, proj); err != nil {
			return sdk.WrapError(err, "addApplicationFromRepositoriesManagerHandler> Cannot load group from project")
		}

		//Add the  groups on the application
		if err := application.AddGroup(tx, api.Cache, proj, &app, getUser(ctx), proj.ProjectGroups...); err != nil {
			return sdk.WrapError(err, "addApplicationFromRepositoriesManagerHandler> Cannot add groups on application")
		}

		//Commit the transaction
		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "addApplicationFromRepositoriesManagerHandler> Cannot commit transaction")
		}

		//Attach the application to the repositories manager
		app.RepositoriesManager = rm
		app.RepositoryFullname = repoFullname
		if err := repositoriesmanager.InsertForApplication(api.mustDB(), &app, projectKey); err != nil {
			return sdk.WrapError(err, "addApplicationFromRepositoriesManagerHandler> Cannot attach application")
		}

		return nil
	}
}
