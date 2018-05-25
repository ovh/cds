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
	"github.com/ovh/cds/engine/api/workflowv0"
	"github.com/ovh/cds/sdk"
)

// Deprecated
func (api *API) attachPipelineToApplicationHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars["key"]
		appName := vars["permApplicationName"]
		pipelineName := vars["permPipelineKey"]

		pipeline, err := pipeline.LoadPipeline(api.mustDB(), key, pipelineName, true)
		if err != nil {
			return sdk.WrapError(sdk.ErrNotFound, "addPipelineInApplicationHandler> Cannot load pipeline %s: %s", appName, err)
		}

		app, err := application.LoadByName(api.mustDB(), api.Cache, key, appName, getUser(ctx), application.LoadOptions.Default)
		if err != nil {
			return sdk.WrapError(sdk.ErrNotFound, "addPipelineInApplicationHandler> Cannot load application %s: %s", appName, err)
		}

		if _, err := application.AttachPipeline(api.mustDB(), app.ID, pipeline.ID); err != nil {
			return sdk.WrapError(err, "addPipelineInApplicationHandler> Cannot attach pipeline %s to application %s", pipelineName, appName)
		}
		return WriteJSON(w, app, http.StatusOK)
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

		app, err := application.LoadByName(api.mustDB(), api.Cache, key, appName, getUser(ctx), application.LoadOptions.Default)
		if err != nil {
			return sdk.WrapError(err, "attachPipelinesToApplicationHandler: Cannot load application %s", appName)
		}

		tx, errBegin := api.mustDB().Begin()
		if errBegin != nil {
			return sdk.WrapError(errBegin, "attachPipelinesToApplicationHandler: Cannot begin transaction")
		}

		for _, pipName := range pipelines {
			pip, err := pipeline.LoadPipeline(tx, key, pipName, true)
			if err != nil {
				return sdk.WrapError(err, "attachPipelinesToApplicationHandler: Cannot load pipeline %s", pipName)
			}

			id, errA := application.AttachPipeline(tx, app.ID, pip.ID)
			if errA != nil {
				return sdk.WrapError(errA, "attachPipelinesToApplicationHandler: Cannot attach pipeline %s to application %s", pipName, appName)
			}

			app.Pipelines = append(app.Pipelines, sdk.ApplicationPipeline{
				Pipeline: *pip,
				ID:       id,
			})

			projTmp := &sdk.Project{Key: key}
			if err := pipeline.UpdatePipelineLastModified(tx, api.Cache, projTmp, pip, getUser(ctx)); err != nil {
				return sdk.WrapError(err, "attachPipelinesToApplicationHandler> Cannot update pipeline last modified date")
			}
		}

		if err := application.UpdateLastModified(tx, api.Cache, app, getUser(ctx)); err != nil {
			return sdk.WrapError(err, "attachPipelinesToApplicationHandler: Cannot update application last modified date")
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "attachPipelinesToApplicationHandler: Cannot commit transaction")
		}

		var errW error
		app.Workflows, errW = workflowv0.LoadCDTree(api.mustDB(), api.Cache, key, app.Name, getUser(ctx), "", "", 0)
		if errW != nil {
			return sdk.WrapError(errW, "attachPipelinesToApplicationHandler: Cannot load application workflow")
		}

		return WriteJSON(w, app, http.StatusOK)
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
			return sdk.WrapError(sdk.ErrApplicationNotFound, "updatePipelinesToApplicationHandler: Cannot load application %s", appName)
		}

		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WrapError(sdk.ErrUnknownError, "updatePipelinesToApplicationHandler: Cannot start transaction")
		}
		defer tx.Rollback()

		for _, appPip := range appPipelines {
			err = application.UpdatePipelineApplication(tx, api.Cache, app, appPip.Pipeline.ID, appPip.Parameters, getUser(ctx))
			if err != nil {
				return sdk.WrapError(sdk.ErrUnknownError, "updatePipelinesToApplicationHandler: Cannot update  application pipeline  %s/%s parameters", appName, appPip.Pipeline.Name)
			}
		}
		err = tx.Commit()
		if err != nil {
			return sdk.WrapError(sdk.ErrUnknownError, "updatePipelinesToApplicationHandler: Cannot commit transaction")
		}

		return WriteJSON(w, app, http.StatusOK)
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
			return sdk.WrapError(sdk.ErrNotFound, "updatePipelineToApplicationHandler: Cannot load pipeline %s", appName)
		}

		app, err := application.LoadByName(api.mustDB(), api.Cache, key, appName, getUser(ctx))
		if err != nil {
			return sdk.WrapError(sdk.ErrNotFound, "updatePipelineToApplicationHandler: Cannot load application %s", appName)

		}

		// Get args in body
		data, err := ioutil.ReadAll(r.Body)
		if err != nil {
			return sdk.ErrWrongRequest
		}

		err = application.UpdatePipelineApplicationString(api.mustDB(), api.Cache, app, pipeline.ID, string(data), getUser(ctx))
		if err != nil {
			return sdk.WrapError(err, "updatePipelineToApplicationHandler: Cannot update application %s pipeline %s parameters %s", appName, pipelineName)
		}

		return WriteJSON(w, app, http.StatusOK)
	}
}

func (api *API) getPipelinesInApplicationHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars["key"]
		appName := vars["permApplicationName"]

		pipelines, err := application.GetAllPipelines(api.mustDB(), key, appName)
		if err != nil {
			return sdk.WrapError(sdk.ErrNotFound, "getPipelinesInApplicationHandler: Cannot load pipelines for application %s", appName)
		}

		return WriteJSON(w, pipelines, http.StatusOK)
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
			return sdk.WrapError(errA, "removePipelineFromApplicationHandler> Cannot load application")
		}

		tx, errB := api.mustDB().Begin()
		if errB != nil {
			return sdk.WrapError(errB, "removePipelineFromApplicationHandler> Cannot start tx")
		}
		defer tx.Rollback()

		if err := application.RemovePipeline(tx, key, appName, pipelineName); err != nil {
			return sdk.WrapError(err, "removePipelineFromApplicationHandler: Cannot detach pipeline %s from %s", pipelineName, appName)
		}

		if err := application.UpdateLastModified(tx, api.Cache, a, getUser(ctx)); err != nil {
			return sdk.WrapError(err, "removePipelineFromApplicationHandler> Cannot update application last modified date")
		}

		// Remove pipeline from struct
		var indexPipeline int
		for i, appPip := range a.Pipelines {
			if appPip.Pipeline.Name == pipelineName {
				indexPipeline = i
				break
			}
		}

		projTmp := &sdk.Project{Key: key}
		if err := pipeline.UpdatePipelineLastModified(tx, api.Cache, projTmp, &a.Pipelines[indexPipeline].Pipeline, getUser(ctx)); err != nil {
			return sdk.WrapError(err, "removePipelineFromApplicationHandler> Cannot update pipeline last modified date")
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "removePipelineFromApplicationHandler> Cannot commit tx")
		}

		var errW error
		a.Workflows, errW = workflowv0.LoadCDTree(api.mustDB(), api.Cache, key, a.Name, getUser(ctx), "", "", 0)
		if errW != nil {
			return sdk.WrapError(errW, "removePipelineFromApplicationHandler> Cannot load workflow")
		}

		a.Pipelines = append(a.Pipelines[:indexPipeline], a.Pipelines[indexPipeline+1:]...)

		return WriteJSON(w, a, http.StatusOK)
	}
}

func (api *API) getUserNotificationTypeHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		var types = []sdk.UserNotificationSettingsType{sdk.EmailUserNotification, sdk.JabberUserNotification}
		return WriteJSON(w, types, http.StatusOK)
	}
}

func (api *API) getUserNotificationStateValueHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		states := []sdk.UserNotificationEventType{sdk.UserNotificationAlways, sdk.UserNotificationChange, sdk.UserNotificationNever}
		return WriteJSON(w, states, http.StatusOK)
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
			return sdk.WrapError(sdk.ErrUnknownError, "getPipelineHistoryHandler> Cannot parse form")
		}
		envName := r.Form.Get("envName")

		//Load application
		application, err := application.LoadByName(api.mustDB(), api.Cache, key, appName, getUser(ctx))
		if err != nil {
			return sdk.WrapError(err, "getUserNotificationApplicationPipelineHandler> Cannot load application %s for project %s from db", appName, key)
		}

		//Load pipeline
		pipeline, err := pipeline.LoadPipeline(api.mustDB(), key, pipelineName, false)
		if err != nil {
			return sdk.WrapError(err, "getUserNotificationApplicationPipelineHandler> Cannot load pipeline %s", pipelineName)
		}

		//Load environment
		env := &sdk.DefaultEnv
		if envName != "" {
			env, err = environment.LoadEnvironmentByName(api.mustDB(), key, envName)
			if err != nil {
				return sdk.WrapError(err, "getUserNotificationApplicationPipelineHandler> cannot load environment %s", envName)
			}
		}

		if !permission.AccessToEnvironment(key, env.Name, getUser(ctx), permission.PermissionRead) {
			return sdk.WrapError(sdk.ErrForbidden, "getUserNotificationApplicationPipelineHandler> Cannot access to this environment")
		}

		//Load notifs
		notifs, err := notification.LoadUserNotificationSettings(api.mustDB(), application.ID, pipeline.ID, env.ID)
		if err != nil {
			return sdk.WrapError(err, "getUserNotificationApplicationPipelineHandler> cannot load notification settings")
		}
		if notifs == nil {
			return WriteJSON(w, nil, http.StatusOK)
		}

		return WriteJSON(w, notifs, http.StatusOK)
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
			return sdk.WrapError(sdk.ErrUnknownError, "deleteUserNotificationApplicationPipelineHandler> Cannot parse form")

		}
		envName := r.Form.Get("envName")

		///Load application
		applicationData, err := application.LoadByName(api.mustDB(), api.Cache, key, appName, getUser(ctx))
		if err != nil {
			return sdk.WrapError(err, "deleteUserNotificationApplicationPipelineHandler> Cannot load application %s for project %s from db", appName, key)

		}

		//Load pipeline
		pipeline, err := pipeline.LoadPipeline(api.mustDB(), key, pipelineName, false)
		if err != nil {
			return sdk.WrapError(err, "deleteUserNotificationApplicationPipelineHandler> Cannot load pipeline %s", pipelineName)

		}

		//Load environment
		env := &sdk.DefaultEnv
		if envName != "" && envName != sdk.DefaultEnv.Name {
			env, err = environment.LoadEnvironmentByName(api.mustDB(), key, envName)
			if err != nil {
				return sdk.WrapError(err, "deleteUserNotificationApplicationPipelineHandler> cannot load environment %s", envName)

			}
		}

		if !permission.AccessToEnvironment(key, env.Name, getUser(ctx), permission.PermissionReadWriteExecute) {
			return sdk.WrapError(sdk.ErrForbidden, "deleteUserNotificationApplicationPipelineHandler> Cannot access to this environment")
		}

		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WrapError(err, "deleteUserNotificationApplicationPipelineHandler> cannot start transaction")
		}

		err = notification.DeleteNotification(tx, applicationData.ID, pipeline.ID, env.ID)
		if err != nil {
			return sdk.WrapError(err, "deleteUserNotificationApplicationPipelineHandler> cannot delete user notification")
		}

		err = application.UpdateLastModified(tx, api.Cache, applicationData, getUser(ctx))
		if err != nil {
			return sdk.WrapError(err, "deleteUserNotificationApplicationPipelineHandler> cannot update application last_modified date")
		}

		err = tx.Commit()
		if err != nil {
			return sdk.WrapError(err, "deleteUserNotificationApplicationPipelineHandler> cannot commit transaction")

		}

		var errN error
		applicationData.Notifications, errN = notification.LoadAllUserNotificationSettings(api.mustDB(), applicationData.ID)
		if errN != nil {
			return sdk.WrapError(errN, "deleteUserNotificationApplicationPipelineHandler> cannot load notifications")
		}
		return WriteJSON(w, applicationData, http.StatusOK)
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
			return sdk.WrapError(sdk.ErrWrongRequest, "addNotificationsHandler: Cannot load application")
		}

		mapID := map[int64]string{}
		for _, appPip := range app.Pipelines {
			mapID[appPip.ID] = ""
		}

		tx, errBegin := api.mustDB().Begin()
		if errBegin != nil {
			return sdk.WrapError(errBegin, "addNotificationsHandler: Cannot begin transaction")
		}
		defer tx.Rollback()

		for _, n := range notifs {
			if _, ok := mapID[n.ApplicationPipelineID]; !ok {
				return sdk.WrapError(sdk.ErrWrongRequest, "addNotificationsHandler: Cannot get pipeline for this application")
			}

			//Load environment
			if n.Environment.ID == 0 {
				n.Environment = sdk.DefaultEnv
			}

			if !permission.AccessToEnvironment(key, n.Environment.Name, getUser(ctx), permission.PermissionReadWriteExecute) {
				return sdk.WrapError(sdk.ErrForbidden, "addNotificationsHandler > Cannot access to this environment")
			}

			// Insert or update notification
			if err := notification.InsertOrUpdateUserNotificationSettings(tx, app.ID, n.Pipeline.ID, n.Environment.ID, &n); err != nil {
				return sdk.WrapError(err, "addNotificationsHandler> cannot update user notification")

			}
		}

		if err := application.UpdateLastModified(tx, api.Cache, app, getUser(ctx)); err != nil {
			return sdk.WrapError(err, "addNotificationsHandler> cannot update application last_modified date")

		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "addNotificationsHandler: Cannot commit transaction")
		}

		var errNotif error
		app.Notifications, errNotif = notification.LoadAllUserNotificationSettings(api.mustDB(), app.ID)
		if errNotif != nil {
			return sdk.WrapError(errNotif, "addNotificationsHandler> cannot load notifications")
		}

		return WriteJSON(w, app, http.StatusOK)
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
			return sdk.WrapError(err, "updateUserNotificationApplicationPipelineHandler> Cannot load application %s for project %s from db", appName, key)
		}

		//Load pipeline
		pipeline, err := pipeline.LoadPipeline(api.mustDB(), key, pipelineName, false)
		if err != nil {
			return sdk.WrapError(err, "updateUserNotificationApplicationPipelineHandler> Cannot load pipeline %s", pipelineName)

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

		if !permission.AccessToEnvironment(key, notifs.Environment.Name, getUser(ctx), permission.PermissionReadWriteExecute) {
			return sdk.WrapError(sdk.ErrForbidden, "updateUserNotificationApplicationPipelineHandler> Cannot access to this environment")
		}

		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WrapError(err, "updateUserNotificationApplicationPipelineHandler> cannot start transaction")

		}
		defer tx.Rollback()

		// Insert or update notification
		if err := notification.InsertOrUpdateUserNotificationSettings(tx, applicationData.ID, pipeline.ID, notifs.Environment.ID, notifs); err != nil {
			return sdk.WrapError(err, "updateUserNotificationApplicationPipelineHandler> cannot update user notification")

		}

		err = application.UpdateLastModified(tx, api.Cache, applicationData, getUser(ctx))
		if err != nil {
			return sdk.WrapError(err, "updateUserNotificationApplicationPipelineHandler> cannot update application last_modified date")

		}

		err = tx.Commit()
		if err != nil {
			return sdk.WrapError(err, "updateUserNotificationApplicationPipelineHandler> cannot commit transaction")

		}

		var errNotif error
		applicationData.Notifications, errNotif = notification.LoadAllUserNotificationSettings(api.mustDB(), applicationData.ID)
		if errNotif != nil {
			return sdk.WrapError(errNotif, "updateUserNotificationApplicationPipelineHandler> Cannot load notifications")
		}

		return WriteJSON(w, applicationData, http.StatusOK)
	}
}
