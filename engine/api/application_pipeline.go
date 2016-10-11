package main

import (
	"database/sql"
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/context"
	"github.com/ovh/cds/engine/api/environment"
	"github.com/ovh/cds/engine/api/notification"
	"github.com/ovh/cds/engine/api/permission"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/sanity"
	"github.com/ovh/cds/engine/log"
	"github.com/ovh/cds/sdk"
)

func attachPipelineToApplicationHandler(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {

	vars := mux.Vars(r)
	key := vars["key"]
	appName := vars["permApplicationName"]
	pipelineName := vars["permPipelineKey"]

	project, err := project.LoadProject(db, key, c.User)
	if err != nil {
		log.Warning("addPipelineInApplicationHandler: Cannot load project: %s: %s\n", key, err)
		WriteError(w, r, err)
		return
	}

	pipeline, err := pipeline.LoadPipeline(db, key, pipelineName, true)
	if err != nil {
		log.Warning("addPipelineInApplicationHandler: Cannot load pipeline %s: %s\n", appName, err)
		w.WriteHeader(http.StatusNotFound)
		return
	}

	app, err := application.LoadApplicationByName(db, key, appName)
	if err != nil {
		log.Warning("addPipelineInApplicationHandler: Cannot load application %s: %s\n", appName, err)
		w.WriteHeader(http.StatusNotFound)
		return
	}

	err = application.AttachPipeline(db, app.ID, pipeline.ID)
	if err != nil {
		log.Warning("addPipelineInApplicationHandler: Cannot attach pipeline %s to application %s:  %s\n", pipelineName, appName, err)
		WriteError(w, r, err)
		return
	}

	err = sanity.CheckPipeline(db, project, pipeline)
	if err != nil {
		log.Warning("addPipelineInApplicationHandler: Cannot check pipeline sanity: %s\n", err)
		WriteError(w, r, err)
		return
	}

	k := cache.Key("application", key, "*"+appName+"*")
	cache.DeleteAll(k)

	w.WriteHeader(http.StatusOK)
}

func updatePipelinesToApplicationHandler(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {

	vars := mux.Vars(r)
	key := vars["key"]
	appName := vars["permApplicationName"]

	// Get args in body
	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Warning("updatePipelinesToApplicationHandler>Cannot read body: %s\n", err)
		WriteError(w, r, sdk.ErrUnknownError)
		return
	}

	var appPipelines []sdk.ApplicationPipeline
	err = json.Unmarshal([]byte(data), &appPipelines)
	if err != nil {
		log.Warning("updatePipelinesToApplicationHandler: Cannot unmarshal body: %s\n", err)
		WriteError(w, r, sdk.ErrUnknownError)
		return
	}

	app, err := application.LoadApplicationByName(db, key, appName)
	if err != nil {
		log.Warning("updatePipelinesToApplicationHandler: Cannot load application %s: %s\n", appName, err)
		WriteError(w, r, sdk.ErrApplicationNotFound)
		return
	}

	tx, err := db.Begin()
	if err != nil {
		log.Warning("updatePipelinesToApplicationHandler: Cannot start transaction: %s\n", err)
		WriteError(w, r, sdk.ErrUnknownError)
		return
	}
	defer tx.Rollback()

	for _, appPip := range appPipelines {
		err = application.UpdatePipelineApplication(tx, app.ID, appPip.Pipeline.ID, appPip.Parameters)
		if err != nil {
			log.Warning("updatePipelinesToApplicationHandler: Cannot update  application pipeline  %s/%s parameters: %s\n", appName, appPip.Pipeline.Name, err)
			WriteError(w, r, sdk.ErrUnknownError)
			return
		}
	}
	err = tx.Commit()
	if err != nil {
		log.Warning("updatePipelinesToApplicationHandler: Cannot commit transaction: %s\n", err)
		WriteError(w, r, sdk.ErrUnknownError)
		return
	}

	k := cache.Key("application", key, "*"+appName+"*")
	cache.DeleteAll(k)

	w.WriteHeader(http.StatusOK)
}

func updatePipelineToApplicationHandler(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {

	vars := mux.Vars(r)
	key := vars["key"]
	appName := vars["permApplicationName"]
	pipelineName := vars["permPipelineKey"]

	pipeline, err := pipeline.LoadPipeline(db, key, pipelineName, false)
	if err != nil {
		log.Warning("updatePipelineToApplicationHandler: Cannot load pipeline %s: %s\n", appName, err)
		w.WriteHeader(http.StatusNotFound)
		return
	}

	app, err := application.LoadApplicationByName(db, key, appName)
	if err != nil {
		log.Warning("updatePipelineToApplicationHandler: Cannot load application %s: %s\n", appName, err)
		w.WriteHeader(http.StatusNotFound)
		return
	}

	// Get args in body
	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		log.Warning("updatePipelineToApplicationHandler>Cannot read body: %s\n", err)
		return
	}

	err = application.UpdatePipelineApplicationString(db, app.ID, pipeline.ID, string(data))
	if err != nil {
		log.Warning("updatePipelineToApplicationHandler: Cannot update application %s pipeline %s parameters %s:  %s\n", appName, pipelineName, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	k := cache.Key("application", key, "*"+appName+"*")
	cache.DeleteAll(k)

	w.WriteHeader(http.StatusOK)
}

func getPipelinesInApplicationHandler(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {
	vars := mux.Vars(r)
	key := vars["key"]
	appName := vars["permApplicationName"]

	pipelines, err := application.GetAllPipelines(db, key, appName)
	if err != nil {
		log.Warning("getPipelinesInApplicationHandler: Cannot load pipelines for application %s: %s\n", appName, err)
		w.WriteHeader(http.StatusNotFound)
		return
	}

	WriteJSON(w, r, pipelines, http.StatusOK)
}

func removePipelineFromApplicationHandler(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {

	vars := mux.Vars(r)
	key := vars["key"]
	appName := vars["permApplicationName"]
	pipelineName := vars["permPipelineKey"]

	tx, err := db.Begin()
	if err != nil {
		log.Warning("removePipelineFromApplicationHandler> Cannot start tx: %s\n", err)
		WriteError(w, r, err)
		return
	}
	defer tx.Rollback()

	err = application.RemovePipeline(tx, key, appName, pipelineName)
	if err != nil {
		log.Warning("removePipelineFromApplicationHandler: Cannot detach pipeline %s from %s: %s\n", pipelineName, appName, err)
		WriteError(w, r, err)
		return
	}

	err = tx.Commit()
	if err != nil {
		log.Warning("removePipelineFromApplicationHandler> Cannot commit tx: %s\n", err)
		WriteError(w, r, err)
		return
	}

	k := cache.Key("application", key, "*"+appName+"*")
	cache.DeleteAll(k)

	w.WriteHeader(http.StatusOK)
}

func getUserNotificationTypeHandler(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {
	var types = []sdk.UserNotificationSettingsType{sdk.EmailUserNotification, sdk.JabberUserNotification}
	WriteJSON(w, r, types, http.StatusOK)
}

func getUserNotificationStateValueHandler(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {
	states := []sdk.UserNotificationEventType{sdk.UserNotificationAlways, sdk.UserNotificationChange, sdk.UserNotificationNever}
	WriteJSON(w, r, states, http.StatusOK)
}

func getUserNotificationApplicationPipelineHandler(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {
	vars := mux.Vars(r)
	key := vars["key"]
	appName := vars["permApplicationName"]
	pipelineName := vars["permPipelineKey"]

	err := r.ParseForm()
	if err != nil {
		log.Warning("getPipelineHistoryHandler> Cannot parse form: %s\n", err)
		WriteError(w, r, sdk.ErrUnknownError)
		return
	}
	envName := r.Form.Get("envName")

	//Load application
	application, err := application.LoadApplicationByName(db, key, appName)
	if err != nil {
		log.Warning("getUserNotificationApplicationPipelineHandler> Cannot load application %s for project %s from db: %s\n", appName, key, err)
		WriteError(w, r, err)
		return
	}

	//Load pipeline
	pipeline, err := pipeline.LoadPipeline(db, key, pipelineName, false)
	if err != nil {
		log.Warning("getUserNotificationApplicationPipelineHandler> Cannot load pipeline %s: %s\n", pipelineName, err)
		WriteError(w, r, err)
		return
	}

	//Load environment
	env := &sdk.DefaultEnv
	if envName != "" {
		env, err = environment.LoadEnvironmentByName(db, key, envName)
		if err != nil {
			log.Warning("getUserNotificationApplicationPipelineHandler> cannot load environment %s: %s\n", envName, err)
			WriteError(w, r, err)
			return
		}
	}

	if env.ID != sdk.DefaultEnv.ID {
		if !permission.AccessToEnvironment(env.ID, c.User, permission.PermissionRead) {
			log.Warning("getUserNotificationApplicationPipelineHandler> Cannot access to this environment")
			WriteError(w, r, sdk.ErrForbidden)
			return
		}

	}

	//Load notifs
	notifs, err := notification.LoadUserNotificationSettings(db, application.ID, pipeline.ID, env.ID)
	if err != nil {
		log.Warning("getUserNotificationApplicationPipelineHandler> cannot load notification settings %s\n", err)
		WriteError(w, r, err)
		return
	}
	if notifs == nil {
		WriteJSON(w, r, nil, http.StatusNoContent)
		return
	}

	WriteJSON(w, r, notifs, http.StatusOK)
	return
}

func deleteUserNotificationApplicationPipelineHandler(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {
	vars := mux.Vars(r)
	key := vars["key"]
	appName := vars["permApplicationName"]
	pipelineName := vars["permPipelineKey"]

	err := r.ParseForm()
	if err != nil {
		log.Warning("deleteUserNotificationApplicationPipelineHandler> Cannot parse form: %s\n", err)
		WriteError(w, r, sdk.ErrUnknownError)
		return
	}
	envName := r.Form.Get("envName")

	///Load application
	applicationData, err := application.LoadApplicationByName(db, key, appName)
	if err != nil {
		log.Warning("deleteUserNotificationApplicationPipelineHandler> Cannot load application %s for project %s from db: %s\n", appName, key, err)
		WriteError(w, r, err)
		return
	}

	//Load pipeline
	pipeline, err := pipeline.LoadPipeline(db, key, pipelineName, false)
	if err != nil {
		log.Warning("deleteUserNotificationApplicationPipelineHandler> Cannot load pipeline %s: %s\n", pipelineName, err)
		WriteError(w, r, err)
		return
	}

	//Load environment
	env := &sdk.DefaultEnv
	if envName != "" && envName != sdk.DefaultEnv.Name {
		env, err = environment.LoadEnvironmentByName(db, key, envName)
		if err != nil {
			log.Warning("deleteUserNotificationApplicationPipelineHandler> cannot load environment %s: %s\n", envName, err)
			WriteError(w, r, err)
			return
		}
	}

	if env.ID != sdk.DefaultEnv.ID {
		if !permission.AccessToEnvironment(env.ID, c.User, permission.PermissionReadWriteExecute) {
			log.Warning("deleteUserNotificationApplicationPipelineHandler> Cannot access to this environment")
			WriteError(w, r, sdk.ErrForbidden)
			return
		}

	}

	tx, err := db.Begin()
	if err != nil {
		log.Warning("deleteUserNotificationApplicationPipelineHandler> cannot start transaction: %s\n", err)
		WriteError(w, r, err)
		return
	}

	err = notification.DeleteNotification(tx, applicationData.ID, pipeline.ID, env.ID)
	if err != nil {
		log.Warning("deleteUserNotificationApplicationPipelineHandler> cannot delete user notification %s\n", err)
		WriteError(w, r, err)
		return
	}

	err = application.UpdateLastModified(tx, applicationData.ID)
	if err != nil {
		log.Warning("deleteUserNotificationApplicationPipelineHandler> cannot update application last_modified date: %s\n", err)
		WriteError(w, r, err)
		return
	}

	err = tx.Commit()
	if err != nil {
		log.Warning("deleteUserNotificationApplicationPipelineHandler> cannot commit transaction: %s\n", err)
		WriteError(w, r, err)
		return
	}

	k := cache.Key("application", key, "*"+appName+"*")
	cache.DeleteAll(k)

	w.WriteHeader(http.StatusOK)
}
func updateUserNotificationApplicationPipelineHandler(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {
	vars := mux.Vars(r)
	key := vars["key"]
	appName := vars["permApplicationName"]
	pipelineName := vars["permPipelineKey"]

	///Load application
	applicationData, err := application.LoadApplicationByName(db, key, appName)
	if err != nil {
		log.Warning("updateUserNotificationApplicationPipelineHandler> Cannot load application %s for project %s from db: %s\n", appName, key, err)
		WriteError(w, r, err)
		return
	}

	//Load pipeline
	pipeline, err := pipeline.LoadPipeline(db, key, pipelineName, false)
	if err != nil {
		log.Warning("updateUserNotificationApplicationPipelineHandler> Cannot load pipeline %s: %s\n", pipelineName, err)
		WriteError(w, r, err)
		return
	}

	//Parse notification settings
	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		WriteError(w, r, sdk.ErrParseUserNotification)
		return
	}
	notifs, err := notification.ParseUserNotification(data)
	if err != nil {
		WriteError(w, r, err)
		return
	}

	//Load environment
	if notifs.Environment.ID == 0 {
		notifs.Environment = sdk.DefaultEnv
	}

	if notifs.Environment.ID != sdk.DefaultEnv.ID {
		if !permission.AccessToEnvironment(notifs.Environment.ID, c.User, permission.PermissionReadWriteExecute) {
			log.Warning("updateUserNotificationApplicationPipelineHandler> Cannot access to this environment")
			WriteError(w, r, sdk.ErrForbidden)
			return
		}

	}

	tx, err := db.Begin()
	if err != nil {
		log.Warning("updateUserNotificationApplicationPipelineHandler> cannot start transaction: %s\n", err)
		WriteError(w, r, err)
		return
	}
	defer tx.Rollback()

	//Insert or update notification
	if err := notification.InsertOrUpdateUserNotificationSettings(tx, applicationData.ID, pipeline.ID, notifs.Environment.ID, notifs); err != nil {
		log.Warning("updateUserNotificationApplicationPipelineHandler> cannot update user notification %s\n", err)
		WriteError(w, r, err)
		return
	}

	err = application.UpdateLastModified(tx, applicationData.ID)
	if err != nil {
		log.Warning("updateUserNotificationApplicationPipelineHandler> cannot update application last_modified date: %s\n", err)
		WriteError(w, r, err)
		return
	}

	err = tx.Commit()
	if err != nil {
		log.Warning("updateUserNotificationApplicationPipelineHandler> cannot commit transaction: %s\n", err)
		WriteError(w, r, err)
		return
	}

	k := cache.Key("application", key, "*"+appName+"*")
	cache.DeleteAll(k)

	WriteJSON(w, r, notifs, http.StatusOK)
	return
}
