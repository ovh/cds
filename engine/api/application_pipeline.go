package main

import (
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
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/engine/log"
	"github.com/ovh/cds/sdk"
)

// Deprecated
func attachPipelineToApplicationHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	vars := mux.Vars(r)
	key := vars["key"]
	appName := vars["permApplicationName"]
	pipelineName := vars["permPipelineKey"]

	proj, err := project.Load(db, key, c.User, project.LoadOptions.Default)
	if err != nil {
		log.Warning("addPipelineInApplicationHandler: Cannot load project: %s: %s\n", key, err)
		return err
	}

	pipeline, err := pipeline.LoadPipeline(db, key, pipelineName, true)
	if err != nil {
		log.Warning("addPipelineInApplicationHandler: Cannot load pipeline %s: %s\n", appName, err)
		return sdk.ErrNotFound
	}

	app, err := application.LoadByName(db, key, appName, c.User, application.LoadOptions.Default)
	if err != nil {
		log.Warning("addPipelineInApplicationHandler: Cannot load application %s: %s\n", appName, err)
		return sdk.ErrNotFound
	}

	if _, err := application.AttachPipeline(db, app.ID, pipeline.ID); err != nil {
		log.Warning("addPipelineInApplicationHandler: Cannot attach pipeline %s to application %s:  %s\n", pipelineName, appName, err)
		return err
	}

	if err := sanity.CheckPipeline(db, proj, pipeline); err != nil {
		log.Warning("addPipelineInApplicationHandler: Cannot check pipeline sanity: %s\n", err)
		return err
	}

	k := cache.Key("application", key, "*"+appName+"*")
	cache.DeleteAll(k)

	return WriteJSON(w, r, app, http.StatusOK)
}

func attachPipelinesToApplicationHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	vars := mux.Vars(r)
	key := vars["key"]
	appName := vars["permApplicationName"]

	var pipelines []string
	if err := UnmarshalBody(r, &pipelines); err != nil {
		return err
	}

	project, err := project.Load(db, key, c.User, project.LoadOptions.Default)
	if err != nil {
		log.Warning("attachPipelinesToApplicationHandler: Cannot load project: %s: %s\n", key, err)
		return err
	}

	app, err := application.LoadByName(db, key, appName, c.User, application.LoadOptions.Default)
	if err != nil {
		log.Warning("attachPipelinesToApplicationHandler: Cannot load application %s: %s\n", appName, err)
		return err
	}

	tx, errBegin := db.Begin()
	if errBegin != nil {
		log.Warning("attachPipelinesToApplicationHandler: Cannot begin transaction: %s\n", errBegin)
		return errBegin
	}

	for _, pipName := range pipelines {
		pipeline, err := pipeline.LoadPipeline(tx, key, pipName, true)
		if err != nil {
			log.Warning("attachPipelinesToApplicationHandler: Cannot load pipeline %s: %s\n", pipName, err)
			return err
		}

		id, errA := application.AttachPipeline(tx, app.ID, pipeline.ID)
		if errA != nil {
			log.Warning("attachPipelinesToApplicationHandler: Cannot attach pipeline %s to application %s:  %s\n", pipName, appName, errA)
			return errA
		}

		app.Pipelines = append(app.Pipelines, sdk.ApplicationPipeline{
			Pipeline: *pipeline,
			ID:       id,
		})

	}

	if err := application.UpdateLastModified(tx, app, c.User); err != nil {
		log.Warning("attachPipelinesToApplicationHandler: Cannot update application last modified date: %s\n", err)
		return err
	}

	if err := tx.Commit(); err != nil {
		log.Warning("attachPipelinesToApplicationHandler: Cannot commit transaction: %s\n", err)
		return err
	}

	if err := sanity.CheckProjectPipelines(db, project); err != nil {
		log.Warning("attachPipelinesToApplicationHandler: Cannot check project sanity: %s\n", err)
		return err
	}

	var errW error
	app.Workflows, errW = workflow.LoadCDTree(db, project.Key, app.Name, c.User)
	if errW != nil {
		log.Warning("attachPipelinesToApplicationHandler: Cannot load application workflow: %s\n", errW)
		return errW
	}

	k := cache.Key("application", key, "*"+appName+"*")
	cache.DeleteAll(k)

	return WriteJSON(w, r, app, http.StatusOK)
}

func updatePipelinesToApplicationHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	vars := mux.Vars(r)
	key := vars["key"]
	appName := vars["permApplicationName"]

	var appPipelines []sdk.ApplicationPipeline
	if err := UnmarshalBody(r, &appPipelines); err != nil {
		return err
	}

	app, err := application.LoadByName(db, key, appName, c.User, application.LoadOptions.Default)
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
		err = application.UpdatePipelineApplication(tx, app, appPip.Pipeline.ID, appPip.Parameters, c.User)
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

// DEPRECATED
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

	app, err := application.LoadByName(db, key, appName, c.User)
	if err != nil {
		log.Warning("updatePipelineToApplicationHandler: Cannot load application %s: %s\n", appName, err)
		return sdk.ErrNotFound

	}

	// Get args in body
	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return sdk.ErrWrongRequest
	}

	err = application.UpdatePipelineApplicationString(db, app, pipeline.ID, string(data), c.User)
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

	a, errA := application.LoadByName(db, key, appName, c.User, application.LoadOptions.WithPipelines)
	if errA != nil {
		log.Warning("removePipelineFromApplicationHandler> Cannot load application: %s\n", errA)
		return errA
	}

	tx, errB := db.Begin()
	if errB != nil {
		log.Warning("removePipelineFromApplicationHandler> Cannot start tx: %s\n", errB)
		return errB
	}
	defer tx.Rollback()

	if err := application.RemovePipeline(tx, key, appName, pipelineName); err != nil {
		log.Warning("removePipelineFromApplicationHandler: Cannot detach pipeline %s from %s: %s\n", pipelineName, appName, err)
		return err
	}

	if err := application.UpdateLastModified(tx, a, c.User); err != nil {
		log.Warning("removePipelineFromApplicationHandler> Cannot update application last modified date: %s\n", err)
		return err
	}

	if err := tx.Commit(); err != nil {
		log.Warning("removePipelineFromApplicationHandler> Cannot commit tx: %s\n", err)
		return err
	}

	k := cache.Key("application", key, "*"+appName+"*")
	cache.DeleteAll(k)

	var errW error
	a.Workflows, errW = workflow.LoadCDTree(db, key, a.Name, c.User)
	if errW != nil {
		log.Warning("removePipelineFromApplicationHandler> Cannot load workflow: %s\n", errW)
		return errW
	}

	// Remove pipeline from struct
	var indexPipeline int
	for i, appPip := range a.Pipelines {
		if appPip.Pipeline.Name == pipelineName {
			indexPipeline = i
			break
		}
	}
	a.Pipelines = append(a.Pipelines[:indexPipeline], a.Pipelines[indexPipeline+1:]...)

	return WriteJSON(w, r, a, http.StatusOK)
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
	application, err := application.LoadByName(db, key, appName, c.User)
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
	applicationData, err := application.LoadByName(db, key, appName, c.User)
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

	err = application.UpdateLastModified(tx, applicationData, c.User)
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
	if err := UnmarshalBody(r, &notifs); err != nil {
		return err
	}

	app, errApp := application.LoadByName(db, key, appName, c.User, application.LoadOptions.WithPipelines)
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

	if err := application.UpdateLastModified(tx, app, c.User); err != nil {
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
	applicationData, err := application.LoadByName(db, key, appName, c.User)
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
	notifs := &sdk.UserNotification{}
	if err := UnmarshalBody(r, &notifs); err != nil {
		return err
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

	err = application.UpdateLastModified(tx, applicationData, c.User)
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
