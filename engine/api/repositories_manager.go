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

		proj, errproj := project.Load(api.mustDB(), api.Cache, key)
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

		proj, err := project.Load(api.mustDB(), api.Cache, key)
		if err != nil {
			return sdk.WrapError(err, "cannot load project")
		}

		if repositoriesmanager.GetProjectVCSServer(*proj, rmName) != nil {
			return sdk.WithStack(fmt.Errorf("cannot load project vcs server"))
		}

		vcsServer, err := repositoriesmanager.NewVCSServerConsumer(api.mustDBWithCtx, api.Cache, rmName)
		if err != nil {
			return sdk.WrapError(err, "cannot start transaction")
		}

		token, url, err := vcsServer.AuthorizeRedirect(ctx)
		if err != nil {
			return sdk.WrapError(sdk.ErrNoReposManagerAuth, "error with AuthorizeRedirect %s", err)
		}

		data := map[string]string{
			"project_key":          proj.Key,
			"last_modified":        strconv.FormatInt(time.Now().Unix(), 10),
			"repositories_manager": rmName,
			"url":                  url,
			"request_token":        token,
			"username":             getAPIConsumer(ctx).AuthentifiedUser.Username,
		}

		if token != "" {
			data["auth_type"] = "oauth"
		} else {
			data["auth_type"] = "basic"
		}

		keyr := cache.Key("reposmanager", "oauth", token)
		if err := api.Cache.Set(keyr, data); err != nil {
			log.Error(ctx, "unable to cache set %v: %v", keyr, err)
		}
		return service.WriteJSON(w, data, http.StatusOK)
	}
}

func (api *API) repositoriesManagerOAuthCallbackHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		cberr := r.FormValue("error")
		errDescription := r.FormValue("error_description")
		errURI := r.FormValue("error_uri")

		if cberr != "" {
			log.Error(ctx, "Callback Error: %s - %s - %s", cberr, errDescription, errURI)
			return fmt.Errorf("OAuth Error %s", cberr)
		}

		code := r.FormValue("code")
		state := r.FormValue("state")

		data := map[string]string{}

		key := cache.Key("reposmanager", "oauth", state)
		find, err := api.Cache.Get(key, &data)
		if err != nil {
			log.Error(ctx, "cannot get from cache %s: %v", key, err)
		}
		if !find {
			return sdk.WrapError(sdk.ErrForbidden, "repositoriesManagerAuthorizeCallback> Error")
		}
		projectKey := data["project_key"]
		rmName := data["repositories_manager"]
		username := data["username"]

		//u, err := user.LoadByUsername(api.mustDB(), username)
		//if err != nil {
		//	return sdk.WrapError(err, "repositoriesManagerAuthorizeCallback> Cannot load user %s", username)
		//}

		proj, errP := project.Load(api.mustDB(), api.Cache, projectKey)
		if errP != nil {
			return sdk.WrapError(errP, "repositoriesManagerAuthorizeCallback> Cannot load project")
		}

		vcsServer, errVCSServer := repositoriesmanager.NewVCSServerConsumer(api.mustDBWithCtx, api.Cache, rmName)
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
		defer tx.Rollback() // nolint

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

		event.PublishAddVCSServer(ctx, proj, vcsServerForProject.Name, getAPIConsumer(ctx))

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

		proj, errP := project.Load(api.mustDB(), api.Cache, projectKey)
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
			Username: getAPIConsumer(ctx).AuthentifiedUser.Username,
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

		event.PublishAddVCSServer(ctx, proj, vcsServerForProject.Name, getAPIConsumer(ctx))

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

		proj, errP := project.Load(api.mustDB(), api.Cache, projectKey)
		if errP != nil {
			return sdk.WrapError(errP, "repositoriesManagerAuthorizeCallback> Cannot load project")
		}

		vcsServer, errVCSServer := repositoriesmanager.NewVCSServerConsumer(api.mustDBWithCtx, api.Cache, rmName)
		if errVCSServer != nil {
			return sdk.WrapError(errVCSServer, "repositoriesManagerAuthorizeCallback> Cannot create VCS Server Consumer project:%s repoManager:%s", proj.Key, rmName)
		}

		tx, errT := api.mustDB().Begin()
		if errT != nil {
			return sdk.WrapError(errT, "repositoriesManagerAuthorizeCallback> Cannot start transaction")
		}
		defer tx.Rollback() // nolint

		token, secret, err := vcsServer.AuthorizeToken(ctx, token, verifier)
		if err != nil {
			return sdk.WrapError(sdk.ErrNoReposManagerClientAuth, "repositoriesManagerAuthorizeCallback> Error with AuthorizeToken: %s project:%s", err, proj.Key)
		}
		log.Debug("repositoriesManagerAuthorizeCallback> [%s] AccessToken=%s; AccessTokenSecret=%s", projectKey, token, secret)

		vcsServerForProject := &sdk.ProjectVCSServer{
			Name:     rmName,
			Username: getAPIConsumer(ctx).AuthentifiedUser.Username,
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

		event.PublishAddVCSServer(ctx, proj, vcsServerForProject.Name, getAPIConsumer(ctx))

		return service.WriteJSON(w, proj, http.StatusOK)
	}
}

func (api *API) deleteRepositoriesManagerHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		projectKey := vars[permProjectKey]
		rmName := vars["name"]

		force := FormBool(r, "force")

		p, err := project.Load(api.mustDB(), api.Cache, projectKey)
		if err != nil {
			return sdk.WrapError(err, "cannot load project %s", projectKey)
		}

		// Load the repositories manager from the DB
		vcsServer := repositoriesmanager.GetProjectVCSServer(*p, rmName)
		if vcsServer == nil {
			return sdk.WithStack(sdk.ErrRepoNotFound)
		}

		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WrapError(err, "cannot start transaction")
		}
		defer tx.Rollback() // nolint

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

		event.PublishDeleteVCSServer(ctx, p, vcsServer.Name, getAPIConsumer(ctx))

		return service.WriteJSON(w, p, http.StatusOK)
	}
}

func (api *API) getReposFromRepositoriesManagerHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		projectKey := vars[permProjectKey]
		rmName := vars["name"]
		sync := FormBool(r, "synchronize")

		proj, err := project.Load(api.mustDB(), api.Cache, projectKey)
		if err != nil {
			return sdk.NewErrorWithStack(err,
				sdk.NewErrorFrom(sdk.ErrNoReposManagerClientAuth, "cannot get client got %s %s", projectKey, rmName))
		}

		log.Debug("getReposFromRepositoriesManagerHandler> Loading repo for %s", rmName)

		vcsServer := repositoriesmanager.GetProjectVCSServer(*proj, rmName)
		if vcsServer == nil {
			return sdk.WrapError(sdk.ErrNoReposManagerClientAuth, "cannot get client got %s %s", projectKey, rmName)
		}

		log.Debug("getReposFromRepositoriesManagerHandler> Loading repo for %s; ok", vcsServer.Name)

		client, err := repositoriesmanager.AuthorizedClient(ctx, api.mustDB(), api.Cache, projectKey, vcsServer)
		if err != nil {
			return sdk.NewErrorWithStack(err, sdk.NewErrorFrom(sdk.ErrNoReposManagerClientAuth,
				"cannot get client got %s %s", projectKey, rmName))
		}

		cacheKey := cache.Key("reposmanager", "repos", projectKey, rmName)
		if sync {
			if err := api.Cache.Delete(cacheKey); err != nil {
				log.Error(ctx, "getReposFromRepositoriesManagerHandler> error on delete cache key %v: %s", cacheKey, err)
			}
		}

		var repos []sdk.VCSRepo
		find, err := api.Cache.Get(cacheKey, &repos)
		if err != nil {
			log.Error(ctx, "cannot get from cache %s: %v", cacheKey, err)
		}
		if !find || len(repos) == 0 {
			repos, err = client.Repos(ctx)
			if err != nil {
				return sdk.WrapError(err, "cannot get repos")
			}
			if err := api.Cache.SetWithTTL(cacheKey, repos, 0); err != nil {
				log.Error(ctx, "cannot SetWithTTL: %s: %v", cacheKey, err)
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

		proj, err := project.Load(api.mustDB(), api.Cache, projectKey)
		if err != nil {
			return sdk.NewErrorWithStack(err, sdk.NewErrorFrom(sdk.ErrNoReposManagerClientAuth,
				"cannot get client got %s %s", projectKey, rmName))
		}

		vcsServer := repositoriesmanager.GetProjectVCSServer(*proj, rmName)
		if vcsServer == nil {
			return sdk.WrapError(sdk.ErrNoReposManagerClientAuth, "cannot get client got %s %s", projectKey, rmName)
		}

		client, err := repositoriesmanager.AuthorizedClient(ctx, api.mustDB(), api.Cache, projectKey, vcsServer)
		if err != nil {
			return sdk.NewErrorWithStack(err, sdk.NewErrorFrom(sdk.ErrNoReposManagerClientAuth,
				"cannot get client got %s %s", projectKey, rmName))
		}

		log.Info(ctx, "getRepoFromRepositoriesManagerHandler> Loading repository on %s", vcsServer.Name)

		repo, err := client.RepoByFullname(ctx, repoName)
		if err != nil {
			return sdk.WrapError(err, "cannot get repos")
		}

		return service.WriteJSON(w, repo, http.StatusOK)
	}
}

func (api *API) getRepositoriesManagerLinkedApplicationsHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		// Get project name in URL
		vars := mux.Vars(r)
		projectKey := vars[permProjectKey]
		rmName := vars["name"]

		proj, err := project.Load(api.mustDB(), api.Cache, projectKey)
		if err != nil {
			return sdk.WrapError(err, "cannot get client got %s %s", projectKey, rmName)
		}

		vcsServer := repositoriesmanager.GetProjectVCSServer(*proj, rmName)
		if vcsServer == nil {
			return sdk.WrapError(sdk.ErrNoReposManagerClientAuth, "cannot get client got %s %s", projectKey, rmName)
		}

		appNames, err := repositoriesmanager.LoadLinkedApplicationNames(api.mustDB(), proj.Key, rmName)
		if err != nil {
			return sdk.WrapError(err, "cannot load linked application names")
		}

		return service.WriteJSON(w, appNames, http.StatusOK)
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
		defer tx.Rollback() // nolint

		if err := repositoriesmanager.InsertForApplication(tx, app, projectKey); err != nil {
			return sdk.WrapError(err, "Cannot insert for application")
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "Cannot commit transaction")
		}

		usage, errU := loadApplicationUsage(ctx, db, projectKey, appName)
		if errU != nil {
			return sdk.WrapError(errU, "attachRepositoriesManager> Cannot load application usage")
		}

		// Update default payload of linked workflow root
		if len(usage.Workflows) > 0 {
			proj, errP := project.Load(db, api.Cache, projectKey, project.LoadOptions.WithIntegrations)
			if errP != nil {
				return sdk.WrapError(errP, "attachRepositoriesManager> Cannot load project")
			}

			for _, wf := range usage.Workflows {
				wfDB, err := workflow.LoadByID(ctx, db, api.Cache, *proj, wf.ID, workflow.LoadOptions{})
				if err != nil {
					return err
				}

				wfOld, err := workflow.LoadByID(ctx, db, api.Cache, *proj, wf.ID, workflow.LoadOptions{})
				if err != nil {
					return err
				}

				if wfDB.WorkflowData.Node.Context == nil {
					wfDB.WorkflowData.Node.Context = &sdk.NodeContext{}
				}
				if wfDB.WorkflowData.Node.Context.ApplicationID != app.ID {
					continue
				}

				payload, err := wfDB.WorkflowData.Node.Context.DefaultPayloadToMap()
				if err != nil {
					return sdk.WithStack(err)
				}

				if _, ok := payload["git.branch"]; ok && payload["git.repository"] == app.RepositoryFullname {
					continue
				}

				defaultPayload, err := workflow.DefaultPayload(ctx, db, api.Cache, *proj, wfDB)
				if err != nil {
					return sdk.WithStack(err)
				}

				wfDB.WorkflowData.Node.Context.DefaultPayload = defaultPayload

				if err := workflow.Update(ctx, db, api.Cache, *proj, wfDB, workflow.UpdateOptions{DisableHookManagement: true, OldWorkflowID: wfOld.ID}); err != nil {
					return sdk.WrapError(err, "cannot update node context %d", wf.WorkflowData.Node.Context.ID)
				}

				event.PublishWorkflowUpdate(ctx, proj.Key, *wfDB, *wfOld, getAPIConsumer(ctx))
			}
		}

		event.PublishApplicationRepositoryAdd(ctx, projectKey, *app, getAPIConsumer(ctx))

		return service.WriteJSON(w, app, http.StatusOK)
	}
}

func (api *API) detachRepositoriesManagerHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		projectKey := vars[permProjectKey]
		appName := vars["applicationName"]
		db := api.mustDB()
		u := getAPIConsumer(ctx)

		app, errl := application.LoadByName(db, api.Cache, projectKey, appName)
		if errl != nil {
			return sdk.WrapError(errl, "detachRepositoriesManager> error on load project %s", projectKey)
		}

		// Check if there is hooks on this application
		repositoryWebHooksCount, err := workflow.CountRepositoryWebHooksByApplication(db, app.ID)
		if err != nil {
			return err
		}
		if repositoryWebHooksCount > 0 {
			return sdk.WithStack(sdk.ErrRepositoryUsedByHook)
		}

		//Remove all the things in a transaction
		tx, errT := db.Begin()
		if errT != nil {
			return sdk.WrapError(errT, "detachRepositoriesManager> Cannot start transaction")
		}
		defer tx.Rollback() // nolint

		if err := repositoriesmanager.DeleteForApplication(tx, app); err != nil {
			return sdk.WrapError(err, "Cannot delete for application")
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "Cannot commit transaction")
		}

		event.PublishApplicationRepositoryDelete(ctx, projectKey, appName, app.VCSServer, app.RepositoryFullname, u)

		return service.WriteJSON(w, app, http.StatusOK)
	}
}
