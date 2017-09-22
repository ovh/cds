package api

import (
	"context"
	"io/ioutil"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/environment"
	"github.com/ovh/cds/engine/api/notification"
	"github.com/ovh/cds/engine/api/permission"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/sanity"
	"github.com/ovh/cds/engine/api/workflowv0"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

// Deprecated
func (api *API) attachPipelineToApplicationHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars["key"]
		appName := vars["permApplicationName"]
		pipelineName := vars["permPipelineKey"]

		proj, err := project.Load(api.mustDB(), api.Cache, key, getUser(ctx), project.LoadOptions.Default)
		if err != nil {
			log.Warning("addPipelineInApplicationHandler: Cannot load project: %s: %s\n", key, err)
			return err
		}

		pipeline, err := pipeline.LoadPipeline(api.mustDB(), key, pipelineName, true)
		if err != nil {
			log.Warning("addPipelineInApplicationHandler: Cannot load pipeline %s: %s\n", appName, err)
			return sdk.ErrNotFound
		}

		app, err := application.LoadByName(api.mustDB(), api.Cache, key, appName, getUser(ctx), application.LoadOptions.Default)
		if err != nil {
			log.Warning("addPipelineInApplicationHandler: Cannot load application %s: %s\n", appName, err)
			return sdk.ErrNotFound
		}

		if _, err := application.AttachPipeline(api.mustDB(), app.ID, pipeline.ID); err != nil {
			log.Warning("addPipelineInApplicationHandler: Cannot attach pipeline %s to application %s:  %s\n", pipelineName, appName, err)
			return err
		}

		if err := sanity.CheckPipeline(api.mustDB(), api.Cache, proj, pipeline); err != nil {
			log.Warning("addPipelineInApplicationHandler: Cannot check pipeline sanity: %s\n", err)
			return err
		}

		return WriteJSON(w, r, app, http.StatusOK)
	}
}

func (api *API) attachPipelinesToApplicationHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars["key"]
		appName := vars["permApplicationName"]

		var pipelines []string
		if err := UnmarshalBody(r, &pipelines); err != nil {
			return err
		}

		project, err := project.Load(api.mustDB(), api.Cache, key, getUser(ctx), project.LoadOptions.Default)
		if err != nil {
			log.Warning("attachPipelinesToApplicationHandler: Cannot load project: %s: %s\n", key, err)
			return err
		}

		app, err := application.LoadByName(api.mustDB(), api.Cache, key, appName, getUser(ctx), application.LoadOptions.Default)
		if err != nil {
			log.Warning("attachPipelinesToApplicationHandler: Cannot load application %s: %s\n", appName, err)
			return err
		}

		tx, errBegin := api.mustDB().Begin()
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

		if err := application.UpdateLastModified(tx, api.Cache, app, getUser(ctx)); err != nil {
			log.Warning("attachPipelinesToApplicationHandler: Cannot update application last modified date: %s\n", err)
			return err
		}

		if err := tx.Commit(); err != nil {
			log.Warning("attachPipelinesToApplicationHandler: Cannot commit transaction: %s\n", err)
			return err
		}

		if err := sanity.CheckProjectPipelines(api.mustDB(), api.Cache, project); err != nil {
			log.Warning("attachPipelinesToApplicationHandler: Cannot check project sanity: %s\n", err)
			return err
		}

		var errW error
		app.Workflows, errW = workflowv0.LoadCDTree(api.mustDB(), api.Cache, project.Key, app.Name, getUser(ctx), "", 0)
		if errW != nil {
			log.Warning("attachPipelinesToApplicationHandler: Cannot load application workflow: %s\n", errW)
			return errW
		}

		return WriteJSON(w, r, app, http.StatusOK)
	}
}

func (api *API) updatePipelinesToApplicationHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars["key"]
		appName := vars["permApplicationName"]

		var appPipelines []sdk.ApplicationPipeline
		if err := UnmarshalBody(r, &appPipelines); err != nil {
			return err
		}

		app, err := application.LoadByName(api.mustDB(), api.Cache, key, appName, getUser(ctx), application.LoadOptions.Default)
		if err != nil {
			log.Warning("updatePipelinesToApplicationHandler: Cannot load application %s: %s\n", appName, err)
			return sdk.ErrApplicationNotFound
		}

		tx, err := api.mustDB().Begin()
		if err != nil {
			log.Warning("updatePipelinesToApplicationHandler: Cannot start transaction: %s\n", err)
			return sdk.ErrUnknownError
		}
		defer tx.Rollback()

		for _, appPip := range appPipelines {
			err = application.UpdatePipelineApplication(tx, api.Cache, app, appPip.Pipeline.ID, appPip.Parameters, getUser(ctx))
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

		return WriteJSON(w, r, app, http.StatusOK)
	}
}

// DEPRECATED
func (api *API) updatePipelineToApplicationHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars["key"]
		appName := vars["permApplicationName"]
		pipelineName := vars["permPipelineKey"]

		pipeline, err := pipeline.LoadPipeline(api.mustDB(), key, pipelineName, false)
		if err != nil {
			log.Warning("updatePipelineToApplicationHandler: Cannot load pipeline %s: %s\n", appName, err)
			return sdk.ErrNotFound
		}

		app, err := application.LoadByName(api.mustDB(), api.Cache, key, appName, getUser(ctx))
		if err != nil {
			log.Warning("updatePipelineToApplicationHandler: Cannot load application %s: %s\n", appName, err)
			return sdk.ErrNotFound

		}

		// Get args in body
		data, err := ioutil.ReadAll(r.Body)
		if err != nil {
			return sdk.ErrWrongRequest
		}

		err = application.UpdatePipelineApplicationString(api.mustDB(), api.Cache, app, pipeline.ID, string(data), getUser(ctx))
		if err != nil {
			log.Warning("updatePipelineToApplicationHandler: Cannot update application %s pipeline %s parameters %s:  %s\n", appName, pipelineName, err)
			return err
		}

		return WriteJSON(w, r, app, http.StatusOK)
	}
}

func (api *API) getPipelinesInApplicationHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars["key"]
		appName := vars["permApplicationName"]

		pipelines, err := application.GetAllPipelines(api.mustDB(), key, appName)
		if err != nil {
			log.Warning("getPipelinesInApplicationHandler: Cannot load pipelines for application %s: %s\n", appName, err)
			return sdk.ErrNotFound
		}

		return WriteJSON(w, r, pipelines, http.StatusOK)
	}
}

func (api *API) removePipelineFromApplicationHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars["key"]
		appName := vars["permApplicationName"]
		pipelineName := vars["permPipelineKey"]

		a, errA := application.LoadByName(api.mustDB(), api.Cache, key, appName, getUser(ctx), application.LoadOptions.WithPipelines)
		if errA != nil {
			log.Warning("removePipelineFromApplicationHandler> Cannot load application: %s\n", errA)
			return errA
		}

		tx, errB := api.mustDB().Begin()
		if errB != nil {
			log.Warning("removePipelineFromApplicationHandler> Cannot start tx: %s\n", errB)
			return errB
		}
		defer tx.Rollback()

		if err := application.RemovePipeline(tx, key, appName, pipelineName); err != nil {
			log.Warning("removePipelineFromApplicationHandler: Cannot detach pipeline %s from %s: %s\n", pipelineName, appName, err)
			return err
		}

		if err := application.UpdateLastModified(tx, api.Cache, a, getUser(ctx)); err != nil {
			log.Warning("removePipelineFromApplicationHandler> Cannot update application last modified date: %s\n", err)
			return err
		}

		if err := tx.Commit(); err != nil {
			log.Warning("removePipelineFromApplicationHandler> Cannot commit tx: %s\n", err)
			return err
		}

		var errW error
		a.Workflows, errW = workflowv0.LoadCDTree(api.mustDB(), api.Cache, key, a.Name, getUser(ctx), "", 0)
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
}

func (api *API) getUserNotificationTypeHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		var types = []sdk.UserNotificationSettingsType{sdk.EmailUserNotification, sdk.JabberUserNotification}
		return WriteJSON(w, r, types, http.StatusOK)
	}
}

func (api *API) getUserNotificationStateValueHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		states := []sdk.UserNotificationEventType{sdk.UserNotificationAlways, sdk.UserNotificationChange, sdk.UserNotificationNever}
		return WriteJSON(w, r, states, http.StatusOK)
	}
}

func (api *API) getUserNotificationApplicationPipelineHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
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
		application, err := application.LoadByName(api.mustDB(), api.Cache, key, appName, getUser(ctx))
		if err != nil {
			log.Warning("getUserNotificationApplicationPipelineHandler> Cannot load application %s for project %s from db: %s\n", appName, key, err)
			return err
		}

		//Load pipeline
		pipeline, err := pipeline.LoadPipeline(api.mustDB(), key, pipelineName, false)
		if err != nil {
			log.Warning("getUserNotificationApplicationPipelineHandler> Cannot load pipeline %s: %s\n", pipelineName, err)
			return err
		}

		//Load environment
		env := &sdk.DefaultEnv
		if envName != "" {
			env, err = environment.LoadEnvironmentByName(api.mustDB(), key, envName)
			if err != nil {
				log.Warning("getUserNotificationApplicationPipelineHandler> cannot load environment %s: %s\n", envName, err)
				return err
			}
		}

		if !permission.AccessToEnvironment(env.ID, getUser(ctx), permission.PermissionRead) {
			log.Warning("getUserNotificationApplicationPipelineHandler> Cannot access to this environment")
			return sdk.ErrForbidden
		}

		//Load notifs
		notifs, err := notification.LoadUserNotificationSettings(api.mustDB(), application.ID, pipeline.ID, env.ID)
		if err != nil {
			log.Warning("getUserNotificationApplicationPipelineHandler> cannot load notification settings %s\n", err)
			return err
		}
		if notifs == nil {
			return WriteJSON(w, r, nil, http.StatusNoContent)
		}

		return WriteJSON(w, r, notifs, http.StatusOK)
	}
}

func (api *API) deleteUserNotificationApplicationPipelineHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
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
		applicationData, err := application.LoadByName(api.mustDB(), api.Cache, key, appName, getUser(ctx))
		if err != nil {
			log.Warning("deleteUserNotificationApplicationPipelineHandler> Cannot load application %s for project %s from db: %s\n", appName, key, err)
			return err

		}

		//Load pipeline
		pipeline, err := pipeline.LoadPipeline(api.mustDB(), key, pipelineName, false)
		if err != nil {
			log.Warning("deleteUserNotificationApplicationPipelineHandler> Cannot load pipeline %s: %s\n", pipelineName, err)
			return err

		}

		//Load environment
		env := &sdk.DefaultEnv
		if envName != "" && envName != sdk.DefaultEnv.Name {
			env, err = environment.LoadEnvironmentByName(api.mustDB(), key, envName)
			if err != nil {
				log.Warning("deleteUserNotificationApplicationPipelineHandler> cannot load environment %s: %s\n", envName, err)
				return err

			}
		}

		if !permission.AccessToEnvironment(env.ID, getUser(ctx), permission.PermissionReadWriteExecute) {
			log.Warning("deleteUserNotificationApplicationPipelineHandler> Cannot access to this environment")
			return sdk.ErrForbidden
		}

		tx, err := api.mustDB().Begin()
		if err != nil {
			log.Warning("deleteUserNotificationApplicationPipelineHandler> cannot start transaction: %s\n", err)
			return err
		}

		err = notification.DeleteNotification(tx, applicationData.ID, pipeline.ID, env.ID)
		if err != nil {
			log.Warning("deleteUserNotificationApplicationPipelineHandler> cannot delete user notification %s\n", err)
			return err
		}

		err = application.UpdateLastModified(tx, api.Cache, applicationData, getUser(ctx))
		if err != nil {
			log.Warning("deleteUserNotificationApplicationPipelineHandler> cannot update application last_modified date: %s\n", err)
			return err
		}

		err = tx.Commit()
		if err != nil {
			log.Warning("deleteUserNotificationApplicationPipelineHandler> cannot commit transaction: %s\n", err)
			return err

		}

		var errN error
		applicationData.Notifications, errN = notification.LoadAllUserNotificationSettings(api.mustDB(), applicationData.ID)
		if errN != nil {
			log.Warning("deleteUserNotificationApplicationPipelineHandler> cannot load notifications: %s\n", errN)
			return errN
		}
		return WriteJSON(w, r, applicationData, http.StatusOK)
	}
}

func (api *API) addNotificationsHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars["key"]
		appName := vars["permApplicationName"]

		var notifs []sdk.UserNotification
		if err := UnmarshalBody(r, &notifs); err != nil {
			return err
		}

		app, errApp := application.LoadByName(api.mustDB(), api.Cache, key, appName, getUser(ctx), application.LoadOptions.WithPipelines)
		if errApp != nil {
			log.Warning("addNotificationsHandler: Cannot load application: %s\n", errApp)
			return sdk.ErrWrongRequest
		}

		mapID := map[int64]string{}
		for _, appPip := range app.Pipelines {
			mapID[appPip.ID] = ""
		}

		tx, errBegin := api.mustDB().Begin()
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

			if !permission.AccessToEnvironment(n.Environment.ID, getUser(ctx), permission.PermissionReadWriteExecute) {
				log.Warning("addNotificationsHandler > Cannot access to this environment")
				return sdk.ErrForbidden
			}

			// Insert or update notification
			if err := notification.InsertOrUpdateUserNotificationSettings(tx, app.ID, n.Pipeline.ID, n.Environment.ID, &n); err != nil {
				log.Warning("addNotificationsHandler> cannot update user notification %s\n", err)
				return err

			}
		}

		if err := application.UpdateLastModified(tx, api.Cache, app, getUser(ctx)); err != nil {
			log.Warning("addNotificationsHandler> cannot update application last_modified date: %s\n", err)
			return err

		}

		if err := tx.Commit(); err != nil {
			log.Warning("addNotificationsHandler: Cannot commit transaction: %s\n", err)
			return err
		}

		var errNotif error
		app.Notifications, errNotif = notification.LoadAllUserNotificationSettings(api.mustDB(), app.ID)
		if errNotif != nil {
			log.Warning("addNotificationsHandler> cannot load notifications: %s\n", errNotif)
			return errNotif
		}

		return WriteJSON(w, r, app, http.StatusOK)
	}
}

func (api *API) updateUserNotificationApplicationPipelineHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars["key"]
		appName := vars["permApplicationName"]
		pipelineName := vars["permPipelineKey"]

		///Load application
		applicationData, err := application.LoadByName(api.mustDB(), api.Cache, key, appName, getUser(ctx))
		if err != nil {
			log.Warning("updateUserNotificationApplicationPipelineHandler> Cannot load application %s for project %s from db: %s\n", appName, key, err)
			return err
		}

		//Load pipeline
		pipeline, err := pipeline.LoadPipeline(api.mustDB(), key, pipelineName, false)
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

		if !permission.AccessToEnvironment(notifs.Environment.ID, getUser(ctx), permission.PermissionReadWriteExecute) {
			log.Warning("updateUserNotificationApplicationPipelineHandler> Cannot access to this environment")
			return sdk.ErrForbidden
		}

		tx, err := api.mustDB().Begin()
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

		err = application.UpdateLastModified(tx, api.Cache, applicationData, getUser(ctx))
		if err != nil {
			log.Warning("updateUserNotificationApplicationPipelineHandler> cannot update application last_modified date: %s\n", err)
			return err

		}

		err = tx.Commit()
		if err != nil {
			log.Warning("updateUserNotificationApplicationPipelineHandler> cannot commit transaction: %s\n", err)
			return err

		}

		var errNotif error
		applicationData.Notifications, errNotif = notification.LoadAllUserNotificationSettings(api.mustDB(), applicationData.ID)
		if errNotif != nil {
			log.Warning("updateUserNotificationApplicationPipelineHandler> Cannot load notifications: %s\n", errNotif)
			return errNotif
		}

		return WriteJSON(w, r, applicationData, http.StatusOK)

	}
}
