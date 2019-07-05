package api

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/event"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/repositoriesmanager"
	"github.com/ovh/cds/engine/api/user"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func (api *API) getRepositoriesManagerHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vcsServers, err := repositoriesmanager.LoadAll(ctx, api.mustDB(), api.Cache)
		if err != nil {
			return sdk.WrapError(err, "error")
		}
		rms := make([]string, 0, len(vcsServers))
		for k := range vcsServers {
			rms = append(rms, k)
		}
		return service.WriteJSON(w, rms, http.StatusOK)
	}
}

func (api *API) getRepositoriesManagerForProjectHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars[permProjectKey]

		proj, errproj := project.Load(api.mustDB(), api.Cache, key, deprecatedGetUser(ctx))
		if errproj != nil {
			return errproj
		}

		return service.WriteJSON(w, proj.VCSServers, http.StatusOK)
	}
}

func (api *API) repositoriesManagerAuthorizeHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars[permProjectKey]
		rmName := vars["name"]

		proj, errP := project.Load(api.mustDB(), api.Cache, key, deprecatedGetUser(ctx))
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

		token, url, err := vcsServer.AuthorizeRedirect(ctx)
		if err != nil {
			return sdk.WrapError(sdk.ErrNoReposManagerAuth, "repositoriesManagerAuthorize> error with AuthorizeRedirect %s", err)
		}
		log.Debug("repositoriesManagerAuthorize> [%s] RequestToken=%s; URL=%s", proj.Key, token, url)

		data := map[string]string{
			"project_key":          proj.Key,
			"last_modified":        strconv.FormatInt(time.Now().Unix(), 10),
			"repositories_manager": rmName,
			"url":                  url,
			"request_token":        token,
			"username":             deprecatedGetUser(ctx).Username,
		}

		if token != "" {
			data["auth_type"] = "oauth"
		} else {
			data["auth_type"] = "basic"
		}

		api.Cache.Set(cache.Key("reposmanager", "oauth", token), data)
		return service.WriteJSON(w, data, http.StatusOK)
	}
}

func (api *API) repositoriesManagerOAuthCallbackHandler() service.Handler {
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

		token, secret, err := vcsServer.AuthorizeToken(ctx, state, code)
		if err != nil {
			return sdk.WrapError(sdk.ErrNoReposManagerClientAuth, "repositoriesManagerAuthorizeCallback> Error with AuthorizeToken: %s", err)

		}

		if token == "" || secret == "" {
			return sdk.WrapError(sdk.ErrNoReposManagerClientAuth, "repositoriesManagerAuthorizeCallback> token or secret is empty")
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
				"token":   token,
				"secret":  secret,
				"created": fmt.Sprintf("%d", time.Now().Unix()),
			},
		}

		if err := repositoriesmanager.InsertForProject(tx, proj, vcsServerForProject); err != nil {
			return sdk.WrapError(err, "Error with InsertForProject")
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(errT, "repositoriesManagerAuthorizeCallback> Cannot commit transaction")
		}

		event.PublishAddVCSServer(proj, vcsServerForProject.Name, deprecatedGetUser(ctx))

		//Redirect on UI advanced project page
		url := fmt.Sprintf("%s/project/%s?tab=advanced", api.Config.URL.UI, projectKey)
		http.Redirect(w, r, url, http.StatusTemporaryRedirect)

		return nil
	}
}

func (api *API) repositoriesManagerAuthorizeBasicHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		projectKey := vars["permProjectKey"]
		rmName := vars["name"]

		var tv map[string]interface{}
		if err := service.UnmarshalBody(r, &tv); err != nil {
			return err
		}

		var username, secret string
		if tv["username"] != nil {
			username = tv["username"].(string)
		}
		if tv["secret"] != nil {
			secret = tv["secret"].(string)
		}

		if username == "" || secret == "" {
			return sdk.WrapError(sdk.ErrWrongRequest, "cannot get token nor verifier from data")
		}

		proj, errP := project.Load(api.mustDB(), api.Cache, projectKey, deprecatedGetUser(ctx))
		if errP != nil {
			return sdk.WrapError(errP, "cannot load project %s", projectKey)
		}

		tx, errT := api.mustDB().Begin()
		if errT != nil {
			return sdk.WrapError(errT, "cannot start transaction")
		}
		defer tx.Rollback() // nolint

		vcsServerForProject := &sdk.ProjectVCSServer{
			Name:     rmName,
			Username: deprecatedGetUser(ctx).Username,
			Data: map[string]string{
				"token":   username,
				"secret":  secret,
				"created": fmt.Sprintf("%d", time.Now().Unix()),
			},
		}

		if err := repositoriesmanager.InsertForProject(tx, proj, vcsServerForProject); err != nil {
			return sdk.WrapError(err, "unable to set repository manager data for project %s", projectKey)
		}

		client, err := repositoriesmanager.AuthorizedClient(ctx, tx, api.Cache, proj.Key, vcsServerForProject)
		if err != nil {
			return sdk.WrapError(sdk.ErrNoReposManagerClientAuth, "cannot get client for project %s: %v", proj.Key, err)
		}

		if _, err = client.Repos(ctx); err != nil {
			return sdk.WrapError(err, "unable to connect %s to %s", proj.Key, rmName)
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "cannot commit transaction")
		}

		event.PublishAddVCSServer(proj, vcsServerForProject.Name, deprecatedGetUser(ctx))

		return service.WriteJSON(w, proj, http.StatusOK)

	}
}

func (api *API) repositoriesManagerAuthorizeCallbackHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		projectKey := vars[permProjectKey]
		rmName := vars["name"]

		var tv map[string]interface{}
		if err := service.UnmarshalBody(r, &tv); err != nil {
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

		proj, errP := project.Load(api.mustDB(), api.Cache, projectKey, deprecatedGetUser(ctx))
		if errP != nil {
			return sdk.WrapError(errP, "repositoriesManagerAuthorizeCallback> Cannot load project")
		}

		vcsServer, errVCSServer := repositoriesmanager.NewVCSServerConsumer(api.mustDB, api.Cache, rmName)
		if errVCSServer != nil {
			return sdk.WrapError(errVCSServer, "repositoriesManagerAuthorizeCallback> Cannot create VCS Server Consumer project:%s repoManager:%s", proj.Key, rmName)
		}

		tx, errT := api.mustDB().Begin()
		if errT != nil {
			return sdk.WrapError(errT, "repositoriesManagerAuthorizeCallback> Cannot start transaction")
		}
		defer tx.Rollback()

		token, secret, err := vcsServer.AuthorizeToken(ctx, token, verifier)
		if err != nil {
			return sdk.WrapError(sdk.ErrNoReposManagerClientAuth, "repositoriesManagerAuthorizeCallback> Error with AuthorizeToken: %s project:%s", err, proj.Key)
		}
		log.Debug("repositoriesManagerAuthorizeCallback> [%s] AccessToken=%s; AccessTokenSecret=%s", projectKey, token, secret)

		vcsServerForProject := &sdk.ProjectVCSServer{
			Name:     rmName,
			Username: deprecatedGetUser(ctx).Username,
			Data: map[string]string{
				"token":   token,
				"secret":  secret,
				"created": fmt.Sprintf("%d", time.Now().Unix()),
			},
		}

		if err := repositoriesmanager.InsertForProject(tx, proj, vcsServerForProject); err != nil {
			return sdk.WrapError(err, "Error with SaveDataForProject")
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(errT, "repositoriesManagerAuthorizeCallback> Cannot commit transaction")
		}

		event.PublishAddVCSServer(proj, vcsServerForProject.Name, deprecatedGetUser(ctx))

		return service.WriteJSON(w, proj, http.StatusOK)
	}
}

func (api *API) deleteRepositoriesManagerHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		projectKey := vars[permProjectKey]
		rmName := vars["name"]

		force := FormBool(r, "force")

		p, errl := project.Load(api.mustDB(), api.Cache, projectKey, deprecatedGetUser(ctx))
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

		if !force {
			// Check that the VCS is not used by an application before removing it
			apps, err := application.LoadAll(tx, api.Cache, projectKey)
			if err != nil {
				return err
			}
			for _, app := range apps {
				if app.VCSServer == rmName {
					return sdk.WithStack(sdk.ErrVCSUsedByApplication)
				}
			}
		}

		if err := repositoriesmanager.DeleteForProject(tx, p, vcsServer); err != nil {
			return sdk.WrapError(err, "error deleting %s-%s", projectKey, rmName)
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "Cannot commit transaction")
		}

		event.PublishDeleteVCSServer(p, vcsServer.Name, deprecatedGetUser(ctx))

		return service.WriteJSON(w, p, http.StatusOK)
	}
}

func (api *API) getReposFromRepositoriesManagerHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		projectKey := vars[permProjectKey]
		rmName := vars["name"]
		sync := FormBool(r, "synchronize")

		proj, errproj := project.Load(api.mustDB(), api.Cache, projectKey, deprecatedGetUser(ctx))
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
		client, errAuthClient := repositoriesmanager.AuthorizedClient(ctx, api.mustDB(), api.Cache, projectKey, vcsServer)
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
			repos, errRepos = client.Repos(ctx)
			api.Cache.SetWithTTL(cacheKey, repos, 0)
			if errRepos != nil {
				return sdk.WrapError(errRepos, "getReposFromRepositoriesManagerHandler> Cannot get repos: %v", errRepos)
			}
		}

		return service.WriteJSON(w, repos, http.StatusOK)
	}
}

func (api *API) getRepoFromRepositoriesManagerHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		// Get project name in URL
		vars := mux.Vars(r)
		projectKey := vars[permProjectKey]
		rmName := vars["name"]
		repoName := r.FormValue("repo")

		if repoName == "" {
			return sdk.NewError(sdk.ErrWrongRequest, fmt.Errorf("Missing repository name 'repo' as a query parameter"))
		}

		proj, errproj := project.Load(api.mustDB(), api.Cache, projectKey, deprecatedGetUser(ctx))
		if errproj != nil {
			return sdk.WrapError(sdk.ErrNoReposManagerClientAuth, "getReposFromRepositoriesManagerHandler> Cannot get client got %s %s", projectKey, rmName)
		}

		vcsServer := repositoriesmanager.GetProjectVCSServer(proj, rmName)
		if vcsServer == nil {
			return sdk.WrapError(sdk.ErrNoReposManagerClientAuth, "getReposFromRepositoriesManagerHandler> Cannot get client got %s %s", projectKey, rmName)
		}

		client, err := repositoriesmanager.AuthorizedClient(ctx, api.mustDB(), api.Cache, projectKey, vcsServer)
		if err != nil {
			return sdk.WrapError(sdk.ErrNoReposManagerClientAuth, "getRepoFromRepositoriesManagerHandler> Cannot get client got %s %s : %s", projectKey, rmName, err)
		}

		log.Info("getRepoFromRepositoriesManagerHandler> Loading repository on %s", vcsServer.Name)

		repo, err := client.RepoByFullname(ctx, repoName)
		if err != nil {
			return sdk.WrapError(err, "Cannot get repos")
		}
		return service.WriteJSON(w, repo, http.StatusOK)
	}
}

func (api *API) attachRepositoriesManagerHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		projectKey := vars[permProjectKey]
		appName := vars["applicationName"]
		rmName := vars["name"]
		fullname := r.FormValue("fullname")
		db := api.mustDB()
		u := deprecatedGetUser(ctx)

		app, err := application.LoadByName(db, api.Cache, projectKey, appName)
		if err != nil {
			return sdk.WrapError(err, "Cannot load application %s", appName)
		}

		//Load the repositoriesManager for the project
		rm, err := repositoriesmanager.LoadForProject(db, projectKey, rmName)
		if err != nil {
			return sdk.WrapError(sdk.ErrNoReposManager, "attachRepositoriesManager> error loading %s-%s: %s", projectKey, rmName, err)
		}

		//Get an authorized Client
		client, err := repositoriesmanager.AuthorizedClient(ctx, db, api.Cache, projectKey, rm)
		if err != nil {
			return sdk.WrapError(sdk.ErrNoReposManagerClientAuth, "attachRepositoriesManager> Cannot get client got %s %s : %s", projectKey, rmName, err)
		}

		if _, err := client.RepoByFullname(ctx, fullname); err != nil {
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
			return sdk.WrapError(err, "Cannot insert for application")
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "Cannot commit transaction")
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
				wfDB, errWL := workflow.LoadByID(db, api.Cache, proj, wf.ID, u, workflow.LoadOptions{})
				if errWL != nil {
					return errWL
				}

				wfOld, errWL := workflow.LoadByID(db, api.Cache, proj, wf.ID, u, workflow.LoadOptions{})
				if errWL != nil {
					return errWL
				}

				if wfDB.WorkflowData.Node.Context == nil {
					wfDB.WorkflowData.Node.Context = &sdk.NodeContext{}
				}
				if wfDB.WorkflowData.Node.Context.ApplicationID != app.ID {
					continue
				}

				payload, errD := wfDB.WorkflowData.Node.Context.DefaultPayloadToMap()
				if errD != nil {
					return sdk.WrapError(errP, "attachRepositoriesManager> Cannot DefaultPayloadToMap")
				}

				if _, ok := payload["git.branch"]; ok && payload["git.repository"] == app.RepositoryFullname {
					continue
				}

				defaultPayload, errPay := workflow.DefaultPayload(ctx, db, api.Cache, proj, wfDB)
				if errPay != nil {
					return sdk.WrapError(errPay, "attachRepositoriesManager> Cannot get defaultPayload")
				}
				wfDB.WorkflowData.Node.Context.DefaultPayload = defaultPayload

				if err := workflow.Update(ctx, db, api.Cache, wfDB, proj, u, workflow.UpdateOptions{DisableHookManagement: true}); err != nil {
					return sdk.WrapError(err, "Cannot update node context %d", wf.WorkflowData.Node.Context.ID)
				}

				event.PublishWorkflowUpdate(proj.Key, *wfDB, *wfOld, u)

			}
		}

		event.PublishApplicationRepositoryAdd(projectKey, *app, u)

		return service.WriteJSON(w, app, http.StatusOK)
	}
}

func (api *API) detachRepositoriesManagerHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		projectKey := vars[permProjectKey]
		appName := vars["applicationName"]
		db := api.mustDB()
		u := deprecatedGetUser(ctx)

		app, errl := application.LoadByName(db, api.Cache, projectKey, appName)
		if errl != nil {
			return sdk.WrapError(errl, "detachRepositoriesManager> error on load project %s", projectKey)
		}

		// Check if there is hooks on this application
		hooksCount, err := workflow.CountHooksByApplication(db, app.ID)
		if err != nil {
			return err
		}
		if hooksCount > 0 {
			return sdk.WithStack(sdk.ErrRepositoryUsedByHook)
		}

		//Remove all the things in a transaction
		tx, errT := db.Begin()
		if errT != nil {
			return sdk.WrapError(errT, "detachRepositoriesManager> Cannot start transaction")
		}
		defer tx.Rollback()

		if err := repositoriesmanager.DeleteForApplication(tx, app); err != nil {
			return sdk.WrapError(err, "Cannot delete for application")
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "Cannot commit transaction")
		}

		event.PublishApplicationRepositoryDelete(projectKey, appName, app.VCSServer, app.RepositoryFullname, u)

		return service.WriteJSON(w, app, http.StatusOK)
	}
}
