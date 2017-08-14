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
	"github.com/ovh/cds/engine/api/businesscontext"
	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/hook"
	"github.com/ovh/cds/engine/api/permission"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/poller"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/repositoriesmanager"
	"github.com/ovh/cds/engine/api/user"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func getRepositoriesManagerHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *businesscontext.Ctx) error {
	rms, err := repositoriesmanager.LoadAll(db)
	if err != nil {
		return sdk.WrapError(err, "getRepositoriesManagerHandler> error")
	}
	return WriteJSON(w, r, rms, http.StatusOK)
}

func addRepositoriesManagerHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *businesscontext.Ctx) error {
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

	rm, err := repositoriesmanager.New(sdk.RepositoriesManagerType(t), 0, name, url, options, "")
	if err != nil {
		return sdk.WrapError(err, "addRepositoriesManagerHandler> cannot create %s")
	}
	if err := repositoriesmanager.Insert(db, rm); err != nil {
		return sdk.WrapError(err, "addRepositoriesManagerHandler> cannot insert %s")
	}
	return WriteJSON(w, r, rm, http.StatusCreated)
}

func getRepositoriesManagerForProjectHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *businesscontext.Ctx) error {
	vars := mux.Vars(r)
	key := vars["permProjectKey"]

	rms, err := repositoriesmanager.LoadAllForProject(db, key)
	if err != nil {
		return sdk.WrapError(err, "getRepositoriesManagerForProjectHandler> error %s")
	}
	return WriteJSON(w, r, rms, http.StatusOK)
}

func repositoriesManagerAuthorize(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *businesscontext.Ctx) error {
	vars := mux.Vars(r)
	key := vars["permProjectKey"]
	rmName := vars["name"]

	proj, errP := project.Load(db, key, c.User)
	if errP != nil {
		return sdk.WrapError(errP, "repositoriesManagerAuthorize> Cannot load project")
	}

	//Load the repositories manager from the DB
	rm, errFind := repositoriesmanager.LoadForProject(db, proj.Key, rmName)
	var lastModified time.Time

	//If we don't find any repositories manager for the project, let's insert it
	if errFind == sql.ErrNoRows {
		var errLoad error
		rm, errLoad = repositoriesmanager.LoadByName(db, rmName)
		if errLoad != nil {
			return sdk.WrapError(sdk.ErrNoReposManager, "repositoriesManagerAuthorize> error while loading repositories manager %s", errLoad)
		}

		tx, err := db.Begin()
		if err != nil {
			return sdk.WrapError(err, "repositoriesManagerAuthorize> Cannot start transaction")
		}
		defer tx.Rollback()

		if errI := repositoriesmanager.InsertForProject(tx, rm, proj.Key); errI != nil {
			return sdk.WrapError(errI, "repositoriesManagerAuthorize> error while inserting repositories manager for project %s", proj.Key)
		}

		if err := project.UpdateLastModified(tx, c.User, proj); err != nil {
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
		"username":      c.User.Username,
	}

	cache.Set(cache.Key("reposmanager", "oauth", token), data)
	return WriteJSON(w, r, data, http.StatusOK)
}

func repositoriesManagerOAuthCallbackHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *businesscontext.Ctx) error {
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

	cache.Get(cache.Key("reposmanager", "oauth", state), &data)
	projectKey := data["project_key"]
	rmName := data["repositories_manager"]
	username := data["username"]

	u, errU := user.LoadUserWithoutAuth(db, username)
	if errU != nil {
		return sdk.WrapError(errU, "repositoriesManagerAuthorizeCallback> Cannot load user %s", username)
	}

	proj, errP := project.Load(db, projectKey, u)
	if errP != nil {
		return sdk.WrapError(errP, "repositoriesManagerAuthorizeCallback> Cannot load project")
	}

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

	log.Info("repositoriesManagerAuthorizeCallback> [%s] AccessToken=%s; AccessTokenSecret=%s", projectKey, accessToken, accessTokenSecret)
	result := map[string]string{
		"project_key":          projectKey,
		"repositories_manager": rmName,
		"access_token":         accessToken,
		"access_token_secret":  accessTokenSecret,
	}

	tx, errT := db.Begin()
	if errT != nil {
		return sdk.WrapError(errT, "repositoriesManagerAuthorizeCallback> Cannot start transaction")
	}
	defer tx.Rollback()

	if err := repositoriesmanager.SaveDataForProject(tx, rm, projectKey, result); err != nil {
		return sdk.WrapError(err, "repositoriesManagerAuthorizeCallback> Error with SaveDataForProject")
	}

	if err := project.UpdateLastModified(tx, u, proj); err != nil {
		return sdk.WrapError(err, "repositoriesManagerAuthorizeCallback> Cannot update project last modified date")
	}

	if err := tx.Commit(); err != nil {
		return sdk.WrapError(errT, "repositoriesManagerAuthorizeCallback> Cannot commit transaction")
	}

	//Redirect on UI advanced project page
	url := fmt.Sprintf("%s/project/%s?tab=advanced", baseURL, projectKey)
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)

	return nil
}

func repositoriesManagerAuthorizeCallback(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *businesscontext.Ctx) error {
	vars := mux.Vars(r)
	projectKey := vars["permProjectKey"]
	rmName := vars["name"]

	rm, errl := repositoriesmanager.LoadForProject(db, projectKey, rmName)
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

	if err := repositoriesmanager.SaveDataForProject(db, rm, projectKey, result); err != nil {
		return sdk.WrapError(err, "repositoriesManagerAuthorizeCallback> Error with SaveDataForProject")
	}

	p, err := project.Load(db, projectKey, c.User, project.LoadOptions.WithRepositoriesManagers)
	if err != nil {
		return sdk.WrapError(err, "repositoriesManagerAuthorizeCallback> Cannot load project %s", projectKey)
	}

	return WriteJSON(w, r, p, http.StatusOK)
}

func deleteRepositoriesManagerHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *businesscontext.Ctx) error {
	vars := mux.Vars(r)
	projectKey := vars["permProjectKey"]
	rmName := vars["name"]

	p, errl := project.Load(db, projectKey, c.User)
	if errl != nil {
		return sdk.WrapError(errl, "deleteRepositoriesManagerHandler> Cannot load project %s", projectKey)
	}

	// Load the repositories manager from the DB
	rm, errlp := repositoriesmanager.LoadForProject(db, projectKey, rmName)
	if errlp != nil {
		return sdk.WrapError(sdk.ErrNoReposManager, "deleteRepositoriesManagerHandler> error loading %s-%s: %s", projectKey, rmName, errlp)
	}

	tx, errb := db.Begin()
	if errb != nil {
		return sdk.WrapError(errb, "deleteRepositoriesManagerHandler> Cannot start transaction")
	}
	defer tx.Rollback()

	if err := repositoriesmanager.DeleteForProject(tx, rm, p); err != nil {
		return sdk.WrapError(err, "deleteRepositoriesManagerHandler> error deleting %s-%s", projectKey, rmName)
	}

	if err := project.UpdateLastModified(tx, c.User, p); err != nil {
		return sdk.WrapError(err, "deleteRepositoriesManagerHandler> Cannot update project last modified date")
	}

	if err := tx.Commit(); err != nil {
		return sdk.WrapError(err, "deleteRepositoriesManagerHandler> Cannot commit transaction")
	}

	var errla error
	p.ReposManager, errla = repositoriesmanager.LoadAllForProject(db, p.Key)
	if errla != nil {
		return sdk.WrapError(errla, "deleteRepositoriesManagerHandler> Cannot load repos manager for project %s", p.Key)
	}

	return WriteJSON(w, r, p, http.StatusOK)
}

func getReposFromRepositoriesManagerHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *businesscontext.Ctx) error {
	vars := mux.Vars(r)
	projectKey := vars["permProjectKey"]
	rmName := vars["name"]
	sync := FormBool(r, "synchronize")

	client, err := repositoriesmanager.AuthorizedClient(db, projectKey, rmName)
	if err != nil {
		return sdk.WrapError(sdk.ErrNoReposManagerClientAuth, "getReposFromRepositoriesManagerHandler> Cannot get client got %s %s", projectKey, rmName)
	}

	cacheKey := cache.Key("reposmanager", "repos", projectKey, rmName)
	if sync {
		cache.Delete(cacheKey)
	}

	var repos []sdk.VCSRepo
	if !cache.Get(cacheKey, &repos) {
		log.Debug("getReposFromRepositoriesManagerHandler> loading from Stash")
		repos, err = client.Repos()
		cache.SetWithTTL(cacheKey, repos, 0)
	}
	if err != nil {
		return sdk.WrapError(err, "getReposFromRepositoriesManagerHandler> Cannot get repos")

	}
	return WriteJSON(w, r, repos, http.StatusOK)
}

func getRepoFromRepositoriesManagerHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *businesscontext.Ctx) error {
	// Get project name in URL
	vars := mux.Vars(r)
	projectKey := vars["permProjectKey"]
	rmName := vars["name"]
	repoName := r.FormValue("repo")

	client, err := repositoriesmanager.AuthorizedClient(db, projectKey, rmName)
	if err != nil {
		return sdk.WrapError(sdk.ErrNoReposManagerClientAuth, "getRepoFromRepositoriesManagerHandler> Cannot get client got %s %s : %s", projectKey, rmName, err)
	}
	repo, err := client.RepoByFullname(repoName)
	if err != nil {
		return sdk.WrapError(err, "getRepoFromRepositoriesManagerHandler> Cannot get repos")
	}
	return WriteJSON(w, r, repo, http.StatusOK)
}

func attachRepositoriesManager(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *businesscontext.Ctx) error {
	vars := mux.Vars(r)
	projectKey := vars["key"]
	appName := vars["permApplicationName"]
	rmName := vars["name"]
	fullname := r.FormValue("fullname")

	app, err := application.LoadByName(db, projectKey, appName, c.User)
	if err != nil {
		return sdk.WrapError(err, "attachRepositoriesManager> Cannot load application %s", appName)
	}

	//Load the repositoriesManager for the project
	rm, err := repositoriesmanager.LoadForProject(db, projectKey, rmName)
	if err != nil {
		return sdk.WrapError(sdk.ErrNoReposManager, "attachRepositoriesManager> error loading %s-%s: %s", projectKey, rmName, err)
	}

	//Get an authorized Client
	client, err := repositoriesmanager.AuthorizedClient(db, projectKey, rmName)
	if err != nil {
		return sdk.WrapError(sdk.ErrNoReposManagerClientAuth, "attachRepositoriesManager> Cannot get client got %s %s : %s", projectKey, rmName, err)
	}

	if _, err := client.RepoByFullname(fullname); err != nil {
		return sdk.WrapError(sdk.ErrRepoNotFound, "attachRepositoriesManager> Cannot get repo %s: %s", fullname, err)
	}

	app.RepositoriesManager = rm
	app.RepositoryFullname = fullname

	tx, errT := db.Begin()
	if errT != nil {
		return sdk.WrapError(errT, "attachRepositoriesManager> Cannot start transaction")
	}
	defer tx.Rollback()

	if err := repositoriesmanager.InsertForApplication(tx, app, projectKey); err != nil {
		return sdk.WrapError(err, "attachRepositoriesManager> Cannot insert for application")
	}

	if err := application.UpdateLastModified(tx, app, c.User); err != nil {
		return sdk.WrapError(err, "attachRepositoriesManager> Cannot update application last modified date")
	}

	if err := tx.Commit(); err != nil {
		return sdk.WrapError(err, "attachRepositoriesManager> Cannot commit transaction")
	}

	return WriteJSON(w, r, app, http.StatusOK)
}

func detachRepositoriesManager(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *businesscontext.Ctx) error {
	vars := mux.Vars(r)
	projectKey := vars["key"]
	appName := vars["permApplicationName"]
	rmName := vars["name"]

	app, errl := application.LoadByName(db, projectKey, appName, c.User, application.LoadOptions.WithHooks)
	if errl != nil {
		return sdk.WrapError(errl, "detachRepositoriesManager> error on load project %s", projectKey)
	}

	client, erra := repositoriesmanager.AuthorizedClient(db, projectKey, rmName)
	if erra != nil {
		return sdk.WrapError(sdk.ErrNoReposManagerClientAuth, "detachRepositoriesManager> Cannot get client got %s %s: %s", projectKey, rmName, erra)
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

	for _, h := range app.Hooks {
		s := viper.GetString(viperURLAPI) + hook.HookLink
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

	if err := application.UpdateLastModified(tx, app, c.User); err != nil {
		return sdk.WrapError(err, "detachRepositoriesManager> Cannot update application last modified date")
	}

	if err := tx.Commit(); err != nil {
		return sdk.WrapError(err, "detachRepositoriesManager> Cannot commit transaction")
	}

	return WriteJSON(w, r, app, http.StatusOK)
}

func getRepositoriesManagerForApplicationsHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *businesscontext.Ctx) error {
	return errors.New("Not implemented")
}

func addHookOnRepositoriesManagerHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *businesscontext.Ctx) error {
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

	app, errla := application.LoadByName(db, projectKey, appName, c.User)
	if errla != nil {
		return sdk.ErrApplicationNotFound
	}

	pipeline, errl := pipeline.LoadPipeline(db, projectKey, pipelineName, false)
	if errl != nil {
		return sdk.ErrPipelineNotFound
	}

	if !permission.AccessToPipeline(sdk.DefaultEnv.ID, pipeline.ID, c.User, permission.PermissionReadWriteExecute) {
		return sdk.WrapError(sdk.ErrForbidden, "addHookOnRepositoriesManagerHandler> You don't have enought right on this pipeline %s", pipeline.Name)
	}

	rm, errlp := repositoriesmanager.LoadForProject(db, projectKey, rmName)
	if errlp != nil {
		return sdk.WrapError(sdk.ErrNoReposManager, "addHookOnRepositoriesManagerHandler> error loading %s-%s: %s", projectKey, rmName, errlp)
	}

	b, e := repositoriesmanager.CheckApplicationIsAttached(db, rmName, projectKey, appName)
	if e != nil {
		return sdk.WrapError(e, "addHookOnRepositoriesManagerHandler> Cannot check app (%s,%s,%s)", rmName, projectKey, appName)
	}

	if !b {
		return sdk.ErrNoReposManagerClientAuth
	}

	tx, errb := db.Begin()
	if errb != nil {
		return sdk.WrapError(errb, "addHookOnRepositoriesManagerHandler> cannot start transaction")
	}
	defer tx.Rollback()

	if _, err := hook.CreateHook(tx, projectKey, rm, repoFullname, app, pipeline); err != nil {
		return sdk.WrapError(err, "addHookOnRepositoriesManagerHandler> cannot create hook")
	}

	if err := application.UpdateLastModified(tx, app, c.User); err != nil {
		return sdk.WrapError(err, "addHookOnRepositoriesManagerHandler> cannot update application last modified date")
	}

	if err := tx.Commit(); err != nil {
		return sdk.WrapError(err, "addHookOnRepositoriesManagerHandler> cannot commit transaction")
	}

	var errlah error
	app.Hooks, errlah = hook.LoadApplicationHooks(db, app.ID)
	if errlah != nil {
		return sdk.WrapError(errlah, "addHookOnRepositoriesManagerHandler> cannot load application hooks")
	}

	var errW error
	app.Workflows, errW = workflow.LoadCDTree(db, projectKey, app.Name, c.User, "", 0)
	if errW != nil {
		return sdk.WrapError(errW, "addHookOnRepositoriesManagerHandler> Cannot load workflow")
	}

	return WriteJSON(w, r, app, http.StatusCreated)
}

func deleteHookOnRepositoriesManagerHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *businesscontext.Ctx) error {
	vars := mux.Vars(r)
	projectKey := vars["key"]
	appName := vars["permApplicationName"]
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
	app.Workflows, errW = workflow.LoadCDTree(db, projectKey, app.Name, c.User, "", 0)
	if errW != nil {
		return sdk.WrapError(errW, "deleteHookOnRepositoriesManagerHandler> Unable to load workflow")
	}

	var errR error
	_, app.RepositoriesManager, errR = repositoriesmanager.LoadFromApplicationByID(db, app.ID)
	if errR != nil {
		return sdk.WrapError(errR, "deleteHookOnRepositoriesManagerHandler> Cannot load repository manager from application %s", appName)
	}

	client, errauth := repositoriesmanager.AuthorizedClient(db, projectKey, app.RepositoriesManager.Name)
	if errauth != nil {
		return sdk.WrapError(errauth, "deleteHookOnRepositoriesManagerHandler> Cannot get client %s %s", projectKey, app.RepositoriesManager.Name)
	}

	t := strings.Split(app.RepositoryFullname, "/")
	if len(t) != 2 {
		return sdk.WrapError(sdk.ErrRepoNotFound, "deleteHookOnRepositoriesManagerHandler> Application %s repository fullname is not valid %s", app.Name, app.RepositoryFullname)
	}

	s := viper.GetString(viperURLAPI) + hook.HookLink
	link := fmt.Sprintf(s, h.UID, t[0], t[1])

	if errdelete := client.DeleteHook(app.RepositoryFullname, link); errdelete != nil {
		return sdk.WrapError(errdelete, "deleteHookOnRepositoriesManagerHandler> Cannot delete hook on stash")
	}

	return WriteJSON(w, r, app, http.StatusOK)
}

func addApplicationFromRepositoriesManagerHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *businesscontext.Ctx) error {
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

	proj, errlp := project.Load(db, projectKey, c.User)
	if errlp != nil {
		return sdk.WrapError(sdk.ErrInvalidProject, "addApplicationFromRepositoriesManagerHandler: Cannot load %s: %s", projectKey, errlp)
	}

	rm, errlrm := repositoriesmanager.LoadForProject(db, projectKey, rmName)
	if errlrm != nil {
		return sdk.WrapError(sdk.ErrNoReposManager, "addApplicationFromRepositoriesManagerHandler> error loading %s-%s: %s", projectKey, rmName, errlrm)
	}

	client, errac := repositoriesmanager.AuthorizedClient(db, projectKey, rmName)
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

	tx, errb := db.Begin()
	if errb != nil {
		return sdk.WrapError(errb, "addApplicationFromRepositoriesManagerHandler> Cannot start transaction")
	}

	defer tx.Rollback()

	//Insert application in database
	if err := application.Insert(tx, proj, &app, c.User); err != nil {
		return sdk.WrapError(err, "addApplicationFromRepositoriesManagerHandler> Cannot insert pipeline")
	}

	//Fetch groups from project
	if err := group.LoadGroupByProject(tx, proj); err != nil {
		return sdk.WrapError(err, "addApplicationFromRepositoriesManagerHandler> Cannot load group from project")
	}

	//Add the  groups on the application
	if err := application.AddGroup(tx, proj, &app, c.User, proj.ProjectGroups...); err != nil {
		return sdk.WrapError(err, "addApplicationFromRepositoriesManagerHandler> Cannot add groups on application")
	}

	//Commit the transaction
	if err := tx.Commit(); err != nil {
		return sdk.WrapError(err, "addApplicationFromRepositoriesManagerHandler> Cannot commit transaction")
	}

	//Attach the application to the repositories manager
	app.RepositoriesManager = rm
	app.RepositoryFullname = repoFullname
	if err := repositoriesmanager.InsertForApplication(db, &app, projectKey); err != nil {
		return sdk.WrapError(err, "addApplicationFromRepositoriesManagerHandler> Cannot attach application")
	}

	return nil
}
