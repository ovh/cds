package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/spf13/viper"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/context"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/hook"
	"github.com/ovh/cds/engine/api/permission"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/repositoriesmanager"
	"github.com/ovh/cds/engine/log"
	"github.com/ovh/cds/sdk"
)

func getRepositoriesManagerHandler(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {
	rms, err := repositoriesmanager.LoadAll(db)
	if err != nil {
		log.Warning("getRepositoriesManagerHandler> error %s\n", err)
		WriteError(w, r, err)
		return
	}
	WriteJSON(w, r, rms, http.StatusOK)
}

func addRepositoriesManagerHandler(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {
	var args interface{}
	options := map[string]string{}

	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Warning("addRepositoriesManagerHandler> Cannot read request body: %s", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if e := json.Unmarshal(data, &args); e != nil {
		log.Warning("addRepositoriesManagerHandler> Cannot parse request body: %s", err)
		w.WriteHeader(http.StatusBadRequest)
		return
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
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	rm, err := repositoriesmanager.New(sdk.RepositoriesManagerType(t), 0, name, url, options, "")
	if err != nil {
		log.Warning("addRepositoriesManagerHandler> cannot create %s\n", err)
		WriteError(w, r, err)
		return
	}
	if err := repositoriesmanager.Insert(db, rm); err != nil {
		log.Warning("addRepositoriesManagerHandler> cannot insert %s\n", err)
		WriteError(w, r, err)
		return
	}
	WriteJSON(w, r, rm, http.StatusCreated)
}

func getRepositoriesManagerForProjectHandler(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {
	// Get project name in URL
	vars := mux.Vars(r)
	key := vars["permProjectKey"]

	rms, err := repositoriesmanager.LoadAllForProject(db, key)
	if err != nil {
		log.Warning("getRepositoriesManagerForProjectHandler> error %s\n", err)
		WriteError(w, r, err)
		return
	}
	WriteJSON(w, r, rms, http.StatusOK)
}

func repositoriesManagerAuthorize(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {
	// Get project name in URL
	vars := mux.Vars(r)
	projectKey := vars["permProjectKey"]
	rmName := vars["name"]

	//Load the repositories manager from the DB
	rm, err := repositoriesmanager.LoadForProject(db, projectKey, rmName)
	var lastModified time.Time

	//If we don't find any repositories manager for the project, let's insert it
	if err == sql.ErrNoRows {
		var e error
		rm, e = repositoriesmanager.LoadByName(db, rmName)
		if e != nil {
			log.Warning("repositoriesManagerAuthorize> error while loading repositories manager %s\n", e)
			WriteError(w, r, sdk.ErrNoReposManager)
			return
		}

		tx, err := db.Begin()
		if err != nil {
			log.Warning("repositoriesManagerAuthorize> Cannot start transaction %s\n", err)
			WriteError(w, r, e)
			return
		}
		defer tx.Rollback()

		lastModified, e = repositoriesmanager.InsertForProject(tx, rm, projectKey)
		if e != nil {
			log.Warning("repositoriesManagerAuthorize> error while inserting repositories manager for project %s: %s\n", projectKey, e)
			WriteError(w, r, e)
			return
		}

		err = tx.Commit()
		if err != nil {
			log.Warning("repositoriesManagerAuthorize> Cannot commit transaction %s\n", err)
			WriteError(w, r, e)
			return
		}
	} else if err != nil {
		log.Warning("repositoriesManagerAuthorize> error %s\n", err)
		WriteError(w, r, err)
		return
	}

	token, url, err := rm.Consumer.AuthorizeRedirect()
	if err != nil {
		log.Warning("repositoriesManagerAuthorize> error with AuthorizeRedirect %s\n", err)
		WriteError(w, r, sdk.ErrNoReposManagerAuth)
		return
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

	WriteJSON(w, r, data, http.StatusOK)
}

func repositoriesManagerOAuthCallbackHandler(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {
	cberr := r.FormValue("error")
	errDescription := r.FormValue("error_description")
	errURI := r.FormValue("error_uri")

	if cberr != "" {
		log.Critical("Callback Error: %s - %s - %s", cberr, errDescription, errURI)
		WriteError(w, r, fmt.Errorf("OAuth Error %s", cberr))
		return
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
		WriteError(w, r, sdk.ErrNoReposManager)
		return
	}

	accessToken, accessTokenSecret, err := rm.Consumer.AuthorizeToken(state, code)
	if err != nil {
		log.Warning("repositoriesManagerAuthorizeCallback> Error with AuthorizeToken: %s", err)
		WriteError(w, r, sdk.ErrNoReposManagerClientAuth)
		return
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
		WriteError(w, r, err)
		return
	}

	//Redirect on UI advanced project page
	url := fmt.Sprintf("%s/#/project/%s?tab=advanced", baseURL, projectKey)
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
	return
}

func repositoriesManagerAuthorizeCallback(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {
	// Get project name in URL
	vars := mux.Vars(r)
	projectKey := vars["permProjectKey"]
	rmName := vars["name"]

	rm, err := repositoriesmanager.LoadForProject(db, projectKey, rmName)
	if err != nil {
		log.Warning("repositoriesManagerAuthorizeCallback> Cannot find repository manager %s for project %s\n", rmName, projectKey)
		WriteError(w, r, sdk.ErrNoReposManager)
		return
	}

	var tv map[string]interface{}
	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Warning("repositoriesManagerAuthorizeCallback> Cannot read request body: %s", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if e := json.Unmarshal(data, &tv); e != nil {
		log.Warning("repositoriesManagerAuthorizeCallback> Cannot parse request body: %s", err)
		w.WriteHeader(http.StatusBadRequest)
		return
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
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	accessToken, accessTokenSecret, err := rm.Consumer.AuthorizeToken(token, verifier)
	if err != nil {
		log.Warning("repositoriesManagerAuthorizeCallback> Error with AuthorizeToken: %s", err)
		WriteError(w, r, sdk.ErrNoReposManagerClientAuth)
		return
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
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	p, err := project.LoadProject(db, projectKey, c.User)
	if err != nil {
		log.Warning("repositoriesManagerAuthorizeCallback> Cannot load project %s: %s\n", projectKey, err)
		WriteError(w, r, err)
		return
	}

	p.ReposManager, err = repositoriesmanager.LoadAllForProject(db, projectKey)
	if err != nil {
		log.Warning("repositoriesManagerAuthorizeCallback> Cannot load repositories manager for project %s: %s\n", projectKey, err)
		WriteError(w, r, err)
		return
	}

	WriteJSON(w, r, p, http.StatusOK)
}

func deleteRepositoriesManagerHandler(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {
	// Get project name in URL
	vars := mux.Vars(r)
	projectKey := vars["permProjectKey"]
	rmName := vars["name"]

	p, err := project.LoadProject(db, projectKey, c.User)
	if err != nil {
		log.Warning("deleteRepositoriesManagerHandler> Cannot load project %s: %s\n", projectKey, err)
		WriteError(w, r, err)
		return
	}

	//Load the repositories manager from the DB
	rm, err := repositoriesmanager.LoadForProject(db, projectKey, rmName)
	if err != nil {
		log.Warning("deleteRepositoriesManagerHandler> error loading %s-%s: %s\n", projectKey, rmName, err)
		WriteError(w, r, sdk.ErrNoReposManager)
		return
	}

	tx, err := db.Begin()
	if err != nil {
		log.Warning("deleteRepositoriesManagerHandler> Cannot start transaction: %s\n", err)
		WriteError(w, r, err)
		return
	}
	defer tx.Rollback()

	if err := repositoriesmanager.DeleteForProject(tx, rm, p); err != nil {
		log.Warning("deleteRepositoriesManagerHandler> error deleting %s-%s: %s\n", projectKey, rmName, err)
		WriteError(w, r, err)
		return
	}

	err = tx.Commit()
	if err != nil {
		log.Warning("deleteRepositoriesManagerHandler> Cannot commit transaction: %s\n", err)
		WriteError(w, r, err)
		return
	}

	p.ReposManager, err = repositoriesmanager.LoadAllForProject(db, p.Key)
	if err != nil {
		log.Warning("deleteRepositoriesManagerHandler> Cannot load repos manager for project %s: %s\n", p.Key, err)
		WriteError(w, r, err)
		return
	}

	WriteJSON(w, r, p, http.StatusOK)

}

func getReposFromRepositoriesManagerHandler(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {
	// Get project name in URL
	vars := mux.Vars(r)
	projectKey := vars["permProjectKey"]
	rmName := vars["name"]

	client, err := repositoriesmanager.AuthorizedClient(db, projectKey, rmName)
	if err != nil {
		log.Warning("getReposFromRepositoriesManagerHandler> Cannot get client got %s %s : %s", projectKey, rmName, err)
		WriteError(w, r, sdk.ErrNoReposManagerClientAuth)
		return
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
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	WriteJSON(w, r, repos, http.StatusOK)
}

func getRepoFromRepositoriesManagerHandler(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {
	// Get project name in URL
	vars := mux.Vars(r)
	projectKey := vars["permProjectKey"]
	rmName := vars["name"]
	repoName := r.FormValue("repo")

	client, err := repositoriesmanager.AuthorizedClient(db, projectKey, rmName)
	if err != nil {
		log.Warning("repositoriesManagerAuthorizeCallback> Cannot get client got %s %s : %s", projectKey, rmName, err)
		WriteError(w, r, sdk.ErrNoReposManagerClientAuth)
		return
	}
	repo, err := client.RepoByFullname(repoName)
	if err != nil {
		log.Warning("repositoriesManagerAuthorizeCallback> Cannot get repos: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	WriteJSON(w, r, repo, http.StatusOK)
}

func getCommitsHandler(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {
	// Get project name in URL
	vars := mux.Vars(r)
	projectKey := vars["permProjectKey"]
	rmName := vars["name"]
	repoName := r.FormValue("repo")
	since := r.FormValue("since")
	until := r.FormValue("until")

	client, err := repositoriesmanager.AuthorizedClient(db, projectKey, rmName)
	if err != nil {
		log.Warning("getCommitsHandler> Cannot get client got %s %s : %s", projectKey, rmName, err)
		WriteError(w, r, sdk.ErrNoReposManagerClientAuth)
		return
	}

	log.Notice("getCommitsHandler> Searching commits for %s %s %s", repoName, since, until)
	commits, err := client.Commits(repoName, since, until)
	if err != nil {
		log.Warning("getCommitsHandler> Cannot get commits: %s", err)
		WriteError(w, r, err)
		return
	}
	WriteJSON(w, r, commits, http.StatusOK)
}

func attachRepositoriesManager(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {
	// Get project name in URL
	vars := mux.Vars(r)
	projectKey := vars["key"]
	appName := vars["permApplicationName"]
	rmName := vars["name"]
	fullname := r.FormValue("fullname")

	//Load the repositoriesManager for the project
	rm, err := repositoriesmanager.LoadForProject(db, projectKey, rmName)
	if err != nil {
		log.Warning("attachRepositoriesManager> error loading %s-%s: %s\n", projectKey, rmName, err)
		WriteError(w, r, sdk.ErrNoReposManager)
		return
	}

	//Get an authorized Client
	client, err := repositoriesmanager.AuthorizedClient(db, projectKey, rmName)
	if err != nil {
		log.Warning("attachRepositoriesManager> Cannot get client got %s %s : %s", projectKey, rmName, err)
		WriteError(w, r, sdk.ErrNoReposManagerClientAuth)
		return
	}

	_, errR := client.RepoByFullname(fullname)
	if errR != nil {
		log.Warning("attachRepositoriesManager> Cannot get repo: %s", errR)
		WriteError(w, r, sdk.ErrRepoNotFound)
		return
	}

	if err := repositoriesmanager.InsertForApplication(db, rm, projectKey, appName, fullname); err != nil {
		log.Warning("attachRepositoriesManager> Cannot insert for application: %s", err)
		WriteError(w, r, err)
		return
	}
}

func detachRepositoriesManager(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {
	// Get project name in URL
	vars := mux.Vars(r)
	projectKey := vars["key"]
	appName := vars["permApplicationName"]
	rmName := vars["name"]

	application, err := application.LoadApplicationByName(db, projectKey, appName)
	if err != nil {
		WriteError(w, r, sdk.ErrApplicationNotFound)
		return
	}

	//Load the repositoriesManager for the project
	rm, err := repositoriesmanager.LoadForProject(db, projectKey, rmName)
	if err != nil {
		log.Warning("detachRepositoriesManager> error loading %s-%s: %s\n", projectKey, rmName, err)
		WriteError(w, r, sdk.ErrNoReposManager)
		return
	}

	client, err := repositoriesmanager.AuthorizedClient(db, projectKey, rmName)
	if err != nil {
		log.Warning("detachRepositoriesManager> Cannot get client got %s %s : %s", projectKey, rmName, err)
		WriteError(w, r, sdk.ErrNoReposManagerClientAuth)
		return
	}

	//Remove all the things in a transaction
	tx, err := db.Begin()
	defer tx.Rollback()

	if err := repositoriesmanager.DeleteForApplication(tx, rm, projectKey, appName); err != nil {
		log.Warning("detachRepositoriesManager> Cannot delete for application: %s", err)
		WriteError(w, r, err)
		return
	}

	//Remove reposmanager hooks
	//Load all hooks
	hooks, err := hook.LoadApplicationHooks(tx, application.ID)
	if err != nil {
		log.Warning("detachRepositoriesManager> Cannot get hooks for application: %s", err)
		WriteError(w, r, err)
		return
	}

	for _, h := range hooks {
		s := viper.GetString("api_url") + hook.HookLink
		link := fmt.Sprintf(s, h.UID, h.Project, h.Repository)

		if err = client.DeleteHook(h.Project+"/"+h.Repository, link); err != nil {
			log.Warning("detachRepositoriesManager> Cannot delete hook on stash: %s", err)
			WriteError(w, r, err)
			return
		}

		if err := hook.DeleteHook(tx, h.ID); err != nil {
			log.Warning("detachRepositoriesManager> Cannot get hook: %s", err)
			WriteError(w, r, err)
			return
		}
	}

	if err := tx.Commit(); err != nil {
		log.Warning("detachRepositoriesManager> Cannot commit transaction: %s", err)
		WriteError(w, r, err)
		return
	}

}

func getRepositoriesManagerForApplicationsHandler(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {
	WriteError(w, r, errors.New("Not implemented"))
	return
}

func getApplicationCommitsHandler(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {
	// Get project name in URL
	vars := mux.Vars(r)
	projectKey := vars["key"]
	appName := vars["permApplicationName"]
	rmName := vars["name"]
	fullname := r.FormValue("fullname")
	since := r.FormValue("since")
	until := r.FormValue("until")

	b, e := repositoriesmanager.CheckApplicationIsAttached(db, rmName, projectKey, appName)
	if e != nil {
		log.Warning("getCommitsHandler> Cannot check app (%s,%s,%s): %s", rmName, projectKey, appName, e)
		WriteError(w, r, e)
		return
	}

	if !b {
		WriteError(w, r, sdk.ErrNoReposManagerClientAuth)
		return
	}

	client, err := repositoriesmanager.AuthorizedClient(db, projectKey, rmName)
	if err != nil {
		log.Warning("getCommitsHandler> Cannot get client got %s %s : %s", projectKey, rmName, err)
		WriteError(w, r, sdk.ErrNoReposManagerClientAuth)
		return
	}

	commits, err := client.Commits(fullname, since, until)
	if err != nil {
		log.Warning("getCommitsHandler> Cannot get commits: %s", err)
		WriteError(w, r, err)
		return
	}
	WriteJSON(w, r, commits, http.StatusOK)
}

func addHookOnRepositoriesManagerHandler(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {
	// Get project name in URL
	vars := mux.Vars(r)
	projectKey := vars["key"]
	appName := vars["permApplicationName"]
	rmName := vars["name"]

	var data map[string]string
	dataBytes, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Warning("addHookOnRepositoriesManagerHandler> Cannot read request body: %s", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if e := json.Unmarshal(dataBytes, &data); e != nil {
		log.Warning("addHookOnRepositoriesManagerHandler> Cannot parse request body: %s", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	repoFullname := data["repository_fullname"]
	pipelineName := data["pipeline_name"]

	application, err := application.LoadApplicationByName(db, projectKey, appName)
	if err != nil {
		WriteError(w, r, sdk.ErrApplicationNotFound)
		return
	}

	pipeline, err := pipeline.LoadPipeline(db, projectKey, pipelineName, false)
	if err != nil {
		WriteError(w, r, sdk.ErrPipelineNotFound)
		return
	}

	if !permission.AccessToPipeline(sdk.DefaultEnv.ID, pipeline.ID, c.User, permission.PermissionReadWriteExecute) {
		log.Warning("addHookOnRepositoriesManagerHandler> You don't have enought right on this pipeline %s", pipeline.Name)
		WriteError(w, r, sdk.ErrForbidden)
		return
	}

	rm, err := repositoriesmanager.LoadForProject(db, projectKey, rmName)
	if err != nil {
		log.Warning("addHookOnRepositoriesManagerHandler> error loading %s-%s: %s\n", projectKey, rmName, err)
		WriteError(w, r, sdk.ErrNoReposManager)
		return
	}

	b, e := repositoriesmanager.CheckApplicationIsAttached(db, rmName, projectKey, appName)
	if e != nil {
		log.Warning("addHookOnRepositoriesManagerHandler> Cannot check app (%s,%s,%s): %s", rmName, projectKey, appName, e)
		WriteError(w, r, e)
		return
	}

	if !b {
		WriteError(w, r, sdk.ErrNoReposManagerClientAuth)
		return
	}

	tx, err := db.Begin()
	if err != nil {
		log.Warning("addHookOnRepositoriesManagerHandler> Cannot start transaction: %s", err)
		WriteError(w, r, err)
		return
	}
	defer tx.Rollback()

	h, err := hook.CreateHook(tx, projectKey, rm, repoFullname, application, pipeline)
	if err != nil {
		WriteError(w, r, err)
		return
	}

	err = tx.Commit()
	if err != nil {
		log.Warning("addHookOnRepositoriesManagerHandler> Cannot commit transaction: %s", err)
		WriteError(w, r, err)
		return
	}

	WriteJSON(w, r, h, http.StatusCreated)
}

func deleteHookOnRepositoriesManagerHandler(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {
	// Get project name in URL
	vars := mux.Vars(r)
	projectKey := vars["key"]
	appName := vars["permApplicationName"]
	rmName := vars["name"]

	var data map[string]string
	dataBytes, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Warning("deleteHookOnRepositoriesManagerHandler> Cannot read request body: %s", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if e := json.Unmarshal(dataBytes, &data); e != nil {
		log.Warning("deleteHookOnRepositoriesManagerHandler> Cannot parse request body: %s", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	repoFullname := data["repository_fullname"]
	pipelineName := data["pipeline_name"]

	application, err := application.LoadApplicationByName(db, projectKey, appName)
	if err != nil {
		WriteError(w, r, sdk.ErrApplicationNotFound)
		return
	}

	pipeline, err := pipeline.LoadPipeline(db, projectKey, pipelineName, false)
	if err != nil {
		WriteError(w, r, sdk.ErrPipelineNotFound)
		return
	}

	if !permission.AccessToPipeline(sdk.DefaultEnv.ID, pipeline.ID, c.User, permission.PermissionReadWriteExecute) {
		log.Warning("deleteHookOnRepositoriesManagerHandler> You don't have enought right on this pipeline %s", pipeline.Name)
		WriteError(w, r, sdk.ErrForbidden)
		return
	}

	rm, err := repositoriesmanager.LoadForProject(db, projectKey, rmName)
	if err != nil {
		log.Warning("deleteHookOnRepositoriesManagerHandler> error loading %s-%s: %s\n", projectKey, rmName, err)
		WriteError(w, r, sdk.ErrNoReposManager)
		return
	}

	b, e := repositoriesmanager.CheckApplicationIsAttached(db, rmName, projectKey, appName)
	if e != nil {
		log.Warning("deleteHookOnRepositoriesManagerHandler> Cannot check app (%s,%s,%s): %s", rmName, projectKey, appName, e)
		WriteError(w, r, e)
		return
	}

	if !b {
		WriteError(w, r, sdk.ErrNoReposManagerClientAuth)
		return
	}

	client, err := repositoriesmanager.AuthorizedClient(db, projectKey, rmName)
	if err != nil {
		log.Warning("deleteHookOnRepositoriesManagerHandler> Cannot get client got %s %s : %s", projectKey, rmName, err)
		WriteError(w, r, sdk.ErrNoReposManagerClientAuth)
		return
	}

	t := strings.Split(repoFullname, "/")
	if len(t) != 2 {
		WriteError(w, r, sdk.ErrRepoNotFound)
		return
	}

	var h sdk.Hook

	h, err = hook.FindHook(db, application.ID, pipeline.ID, string(rm.Type), rm.URL, t[0], t[1])
	if err == sql.ErrNoRows {
		WriteError(w, r, sdk.ErrNoHook)
		return
	} else if err != nil {
		log.Warning("deleteHookOnRepositoriesManagerHandler> Cannot get hook: %s", err)
		WriteError(w, r, err)
		return
	}

	s := viper.GetString("api_url") + hook.HookLink
	link := fmt.Sprintf(s, h.UID, t[0], t[1])

	if err = client.DeleteHook(repoFullname, link); err != nil {
		log.Warning("deleteHookOnRepositoriesManagerHandler> Cannot delete hook on stash: %s", err)
		WriteError(w, r, err)
		return
	}

	if err = hook.DeleteHook(db, h.ID); err != nil {
		log.Warning("deleteHookOnRepositoriesManagerHandler> Cannot get hook: %s", err)
		WriteError(w, r, err)
		return
	}

}

func addApplicationFromRepositoriesManagerHandler(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {
	vars := mux.Vars(r)
	projectKey := vars["permProjectKey"]
	rmName := vars["name"]

	var data map[string]string
	dataBytes, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Warning("addApplicationFromRepositoriesManagerHandler> Cannot read request body: %s", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if e := json.Unmarshal(dataBytes, &data); e != nil {
		log.Warning("addApplicationFromRepositoriesManagerHandler> Cannot parse request body: %s", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	repoFullname := data["repository_fullname"]
	if repoFullname == "" {
		log.Warning("addApplicationFromRepositoriesManagerHandler>Repository fullname is mandatory")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	projectData, err := project.LoadProject(db, projectKey, c.User)
	if err != nil {
		log.Warning("addApplicationFromRepositoriesManagerHandler: Cannot load %s: %s\n", projectKey, err)
		WriteError(w, r, sdk.ErrInvalidProject)
		return
	}

	rm, err := repositoriesmanager.LoadForProject(db, projectKey, rmName)
	if err != nil {
		log.Warning("addApplicationFromRepositoriesManagerHandler> error loading %s-%s: %s\n", projectKey, rmName, err)
		WriteError(w, r, sdk.ErrNoReposManager)
		return
	}

	client, err := repositoriesmanager.AuthorizedClient(db, projectKey, rmName)
	if err != nil {
		log.Warning("addApplicationFromRepositoriesManagerHandler> Cannot get client got %s %s : %s", projectKey, rmName, err)
		WriteError(w, r, sdk.ErrNoReposManagerClientAuth)
		return
	}

	repo, err := client.RepoByFullname(repoFullname)
	if err != nil {
		log.Warning("addApplicationFromRepositoriesManagerHandler> Cannot get repo: %s", err)
		WriteError(w, r, sdk.ErrRepoNotFound)
		return
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
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	defer tx.Rollback()

	//Insert application in database
	if err = application.InsertApplication(tx, projectData, &app); err != nil {
		log.Warning("addApplicationFromRepositoriesManagerHandler> Cannot insert pipeline: %s\n", err)
		WriteError(w, r, err)
		return
	}

	//Fetch groups from project
	if err = group.LoadGroupByProject(tx, projectData); err != nil {
		log.Warning("addApplicationFromRepositoriesManagerHandler> Cannot load group from project: %s\n", err)
		WriteError(w, r, err)
		return
	}

	//Add the  groups on the application
	if err = group.InsertGroupsInApplication(tx, projectData.ProjectGroups, app.ID); err != nil {
		log.Warning("addApplicationFromRepositoriesManagerHandler> Cannot add groups on application: %s\n", err)
		WriteError(w, r, err)
		return
	}

	//Commit the transaction
	if err = tx.Commit(); err != nil {
		log.Warning("addApplicationFromRepositoriesManagerHandler> Cannot commit transaction: %s\n", err)
		WriteError(w, r, err)
		return
	}

	//Attach the application to the repositories manager
	if err := repositoriesmanager.InsertForApplication(db, rm, projectKey, app.Name, repoFullname); err != nil {
		log.Warning("addApplicationFromRepositoriesManagerHandler> Cannot attach application: %s", err)
		WriteError(w, r, err)
		return
	}

}
