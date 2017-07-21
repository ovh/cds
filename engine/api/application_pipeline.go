package main

import (
	"io/ioutil"
	"net/http"

	"github.com/go-gorp/gorp"
	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/businesscontext"
	"github.com/ovh/cds/engine/api/environment"
	"github.com/ovh/cds/engine/api/notification"
	"github.com/ovh/cds/engine/api/permission"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/sanity"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

// Deprecated
func attachPipelineToApplicationHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *businesscontext.Ctx) error {
	vars := mux.Vars(r)
	pipelineName := vars["permPipelineKey"]

	pipeline, err := pipeline.LoadPipeline(db, c.Project.Key, pipelineName, true)
	if err != nil {
		log.Warning("addPipelineInApplicationHandler: Cannot load pipeline %s: %s", c.Application.Name, err)
		return sdk.ErrNotFound
	}

	if _, err := application.AttachPipeline(db, c.Application.ID, pipeline.ID); err != nil {
		log.Warning("addPipelineInApplicationHandler: Cannot attach pipeline %s to application %s:  %s", pipelineName, c.Application.Name, err)
		return err
	}

	if err := sanity.CheckPipeline(db, c.Project, pipeline); err != nil {
		log.Warning("addPipelineInApplicationHandler: Cannot check pipeline sanity: %s", err)
		return err
	}

	return WriteJSON(w, r, c.Application, http.StatusOK)
}

func attachPipelinesToApplicationHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *businesscontext.Ctx) error {
	var pipelines []string
	if err := UnmarshalBody(r, &pipelines); err != nil {
		return err
	}

	tx, errBegin := db.Begin()
	if errBegin != nil {
		log.Warning("attachPipelinesToApplicationHandler: Cannot begin transaction: %s", errBegin)
		return errBegin
	}

	for _, pipName := range pipelines {
		pipeline, err := pipeline.LoadPipeline(tx, c.Project.Key, pipName, true)
		if err != nil {
			log.Warning("attachPipelinesToApplicationHandler: Cannot load pipeline %s: %s", pipName, err)
			return err
		}

		id, errA := application.AttachPipeline(tx, c.Application.ID, pipeline.ID)
		if errA != nil {
			log.Warning("attachPipelinesToApplicationHandler: Cannot attach pipeline %s to application:  %s", pipName, errA)
			return errA
		}

		c.Application.Pipelines = append(c.Application.Pipelines, sdk.ApplicationPipeline{
			Pipeline: *pipeline,
			ID:       id,
		})

	}

	if err := application.UpdateLastModified(tx, c.Application, c.User); err != nil {
		log.Warning("attachPipelinesToApplicationHandler: Cannot update application last modified date: %s", err)
		return err
	}

	if err := tx.Commit(); err != nil {
		log.Warning("attachPipelinesToApplicationHandler: Cannot commit transaction: %s", err)
		return err
	}

	if err := sanity.CheckProjectPipelines(db, c.Project); err != nil {
		log.Warning("attachPipelinesToApplicationHandler: Cannot check project sanity: %s", err)
		return err
	}

	var errW error
	c.Application.Workflows, errW = workflow.LoadCDTree(db, c.Project.Key, c.Application.Name, c.User, "", 0)
	if errW != nil {
		log.Warning("attachPipelinesToApplicationHandler: Cannot load application workflow: %s", errW)
		return errW
	}

	return WriteJSON(w, r, c.Application, http.StatusOK)
}

func updatePipelinesToApplicationHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *businesscontext.Ctx) error {
	var appPipelines []sdk.ApplicationPipeline
	if err := UnmarshalBody(r, &appPipelines); err != nil {
		return err
	}

	tx, err := db.Begin()
	if err != nil {
		log.Warning("updatePipelinesToApplicationHandler: Cannot start transaction: %s", err)
		return sdk.ErrUnknownError
	}
	defer tx.Rollback()

	for _, appPip := range appPipelines {
		err = application.UpdatePipelineApplication(tx, c.Application, appPip.Pipeline.ID, appPip.Parameters, c.User)
		if err != nil {
			log.Warning("updatePipelinesToApplicationHandler: Cannot update  application pipeline  %s parameters: %s", appPip.Pipeline.Name, err)
			return sdk.ErrUnknownError
		}
	}
	err = tx.Commit()
	if err != nil {
		log.Warning("updatePipelinesToApplicationHandler: Cannot commit transaction: %s", err)
		return sdk.ErrUnknownError
	}

	return WriteJSON(w, r, c.Application, http.StatusOK)
}

// DEPRECATED
func updatePipelineToApplicationHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *businesscontext.Ctx) error {
	vars := mux.Vars(r)
	pipelineName := vars["permPipelineKey"]

	pipeline, err := pipeline.LoadPipeline(db, c.Project.Key, pipelineName, false)
	if err != nil {
		log.Warning("updatePipelineToApplicationHandler: Cannot load pipeline %s: %s", pipelineName, err)
		return sdk.ErrNotFound
	}

	// Get args in body
	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return sdk.ErrWrongRequest
	}

	err = application.UpdatePipelineApplicationString(db, c.Application, pipeline.ID, string(data), c.User)
	if err != nil {
		log.Warning("updatePipelineToApplicationHandler: Cannot update application %s pipeline %s parameters %s:  %s", c.Application.Name, pipelineName, err)
		return err
	}

	return WriteJSON(w, r, c.Application, http.StatusOK)
}

func getPipelinesInApplicationHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *businesscontext.Ctx) error {
	pipelines, err := application.GetAllPipelines(db, c.Project.Key, c.Project.Name)
	if err != nil {
		log.Warning("getPipelinesInApplicationHandler: Cannot load pipelines for application %s: %s", c.Application.Name, err)
		return sdk.ErrNotFound
	}

	return WriteJSON(w, r, pipelines, http.StatusOK)
}

func removePipelineFromApplicationHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *businesscontext.Ctx) error {
	vars := mux.Vars(r)
	appName := vars["permApplicationName"]
	pipelineName := vars["permPipelineKey"]

	a, errA := application.LoadByName(db, c.Project.Key, appName, c.User, application.LoadOptions.WithPipelines)
	if errA != nil {
		log.Warning("removePipelineFromApplicationHandler> Cannot load application: %s", errA)
		return errA
	}

	tx, errB := db.Begin()
	if errB != nil {
		log.Warning("removePipelineFromApplicationHandler> Cannot start tx: %s", errB)
		return errB
	}
	defer tx.Rollback()

	if err := application.RemovePipeline(tx, c.Project.Key, appName, pipelineName); err != nil {
		log.Warning("removePipelineFromApplicationHandler: Cannot detach pipeline %s from %s: %s", pipelineName, appName, err)
		return err
	}

	if err := application.UpdateLastModified(tx, a, c.User); err != nil {
		log.Warning("removePipelineFromApplicationHandler> Cannot update application last modified date: %s", err)
		return err
	}

	if err := tx.Commit(); err != nil {
		log.Warning("removePipelineFromApplicationHandler> Cannot commit tx: %s", err)
		return err
	}

	var errW error
	a.Workflows, errW = workflow.LoadCDTree(db, c.Project.Key, a.Name, c.User, "", 0)
	if errW != nil {
		log.Warning("removePipelineFromApplicationHandler> Cannot load workflow: %s", errW)
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

func getUserNotificationTypeHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *businesscontext.Ctx) error {
	var types = []sdk.UserNotificationSettingsType{sdk.EmailUserNotification, sdk.JabberUserNotification}
	return WriteJSON(w, r, types, http.StatusOK)
}

func getUserNotificationStateValueHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *businesscontext.Ctx) error {
	states := []sdk.UserNotificationEventType{sdk.UserNotificationAlways, sdk.UserNotificationChange, sdk.UserNotificationNever}
	return WriteJSON(w, r, states, http.StatusOK)
}

func getUserNotificationApplicationPipelineHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *businesscontext.Ctx) error {
	vars := mux.Vars(r)
	pipelineName := vars["permPipelineKey"]

	err := r.ParseForm()
	if err != nil {
		log.Warning("getPipelineHistoryHandler> Cannot parse form: %s", err)
		return sdk.ErrUnknownError
	}
	envName := r.Form.Get("envName")

	//Load pipeline
	pipeline, err := pipeline.LoadPipeline(db, c.Project.Key, pipelineName, false)
	if err != nil {
		log.Warning("getUserNotificationApplicationPipelineHandler> Cannot load pipeline %s: %s", pipelineName, err)
		return err
	}

	//Load environment
	env := &sdk.DefaultEnv
	if envName != "" {
		env, err = environment.LoadEnvironmentByName(db, c.Project.Key, envName)
		if err != nil {
			log.Warning("getUserNotificationApplicationPipelineHandler> cannot load environment %s: %s", envName, err)
			return err
		}
	}

	if !permission.AccessToEnvironment(env.ID, c.User, permission.PermissionRead) {
		log.Warning("getUserNotificationApplicationPipelineHandler> Cannot access to this environment")
		return sdk.ErrForbidden
	}

	//Load notifs
	notifs, err := notification.LoadUserNotificationSettings(db, c.Application.ID, pipeline.ID, env.ID)
	if err != nil {
		log.Warning("getUserNotificationApplicationPipelineHandler> cannot load notification settings %s", err)
		return err
	}
	if notifs == nil {
		return WriteJSON(w, r, nil, http.StatusNoContent)
	}

	return WriteJSON(w, r, notifs, http.StatusOK)
}

func deleteUserNotificationApplicationPipelineHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *businesscontext.Ctx) error {
	vars := mux.Vars(r)
	pipelineName := vars["permPipelineKey"]

	err := r.ParseForm()
	if err != nil {
		log.Warning("deleteUserNotificationApplicationPipelineHandler> Cannot parse form: %s", err)
		return sdk.ErrUnknownError

	}
	envName := r.Form.Get("envName")

	//Load pipeline
	pipeline, err := pipeline.LoadPipeline(db, c.Project.Key, pipelineName, false)
	if err != nil {
		log.Warning("deleteUserNotificationApplicationPipelineHandler> Cannot load pipeline %s: %s", pipelineName, err)
		return err

	}

	//Load environment
	env := &sdk.DefaultEnv
	if envName != "" && envName != sdk.DefaultEnv.Name {
		env, err = environment.LoadEnvironmentByName(db, c.Project.Key, envName)
		if err != nil {
			log.Warning("deleteUserNotificationApplicationPipelineHandler> cannot load environment %s: %s", envName, err)
			return err

		}
	}

	if !permission.AccessToEnvironment(env.ID, c.User, permission.PermissionReadWriteExecute) {
		log.Warning("deleteUserNotificationApplicationPipelineHandler> Cannot access to this environment")
		return sdk.ErrForbidden
	}

	tx, err := db.Begin()
	if err != nil {
		log.Warning("deleteUserNotificationApplicationPipelineHandler> cannot start transaction: %s", err)
		return err
	}

	err = notification.DeleteNotification(tx, c.Application.ID, pipeline.ID, env.ID)
	if err != nil {
		log.Warning("deleteUserNotificationApplicationPipelineHandler> cannot delete user notification %s", err)
		return err
	}

	err = application.UpdateLastModified(tx, c.Application, c.User)
	if err != nil {
		log.Warning("deleteUserNotificationApplicationPipelineHandler> cannot update application last_modified date: %s", err)
		return err
	}

	err = tx.Commit()
	if err != nil {
		log.Warning("deleteUserNotificationApplicationPipelineHandler> cannot commit transaction: %s", err)
		return err

	}

	var errN error
	c.Application.Notifications, errN = notification.LoadAllUserNotificationSettings(db, c.Application.ID)
	if errN != nil {
		log.Warning("deleteUserNotificationApplicationPipelineHandler> cannot load notifications: %s", errN)
		return errN
	}
	return WriteJSON(w, r, c.Application, http.StatusOK)
}

func addNotificationsHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *businesscontext.Ctx) error {
	vars := mux.Vars(r)
	key := vars["key"]
	appName := vars["permApplicationName"]

	var notifs []sdk.UserNotification
	if err := UnmarshalBody(r, &notifs); err != nil {
		return err
	}

	app, errApp := application.LoadByName(db, key, appName, c.User, application.LoadOptions.WithPipelines)
	if errApp != nil {
		log.Warning("addNotificationsHandler: Cannot load application: %s", errApp)
		return sdk.ErrWrongRequest
	}

	mapID := map[int64]string{}
	for _, appPip := range app.Pipelines {
		mapID[appPip.ID] = ""
	}

	tx, errBegin := db.Begin()
	if errBegin != nil {
		log.Warning("addNotificationsHandler: Cannot begin transaction: %s", errBegin)
		return errBegin
	}
	defer tx.Rollback()

	for _, n := range notifs {
		if _, ok := mapID[n.ApplicationPipelineID]; !ok {
			log.Warning("addNotificationsHandler: Cannot get pipeline for this application: %s")
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
			log.Warning("addNotificationsHandler> cannot update user notification %s", err)
			return err

		}
	}

	if err := application.UpdateLastModified(tx, app, c.User); err != nil {
		log.Warning("addNotificationsHandler> cannot update application last_modified date: %s", err)
		return err

	}

	if err := tx.Commit(); err != nil {
		log.Warning("addNotificationsHandler: Cannot commit transaction: %s", err)
		return err
	}

	var errNotif error
	app.Notifications, errNotif = notification.LoadAllUserNotificationSettings(db, app.ID)
	if errNotif != nil {
		log.Warning("addNotificationsHandler> cannot load notifications: %s", errNotif)
		return errNotif
	}

	return WriteJSON(w, r, app, http.StatusOK)
}

func updateUserNotificationApplicationPipelineHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *businesscontext.Ctx) error {
	vars := mux.Vars(r)
	pipelineName := vars["permPipelineKey"]

	//Load pipeline
	pipeline, err := pipeline.LoadPipeline(db, c.Project.Key, pipelineName, false)
	if err != nil {
		log.Warning("updateUserNotificationApplicationPipelineHandler> Cannot load pipeline %s: %s", pipelineName, err)
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
		log.Warning("updateUserNotificationApplicationPipelineHandler> cannot start transaction: %s", err)
		return err

	}
	defer tx.Rollback()

	// Insert or update notification
	if err := notification.InsertOrUpdateUserNotificationSettings(tx, c.Application.ID, pipeline.ID, notifs.Environment.ID, notifs); err != nil {
		log.Warning("updateUserNotificationApplicationPipelineHandler> cannot update user notification %s", err)
		return err

	}

	err = application.UpdateLastModified(tx, c.Application, c.User)
	if err != nil {
		log.Warning("updateUserNotificationApplicationPipelineHandler> cannot update application last_modified date: %s", err)
		return err

	}

	err = tx.Commit()
	if err != nil {
		log.Warning("updateUserNotificationApplicationPipelineHandler> cannot commit transaction: %s", err)
		return err

	}

	var errNotif error
	c.Application.Notifications, errNotif = notification.LoadAllUserNotificationSettings(db, c.Application.ID)
	if errNotif != nil {
		log.Warning("updateUserNotificationApplicationPipelineHandler> Cannot load notifications: %s", errNotif)
		return errNotif
	}

	return WriteJSON(w, r, c.Application, http.StatusOK)

}
