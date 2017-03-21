package main

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-gorp/gorp"
	"github.com/gorilla/mux"
	"github.com/spf13/viper"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/context"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/hook"
	"github.com/ovh/cds/engine/api/permission"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/poller"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/repositoriesmanager"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/engine/log"
	"github.com/ovh/cds/sdk"
)

func getRepositoriesManagerHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	rms, err := repositoriesmanager.LoadAll(db)
	if err != nil {
		log.Warning("getRepositoriesManagerHandler> error %s\n", err)
		return err

	}
	return WriteJSON(w, r, rms, http.StatusOK)
}

func addRepositoriesManagerHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
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
			options[k] = v.(string)
		}
	}

	if t == "" || name == "" || url == "" {
		log.Warning("addProjectVCSHandler> Bad request : type=%s name=%s url=%s\n", t, name, url)
		return sdk.ErrWrongRequest

	}

	rm, err := repositoriesmanager.New(sdk.RepositoriesManagerType(t), 0, name, url, options, "")
	if err != nil {
		log.Warning("addRepositoriesManagerHandler> cannot create %s\n", err)
		return err

	}
	if err := repositoriesmanager.Insert(db, rm); err != nil {
		log.Warning("addRepositoriesManagerHandler> cannot insert %s\n", err)
		return err

	}
	return WriteJSON(w, r, rm, http.StatusCreated)
}

func getRepositoriesManagerForProjectHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	// Get project name in URL
	vars := mux.Vars(r)
	key := vars["permProjectKey"]

	rms, err := repositoriesmanager.LoadAllForProject(db, key)
	if err != nil {
		log.Warning("getRepositoriesManagerForProjectHandler> error %s\n", err)
		return err

	}
	return WriteJSON(w, r, rms, http.StatusOK)
}

func repositoriesManagerAuthorize(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	// Get project name in URL
	vars := mux.Vars(r)
	projectKey := vars["permProjectKey"]
	rmName := vars["name"]

	//Load the repositories manager from the DB
	rm, err := repositoriesmanager.LoadForProject(db, projectKey, rmName)
	var lastModified time.Time

	//If we don't find any repositories manager for the project, let's insert it
	if err == sql.ErrNoRows {
		var err error
		rm, err = repositoriesmanager.LoadByName(db, rmName)
		if err != nil {
			log.Warning("repositoriesManagerAuthorize> error while loading repositories manager %s\n", err)
			return sdk.ErrNoReposManager
		}

		tx, err := db.Begin()
		if err != nil {
			log.Warning("repositoriesManagerAuthorize> Cannot start transaction %s\n", err)
			return err
		}
		defer tx.Rollback()

		var errI error
		lastModified, errI = repositoriesmanager.InsertForProject(tx, rm, projectKey)
		if errI != nil {
			log.Warning("repositoriesManagerAuthorize> error while inserting repositories manager for project %s: %s\n", projectKey, errI)
			return errI
		}

		if err := tx.Commit(); err != nil {
			log.Warning("repositoriesManagerAuthorize> Cannot commit transaction %s\n", err)
			return err
		}
	} else if err != nil {
		log.Warning("repositoriesManagerAuthorize> error %s\n", err)
		return err
	}

	token, url, err := rm.Consumer.AuthorizeRedirect()
	if err != nil {
		log.Warning("repositoriesManagerAuthorize> error with AuthorizeRedirect %s\n", err)
		return sdk.ErrNoReposManagerAuth

	}
	log.Notice("repositoriesManagerAuthorize> [%s] RequestToken=%s; URL=%s\n", projectKey, token, url)

	data := map[string]string{
		"project_key":          projectKey,
		"last_modified":        strconv.FormatInt(lastModified.Unix(), 10),
		"repositories_manager": rmName,
		"url":           url,
		"request_token": token,
	}

	cache.Set(cache.Key("reposmanager", "oauth", token), data)

	return WriteJSON(w, r, data, http.StatusOK)
}

func repositoriesManagerOAuthCallbackHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	cberr := r.FormValue("error")
	errDescription := r.FormValue("error_description")
	errURI := r.FormValue("error_uri")

	if cberr != "" {
		log.Critical("Callback Error: %s - %s - %s", cberr, errDescription, errURI)
		return fmt.Errorf("OAuth Error %s", cberr)
	}

	code := r.FormValue("code")
	state := r.FormValue("state")

	data := map[string]string{}

	cache.Get(cache.Key("reposmanager", "oauth", state), &data)
	projectKey := data["project_key"]
	rmName := data["repositories_manager"]

	//Load the repositories manager from the DB
	rm, err := repositoriesmanager.LoadForProject(db, projectKey, rmName)
	if err != nil {
		log.Warning("repositoriesManagerAuthorizeCallback> error %s\n", err)
		return sdk.ErrNoReposManager

	}

	accessToken, accessTokenSecret, err := rm.Consumer.AuthorizeToken(state, code)
	if err != nil {
		log.Warning("repositoriesManagerAuthorizeCallback> Error with AuthorizeToken: %s", err)
		return sdk.ErrNoReposManagerClientAuth

	}

	log.Notice("repositoriesManagerAuthorizeCallback> [%s] AccessToken=%s; AccessTokenSecret=%s\n", projectKey, accessToken, accessTokenSecret)
	result := map[string]string{
		"project_key":          projectKey,
		"repositories_manager": rmName,
		"access_token":         accessToken,
		"access_token_secret":  accessTokenSecret,
	}

	if err := repositoriesmanager.SaveDataForProject(db, rm, projectKey, result); err != nil {
		log.Warning("repositoriesManagerAuthorizeCallback> Error with SaveDataForProject: %s", err)
		return err
	}

	//Redirect on UI advanced project page
	url := fmt.Sprintf("%s/#/project/%s?tab=advanced", baseURL, projectKey)
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)

	return nil
}

func repositoriesManagerAuthorizeCallback(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	// Get project name in URL
	vars := mux.Vars(r)
	projectKey := vars["permProjectKey"]
	rmName := vars["name"]

	rm, err := repositoriesmanager.LoadForProject(db, projectKey, rmName)
	if err != nil {
		log.Warning("repositoriesManagerAuthorizeCallback> Cannot find repository manager %s for project %s\n", rmName, projectKey)
		return sdk.ErrNoReposManager

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
		log.Warning("repositoriesManagerAuthorizeCallback> Cannot get token nor verifier from data")
		return sdk.ErrWrongRequest

	}

	accessToken, accessTokenSecret, err := rm.Consumer.AuthorizeToken(token, verifier)
	if err != nil {
		log.Warning("repositoriesManagerAuthorizeCallback> Error with AuthorizeToken: %s", err)
		return sdk.ErrNoReposManagerClientAuth

	}

	log.Notice("repositoriesManagerAuthorizeCallback> [%s] AccessToken=%s; AccessTokenSecret=%s\n", projectKey, accessToken, accessTokenSecret)
	result := map[string]string{
		"project_key":          projectKey,
		"repositories_manager": rmName,
		"access_token":         accessToken,
		"access_token_secret":  accessTokenSecret,
	}

	if err := repositoriesmanager.SaveDataForProject(db, rm, projectKey, result); err != nil {
		log.Warning("repositoriesManagerAuthorizeCallback> Error with SaveDataForProject: %s", err)
		return err

	}

	p, err := project.Load(db, projectKey, c.User, project.LoadOptions.WithRepositoriesManagers)
	if err != nil {
		log.Warning("repositoriesManagerAuthorizeCallback> Cannot load project %s: %s\n", projectKey, err)
		return err

	}

	return WriteJSON(w, r, p, http.StatusOK)
}

func deleteRepositoriesManagerHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	// Get project name in URL
	vars := mux.Vars(r)
	projectKey := vars["permProjectKey"]
	rmName := vars["name"]

	p, err := project.Load(db, projectKey, c.User)
	if err != nil {
		log.Warning("deleteRepositoriesManagerHandler> Cannot load project %s: %s\n", projectKey, err)
		return err

	}

	//Load the repositories manager from the DB
	rm, err := repositoriesmanager.LoadForProject(db, projectKey, rmName)
	if err != nil {
		log.Warning("deleteRepositoriesManagerHandler> error loading %s-%s: %s\n", projectKey, rmName, err)
		return sdk.ErrNoReposManager

	}

	tx, err := db.Begin()
	if err != nil {
		log.Warning("deleteRepositoriesManagerHandler> Cannot start transaction: %s\n", err)
		return err

	}
	defer tx.Rollback()

	if err := repositoriesmanager.DeleteForProject(tx, rm, p); err != nil {
		log.Warning("deleteRepositoriesManagerHandler> error deleting %s-%s: %s\n", projectKey, rmName, err)
		return err

	}

	err = tx.Commit()
	if err != nil {
		log.Warning("deleteRepositoriesManagerHandler> Cannot commit transaction: %s\n", err)
		return err

	}

	p.ReposManager, err = repositoriesmanager.LoadAllForProject(db, p.Key)
	if err != nil {
		log.Warning("deleteRepositoriesManagerHandler> Cannot load repos manager for project %s: %s\n", p.Key, err)
		return err

	}

	return WriteJSON(w, r, p, http.StatusOK)

}

func getReposFromRepositoriesManagerHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	// Get project name in URL
	vars := mux.Vars(r)
	projectKey := vars["permProjectKey"]
	rmName := vars["name"]

	client, err := repositoriesmanager.AuthorizedClient(db, projectKey, rmName)
	if err != nil {
		log.Warning("getReposFromRepositoriesManagerHandler> Cannot get client got %s %s : %s", projectKey, rmName, err)
		return sdk.ErrNoReposManagerClientAuth

	}

	var repos []sdk.VCSRepo
	cacheKey := cache.Key("reposmanager", "repos", projectKey, rmName)
	cache.Get(cacheKey, &repos)
	if repos == nil || len(repos) == 0 {
		log.Debug("getReposFromRepositoriesManagerHandler> loading from Stash")
		repos, err = client.Repos()
	}
	if err != nil {
		log.Warning("getReposFromRepositoriesManagerHandler> Cannot get repos: %s", err)
		return err

	}
	return WriteJSON(w, r, repos, http.StatusOK)
}

func getRepoFromRepositoriesManagerHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	// Get project name in URL
	vars := mux.Vars(r)
	projectKey := vars["permProjectKey"]
	rmName := vars["name"]
	repoName := r.FormValue("repo")

	client, err := repositoriesmanager.AuthorizedClient(db, projectKey, rmName)
	if err != nil {
		log.Warning("repositoriesManagerAuthorizeCallback> Cannot get client got %s %s : %s", projectKey, rmName, err)
		return sdk.ErrNoReposManagerClientAuth

	}
	repo, err := client.RepoByFullname(repoName)
	if err != nil {
		log.Warning("repositoriesManagerAuthorizeCallback> Cannot get repos: %s", err)
		return err

	}
	return WriteJSON(w, r, repo, http.StatusOK)
}

func attachRepositoriesManager(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	// Get project name in URL
	vars := mux.Vars(r)
	projectKey := vars["key"]
	appName := vars["permApplicationName"]
	rmName := vars["name"]
	fullname := r.FormValue("fullname")

	app, err := application.LoadByName(db, projectKey, appName, c.User)
	if err != nil {
		log.Warning("attachRepositoriesManager> Cannot load application %s: %s\n", appName, err)
		return err

	}

	//Load the repositoriesManager for the project
	rm, err := repositoriesmanager.LoadForProject(db, projectKey, rmName)
	if err != nil {
		log.Warning("attachRepositoriesManager> error loading %s-%s: %s\n", projectKey, rmName, err)
		return sdk.ErrNoReposManager

	}

	//Get an authorized Client
	client, err := repositoriesmanager.AuthorizedClient(db, projectKey, rmName)
	if err != nil {
		log.Warning("attachRepositoriesManager> Cannot get client got %s %s : %s", projectKey, rmName, err)
		return sdk.ErrNoReposManagerClientAuth
	}

	_, errR := client.RepoByFullname(fullname)
	if errR != nil {
		log.Warning("attachRepositoriesManager> Cannot get repo %s: %s", fullname, errR)
		return sdk.ErrRepoNotFound
	}

	app.RepositoriesManager = rm
	app.RepositoryFullname = fullname

	if err := repositoriesmanager.InsertForApplication(db, app, projectKey); err != nil {
		log.Warning("attachRepositoriesManager> Cannot insert for application: %s", err)
		return err

	}

	return WriteJSON(w, r, app, http.StatusOK)
}

func detachRepositoriesManager(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	// Get project name in URL
	vars := mux.Vars(r)
	projectKey := vars["key"]
	appName := vars["permApplicationName"]
	rmName := vars["name"]

	app, err := application.LoadByName(db, projectKey, appName, c.User, application.LoadOptions.WithHooks)
	if err != nil {
		return err
	}

	client, err := repositoriesmanager.AuthorizedClient(db, projectKey, rmName)
	if err != nil {
		log.Warning("detachRepositoriesManager> Cannot get client got %s %s : %s", projectKey, rmName, err)
		return sdk.ErrNoReposManagerClientAuth

	}

	//Remove all the things in a transaction
	tx, err := db.Begin()
	defer tx.Rollback()

	if err := repositoriesmanager.DeleteForApplication(tx, projectKey, app); err != nil {
		log.Warning("detachRepositoriesManager> Cannot delete for application: %s", err)
		return err

	}

	for _, h := range app.Hooks {
		s := viper.GetString("api_url") + hook.HookLink
		link := fmt.Sprintf(s, h.UID, h.Project, h.Repository)

		if err = client.DeleteHook(h.Project+"/"+h.Repository, link); err != nil {
			log.Warning("detachRepositoriesManager> Cannot delete hook on stash: %s", err)
			//do no return, try to delete the hook in database
		}

		if err := hook.DeleteHook(tx, h.ID); err != nil {
			log.Warning("detachRepositoriesManager> Cannot get hook: %s", err)
			return err

		}
	}

	// Remove reposmanager poller
	if err := poller.DeleteAll(tx, app.ID); err != nil {
		return err

	}

	if err := tx.Commit(); err != nil {
		log.Warning("detachRepositoriesManager> Cannot commit transaction: %s", err)
		return err

	}

	return WriteJSON(w, r, app, http.StatusOK)
}

func getRepositoriesManagerForApplicationsHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	return errors.New("Not implemented")
}

func addHookOnRepositoriesManagerHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	// Get project name in URL
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

	app, err := application.LoadByName(db, projectKey, appName, c.User)
	if err != nil {
		return sdk.ErrApplicationNotFound

	}

	pipeline, err := pipeline.LoadPipeline(db, projectKey, pipelineName, false)
	if err != nil {
		return sdk.ErrPipelineNotFound

	}

	if !permission.AccessToPipeline(sdk.DefaultEnv.ID, pipeline.ID, c.User, permission.PermissionReadWriteExecute) {
		log.Warning("addHookOnRepositoriesManagerHandler> You don't have enought right on this pipeline %s", pipeline.Name)
		return sdk.ErrForbidden

	}

	rm, err := repositoriesmanager.LoadForProject(db, projectKey, rmName)
	if err != nil {
		log.Warning("addHookOnRepositoriesManagerHandler> error loading %s-%s: %s\n", projectKey, rmName, err)
		return sdk.ErrNoReposManager

	}

	b, e := repositoriesmanager.CheckApplicationIsAttached(db, rmName, projectKey, appName)
	if e != nil {
		log.Warning("addHookOnRepositoriesManagerHandler> Cannot check app (%s,%s,%s): %s", rmName, projectKey, appName, e)
		return e

	}

	if !b {
		return sdk.ErrNoReposManagerClientAuth

	}

	tx, err := db.Begin()
	if err != nil {
		log.Warning("addHookOnRepositoriesManagerHandler> Cannot start transaction: %s", err)
		return err

	}
	defer tx.Rollback()

	_, err = hook.CreateHook(tx, projectKey, rm, repoFullname, app, pipeline)
	if err != nil {
		return err

	}

	if err := application.UpdateLastModified(tx, app, c.User); err != nil {
		log.Warning("addHookOnRepositoriesManagerHandler> Cannot update application last modified date: %s", err)
		return err

	}

	err = tx.Commit()
	if err != nil {
		log.Warning("addHookOnRepositoriesManagerHandler> Cannot commit transaction: %s", err)
		return err

	}

	app.Hooks, err = hook.LoadApplicationHooks(db, app.ID)
	if err != nil {
		log.Warning("addHookOnRepositoriesManagerHandler> Cannot load application hooks: %s", err)
		return err

	}

	return WriteJSON(w, r, app, http.StatusCreated)
}

func deleteHookOnRepositoriesManagerHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	// Get project name in URL
	vars := mux.Vars(r)
	projectKey := vars["key"]
	appName := vars["permApplicationName"]
	rmName := vars["name"]
	hookIDString := vars["hookId"]

	hookID, errparse := strconv.ParseInt(hookIDString, 10, 64)
	if errparse != nil {
		return sdk.WrapError(sdk.ErrWrongRequest, "deleteHookOnRepositoriesManagerHandler> Unable to parse hook id")
	}

	app, errload := application.LoadByName(db, projectKey, appName, c.User)
	if errload != nil {
		return sdk.WrapError(errload, "deleteHookOnRepositoriesManagerHandler> Application %s/%s not found ", projectKey, appName)
	}

	h, errhook := hook.LoadHook(db, hookID)
	if errhook != nil {
		return sdk.WrapError(errhook, "deleteHookOnRepositoriesManagerHandler> Unable to load hook %d ", hookID)
	}

	tx, errtx := db.Begin()
	if errtx != nil {
		return sdk.WrapError(errtx, "deleteHookOnRepositoriesManagerHandler> Unable to start transaction")
	}
	defer tx.Rollback()

	if errdelete := hook.DeleteHook(tx, h.ID); errdelete != nil {
		return sdk.WrapError(errdelete, "deleteHookOnRepositoriesManagerHandler> Unable to delete hook %d", h.ID)
	}

	if errupdate := application.UpdateLastModified(tx, app, c.User); errupdate != nil {
		return sdk.WrapError(errupdate, "deleteHookOnRepositoriesManagerHandler> Unable to update last modified")
	}

	if errtx := tx.Commit(); errtx != nil {
		return sdk.WrapError(errtx, "deleteHookOnRepositoriesManagerHandler> Unable to commit transaction")
	}

	var errW error
	app.Workflows, errW = workflow.LoadCDTree(db, projectKey, app.Name, c.User)
	if errW != nil {
		return sdk.WrapError(errW, "deleteHookOnRepositoriesManagerHandler> Unable to load workflow")
	}

	b, errcheck := repositoriesmanager.CheckApplicationIsAttached(db, rmName, projectKey, appName)
	if errcheck != nil {
		return sdk.WrapError(errcheck, "deleteHookOnRepositoriesManagerHandler> Cannot check app (%s,%s,%s)", rmName, projectKey, appName)
	}

	if !b {
		return sdk.WrapError(sdk.ErrNoReposManagerClientAuth, "deleteHookOnRepositoriesManagerHandler> Applicaiton %s is not attached to any repository", appName)
	}

	client, errauth := repositoriesmanager.AuthorizedClient(db, projectKey, rmName)
	if errauth != nil {
		return sdk.WrapError(errauth, "deleteHookOnRepositoriesManagerHandler> Cannot get client %s %s", projectKey, rmName)
	}

	t := strings.Split(app.RepositoryFullname, "/")
	if len(t) != 2 {
		return sdk.WrapError(sdk.ErrRepoNotFound, "deleteHookOnRepositoriesManagerHandler> Application %s repository fullname is not valid %s", app.Name, app.RepositoryFullname)
	}

	s := viper.GetString("api_url") + hook.HookLink
	link := fmt.Sprintf(s, h.UID, t[0], t[1])

	if errdelete := client.DeleteHook(app.RepositoryFullname, link); errdelete != nil {
		return sdk.WrapError(errdelete, "deleteHookOnRepositoriesManagerHandler> Cannot delete hook on stash")
	}

	return WriteJSON(w, r, app, http.StatusOK)
}

func addApplicationFromRepositoriesManagerHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	vars := mux.Vars(r)
	projectKey := vars["permProjectKey"]
	rmName := vars["name"]

	var data map[string]string
	if err := UnmarshalBody(r, &data); err != nil {
		return err
	}

	repoFullname := data["repository_fullname"]
	if repoFullname == "" {
		log.Warning("addApplicationFromRepositoriesManagerHandler>Repository fullname is mandatory")
		return sdk.ErrWrongRequest

	}

	proj, err := project.Load(db, projectKey, c.User)
	if err != nil {
		log.Warning("addApplicationFromRepositoriesManagerHandler: Cannot load %s: %s\n", projectKey, err)
		return sdk.ErrInvalidProject
	}

	rm, err := repositoriesmanager.LoadForProject(db, projectKey, rmName)
	if err != nil {
		log.Warning("addApplicationFromRepositoriesManagerHandler> error loading %s-%s: %s\n", projectKey, rmName, err)
		return sdk.ErrNoReposManager

	}

	client, err := repositoriesmanager.AuthorizedClient(db, projectKey, rmName)
	if err != nil {
		log.Warning("addApplicationFromRepositoriesManagerHandler> Cannot get client got %s %s : %s", projectKey, rmName, err)
		return sdk.ErrNoReposManagerClientAuth

	}

	repo, err := client.RepoByFullname(repoFullname)
	if err != nil {
		log.Warning("addApplicationFromRepositoriesManagerHandler> Cannot get repo: %s", err)
		return sdk.ErrRepoNotFound
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

	tx, err := db.Begin()
	if err != nil {
		log.Warning("addApplicationFromRepositoriesManagerHandler> Cannot start transaction: %s\n", err)
		return err

	}

	defer tx.Rollback()

	//Insert application in database
	if err := application.Insert(tx, proj, &app); err != nil {
		log.Warning("addApplicationFromRepositoriesManagerHandler> Cannot insert pipeline: %s\n", err)
		return err

	}

	//Fetch groups from project
	if err := group.LoadGroupByProject(tx, proj); err != nil {
		log.Warning("addApplicationFromRepositoriesManagerHandler> Cannot load group from project: %s\n", err)
		return err
	}

	//Add the  groups on the application
	if err := application.AddGroup(tx, proj, &app, proj.ProjectGroups...); err != nil {
		log.Warning("addApplicationFromRepositoriesManagerHandler> Cannot add groups on application: %s\n", err)
		return err
	}

	//Commit the transaction
	if err = tx.Commit(); err != nil {
		log.Warning("addApplicationFromRepositoriesManagerHandler> Cannot commit transaction: %s\n", err)
		return err
	}

	//Attach the application to the repositories manager
	app.RepositoriesManager = rm
	app.RepositoryFullname = repoFullname
	if err := repositoriesmanager.InsertForApplication(db, &app, projectKey); err != nil {
		log.Warning("addApplicationFromRepositoriesManagerHandler> Cannot attach application: %s", err)
		return err
	}

	return nil
}
