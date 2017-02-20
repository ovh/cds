package main

import (
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/go-gorp/gorp"
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

func attachPipelineToApplicationHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {

	vars := mux.Vars(r)
	key := vars["key"]
	appName := vars["permApplicationName"]
	pipelineName := vars["permPipelineKey"]

	project, err := project.Load(db, key, c.User)
	if err != nil {
		log.Warning("addPipelineInApplicationHandler: Cannot load project: %s: %s\n", key, err)
		return err
	}

	pipeline, err := pipeline.LoadPipeline(db, key, pipelineName, true)
	if err != nil {
		log.Warning("addPipelineInApplicationHandler: Cannot load pipeline %s: %s\n", appName, err)
		return sdk.ErrNotFound
	}

	app, err := application.LoadApplicationByName(db, key, appName)
	if err != nil {
		log.Warning("addPipelineInApplicationHandler: Cannot load application %s: %s\n", appName, err)
		return sdk.ErrNotFound
	}

	err = application.AttachPipeline(db, app.ID, pipeline.ID)
	if err != nil {
		log.Warning("addPipelineInApplicationHandler: Cannot attach pipeline %s to application %s:  %s\n", pipelineName, appName, err)
		return err
	}

	err = sanity.CheckPipeline(db, project, pipeline)
	if err != nil {
		log.Warning("addPipelineInApplicationHandler: Cannot check pipeline sanity: %s\n", err)
		return err
	}

	k := cache.Key("application", key, "*"+appName+"*")
	cache.DeleteAll(k)

	return WriteJSON(w, r, app, http.StatusOK)
}

func updatePipelinesToApplicationHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	vars := mux.Vars(r)
	key := vars["key"]
	appName := vars["permApplicationName"]

	// Get args in body
	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Warning("updatePipelinesToApplicationHandler>Cannot read body: %s\n", err)
		return sdk.ErrUnknownError
	}

	var appPipelines []sdk.ApplicationPipeline
	err = json.Unmarshal([]byte(data), &appPipelines)
	if err != nil {
		log.Warning("updatePipelinesToApplicationHandler: Cannot unmarshal body: %s\n", err)
		return sdk.ErrUnknownError
	}

	app, err := application.LoadApplicationByName(db, key, appName)
	if err != nil {
		log.Warning("updatePipelinesToApplicationHandler: Cannot load application %s: %s\n", appName, err)
		return sdk.ErrApplicationNotFound
	}

	tx, err := db.Begin()
	if err != nil {
		log.Warning("updatePipelinesToApplicationHandler: Cannot start transaction: %s\n", err)
		return sdk.ErrUnknownError
	}
	defer tx.Rollback()

	for _, appPip := range appPipelines {
		err = application.UpdatePipelineApplication(tx, app, appPip.Pipeline.ID, appPip.Parameters)
		if err != nil {
			log.Warning("updatePipelinesToApplicationHandler: Cannot update  application pipeline  %s/%s parameters: %s\n", appName, appPip.Pipeline.Name, err)
			return sdk.ErrUnknownError
		}
	}
	err = tx.Commit()
	if err != nil {
		log.Warning("updatePipelinesToApplicationHandler: Cannot commit transaction: %s\n", err)
		return sdk.ErrUnknownError
	}

	k := cache.Key("application", key, "*"+appName+"*")
	cache.DeleteAll(k)

	return WriteJSON(w, r, app, http.StatusOK)
}

func updatePipelineToApplicationHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	vars := mux.Vars(r)
	key := vars["key"]
	appName := vars["permApplicationName"]
	pipelineName := vars["permPipelineKey"]

	pipeline, err := pipeline.LoadPipeline(db, key, pipelineName, false)
	if err != nil {
		log.Warning("updatePipelineToApplicationHandler: Cannot load pipeline %s: %s\n", appName, err)
		return sdk.ErrNotFound
	}

	app, err := application.LoadApplicationByName(db, key, appName)
	if err != nil {
		log.Warning("updatePipelineToApplicationHandler: Cannot load application %s: %s\n", appName, err)
		return sdk.ErrNotFound

	}

	// Get args in body
	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return sdk.ErrWrongRequest
	}

	err = application.UpdatePipelineApplicationString(db, app, pipeline.ID, string(data))
	if err != nil {
		log.Warning("updatePipelineToApplicationHandler: Cannot update application %s pipeline %s parameters %s:  %s\n", appName, pipelineName, err)
		return err
	}

	k := cache.Key("application", key, "*"+appName+"*")
	cache.DeleteAll(k)

	return WriteJSON(w, r, app, http.StatusOK)
}

func getPipelinesInApplicationHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	vars := mux.Vars(r)
	key := vars["key"]
	appName := vars["permApplicationName"]

	pipelines, err := application.GetAllPipelines(db, key, appName)
	if err != nil {
		log.Warning("getPipelinesInApplicationHandler: Cannot load pipelines for application %s: %s\n", appName, err)
		return sdk.ErrNotFound
	}

	return WriteJSON(w, r, pipelines, http.StatusOK)
}

func removePipelineFromApplicationHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	vars := mux.Vars(r)
	key := vars["key"]
	appName := vars["permApplicationName"]
	pipelineName := vars["permPipelineKey"]

	tx, err := db.Begin()
	if err != nil {
		log.Warning("removePipelineFromApplicationHandler> Cannot start tx: %s\n", err)
		return err
	}
	defer tx.Rollback()

	err = application.RemovePipeline(tx, key, appName, pipelineName)
	if err != nil {
		log.Warning("removePipelineFromApplicationHandler: Cannot detach pipeline %s from %s: %s\n", pipelineName, appName, err)
		return err
	}

	err = tx.Commit()
	if err != nil {
		log.Warning("removePipelineFromApplicationHandler> Cannot commit tx: %s\n", err)
		return err
	}

	k := cache.Key("application", key, "*"+appName+"*")
	cache.DeleteAll(k)

	return nil
}

func getUserNotificationTypeHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	var types = []sdk.UserNotificationSettingsType{sdk.EmailUserNotification, sdk.JabberUserNotification}
	return WriteJSON(w, r, types, http.StatusOK)
}

func getUserNotificationStateValueHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	states := []sdk.UserNotificationEventType{sdk.UserNotificationAlways, sdk.UserNotificationChange, sdk.UserNotificationNever}
	return WriteJSON(w, r, states, http.StatusOK)
}

func getUserNotificationApplicationPipelineHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	vars := mux.Vars(r)
	key := vars["key"]
	appName := vars["permApplicationName"]
	pipelineName := vars["permPipelineKey"]

	err := r.ParseForm()
	if err != nil {
		log.Warning("getPipelineHistoryHandler> Cannot parse form: %s\n", err)
		return sdk.ErrUnknownError
	}
	envName := r.Form.Get("envName")

	//Load application
	application, err := application.LoadApplicationByName(db, key, appName)
	if err != nil {
		log.Warning("getUserNotificationApplicationPipelineHandler> Cannot load application %s for project %s from db: %s\n", appName, key, err)
		return err
	}

	//Load pipeline
	pipeline, err := pipeline.LoadPipeline(db, key, pipelineName, false)
	if err != nil {
		log.Warning("getUserNotificationApplicationPipelineHandler> Cannot load pipeline %s: %s\n", pipelineName, err)
		return err
	}

	//Load environment
	env := &sdk.DefaultEnv
	if envName != "" {
		env, err = environment.LoadEnvironmentByName(db, key, envName)
		if err != nil {
			log.Warning("getUserNotificationApplicationPipelineHandler> cannot load environment %s: %s\n", envName, err)
			return err
		}
	}

	if !permission.AccessToEnvironment(env.ID, c.User, permission.PermissionRead) {
		log.Warning("getUserNotificationApplicationPipelineHandler> Cannot access to this environment")
		return sdk.ErrForbidden
	}

	//Load notifs
	notifs, err := notification.LoadUserNotificationSettings(db, application.ID, pipeline.ID, env.ID)
	if err != nil {
		log.Warning("getUserNotificationApplicationPipelineHandler> cannot load notification settings %s\n", err)
		return err
	}
	if notifs == nil {
		return WriteJSON(w, r, nil, http.StatusNoContent)
	}

	return WriteJSON(w, r, notifs, http.StatusOK)
}

func deleteUserNotificationApplicationPipelineHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	vars := mux.Vars(r)
	key := vars["key"]
	appName := vars["permApplicationName"]
	pipelineName := vars["permPipelineKey"]

	err := r.ParseForm()
	if err != nil {
		log.Warning("deleteUserNotificationApplicationPipelineHandler> Cannot parse form: %s\n", err)
		return sdk.ErrUnknownError

	}
	envName := r.Form.Get("envName")

	///Load application
	applicationData, err := application.LoadApplicationByName(db, key, appName)
	if err != nil {
		log.Warning("deleteUserNotificationApplicationPipelineHandler> Cannot load application %s for project %s from db: %s\n", appName, key, err)
		return err

	}

	//Load pipeline
	pipeline, err := pipeline.LoadPipeline(db, key, pipelineName, false)
	if err != nil {
		log.Warning("deleteUserNotificationApplicationPipelineHandler> Cannot load pipeline %s: %s\n", pipelineName, err)
		return err

	}

	//Load environment
	env := &sdk.DefaultEnv
	if envName != "" && envName != sdk.DefaultEnv.Name {
		env, err = environment.LoadEnvironmentByName(db, key, envName)
		if err != nil {
			log.Warning("deleteUserNotificationApplicationPipelineHandler> cannot load environment %s: %s\n", envName, err)
			return err

		}
	}

	if !permission.AccessToEnvironment(env.ID, c.User, permission.PermissionReadWriteExecute) {
		log.Warning("deleteUserNotificationApplicationPipelineHandler> Cannot access to this environment")
		return sdk.ErrForbidden
	}

	tx, err := db.Begin()
	if err != nil {
		log.Warning("deleteUserNotificationApplicationPipelineHandler> cannot start transaction: %s\n", err)
		return err
	}

	err = notification.DeleteNotification(tx, applicationData.ID, pipeline.ID, env.ID)
	if err != nil {
		log.Warning("deleteUserNotificationApplicationPipelineHandler> cannot delete user notification %s\n", err)
		return err
	}

	err = application.UpdateLastModified(tx, applicationData)
	if err != nil {
		log.Warning("deleteUserNotificationApplicationPipelineHandler> cannot update application last_modified date: %s\n", err)
		return err
	}

	err = tx.Commit()
	if err != nil {
		log.Warning("deleteUserNotificationApplicationPipelineHandler> cannot commit transaction: %s\n", err)
		return err

	}

	k := cache.Key("application", key, "*"+appName+"*")
	cache.DeleteAll(k)

	var errN error
	applicationData.Notifications, errN = notification.LoadAllUserNotificationSettings(db, applicationData.ID)
	if errN != nil {
		log.Warning("deleteUserNotificationApplicationPipelineHandler> cannot load notifications: %s\n", errN)
		return errN
	}
	return WriteJSON(w, r, applicationData, http.StatusOK)
}

func addNotificationsHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	vars := mux.Vars(r)
	key := vars["key"]
	appName := vars["permApplicationName"]

	var notifs []sdk.UserNotification
	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Warning("addNotificationsHandler: Cannot read body: %s\n", err)
		return sdk.ErrWrongRequest
	}
	if err := json.Unmarshal(data, &notifs); err != nil {
		log.Warning("addNotificationsHandler: Cannot unmarshal request: %s\n", err)
		return sdk.ErrWrongRequest
	}

	app, errApp := application.LoadApplicationByName(db, key, appName)
	if errApp != nil {
		log.Warning("addNotificationsHandler: Cannot load application: %s\n", errApp)
		return sdk.ErrWrongRequest
	}

	mapID := map[int64]string{}
	for _, appPip := range app.Pipelines {
		mapID[appPip.ID] = ""
	}

	tx, errBegin := db.Begin()
	if errBegin != nil {
		log.Warning("addNotificationsHandler: Cannot begin transaction: %s\n", errBegin)
		return errBegin
	}
	defer tx.Rollback()

	for _, n := range notifs {
		if _, ok := mapID[n.ApplicationPipelineID]; !ok {
			log.Warning("addNotificationsHandler: Cannot get pipeline for this application: %s\n")
			return sdk.ErrWrongRequest
		}

		//Load environment
		if n.Environment.ID == 0 {
			n.Environment = sdk.DefaultEnv
		}

		if !permission.AccessToEnvironment(n.Environment.ID, c.User, permission.PermissionReadWriteExecute) {
			log.Warning("addNotificationsHandler > Cannot access to this environment")
			return sdk.ErrForbidden
		}

		// Insert or update notification
		if err := notification.InsertOrUpdateUserNotificationSettings(tx, app.ID, n.Pipeline.ID, n.Environment.ID, &n); err != nil {
			log.Warning("addNotificationsHandler> cannot update user notification %s\n", err)
			return err

		}
	}

	if err := application.UpdateLastModified(tx, app); err != nil {
		log.Warning("addNotificationsHandler> cannot update application last_modified date: %s\n", err)
		return err

	}

	if err := tx.Commit(); err != nil {
		log.Warning("addNotificationsHandler: Cannot commit transaction: %s\n", err)
		return err
	}

	var errNotif error
	app.Notifications, errNotif = notification.LoadAllUserNotificationSettings(db, app.ID)
	if errNotif != nil {
		log.Warning("addNotificationsHandler> cannot load notifications: %s\n", errNotif)
		return errNotif
	}

	return WriteJSON(w, r, app, http.StatusOK)
}

func updateUserNotificationApplicationPipelineHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	vars := mux.Vars(r)
	key := vars["key"]
	appName := vars["permApplicationName"]
	pipelineName := vars["permPipelineKey"]

	///Load application
	applicationData, err := application.LoadApplicationByName(db, key, appName)
	if err != nil {
		log.Warning("updateUserNotificationApplicationPipelineHandler> Cannot load application %s for project %s from db: %s\n", appName, key, err)
		return err

	}

	//Load pipeline
	pipeline, err := pipeline.LoadPipeline(db, key, pipelineName, false)
	if err != nil {
		log.Warning("updateUserNotificationApplicationPipelineHandler> Cannot load pipeline %s: %s\n", pipelineName, err)
		return err

	}

	//Parse notification settings
	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return sdk.ErrWrongRequest
	}
	notifs := &sdk.UserNotification{}
	if err := json.Unmarshal(data, notifs); err != nil {
		return sdk.ErrParseUserNotification
	}

	//Load environment
	if notifs.Environment.ID == 0 {
		notifs.Environment = sdk.DefaultEnv
	}

	if !permission.AccessToEnvironment(notifs.Environment.ID, c.User, permission.PermissionReadWriteExecute) {
		log.Warning("updateUserNotificationApplicationPipelineHandler> Cannot access to this environment")
		return sdk.ErrForbidden
	}

	tx, err := db.Begin()
	if err != nil {
		log.Warning("updateUserNotificationApplicationPipelineHandler> cannot start transaction: %s\n", err)
		return err

	}
	defer tx.Rollback()

	// Insert or update notification
	if err := notification.InsertOrUpdateUserNotificationSettings(tx, applicationData.ID, pipeline.ID, notifs.Environment.ID, notifs); err != nil {
		log.Warning("updateUserNotificationApplicationPipelineHandler> cannot update user notification %s\n", err)
		return err

	}

	err = application.UpdateLastModified(tx, applicationData)
	if err != nil {
		log.Warning("updateUserNotificationApplicationPipelineHandler> cannot update application last_modified date: %s\n", err)
		return err

	}

	err = tx.Commit()
	if err != nil {
		log.Warning("updateUserNotificationApplicationPipelineHandler> cannot commit transaction: %s\n", err)
		return err

	}

	k := cache.Key("application", key, "*"+appName+"*")
	cache.DeleteAll(k)

	var errNotif error
	applicationData.Notifications, errNotif = notification.LoadAllUserNotificationSettings(db, applicationData.ID)
	if errNotif != nil {
		log.Warning("updateUserNotificationApplicationPipelineHandler> Cannot load notifications: %s\n", errNotif)
		return errNotif
	}

	return WriteJSON(w, r, applicationData, http.StatusOK)

}
