package api

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/gorhill/cronexpr"
	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/environment"
	"github.com/ovh/cds/engine/api/permission"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/scheduler"
	"github.com/ovh/cds/engine/api/workflowv0"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func (api *API) getSchedulerApplicationPipelineHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars["key"]
		appName := vars["permApplicationName"]
		pipelineName := vars["permPipelineKey"]

		///Load application
		app, errA := application.LoadByName(api.mustDB(), api.Cache, key, appName, getUser(ctx))
		if errA != nil {
			return sdk.WrapError(errA, "getSchedulerApplicationPipelineHandler> Cannot load application %s for project %s from db", appName, key)

		}

		//Load pipeline
		pip, errP := pipeline.LoadPipeline(api.mustDB(), key, pipelineName, false)
		if errP != nil {
			return sdk.WrapError(errP, "getSchedulerApplicationPipelineHandler> Cannot load pipeline %s", pipelineName)

		}

		//Load environment
		if err := r.ParseForm(); err != nil {
			return sdk.WrapError(sdk.ErrWrongRequest, "getSchedulerApplicationPipelineHandler> Cannot parse form")

		}
		envName := r.Form.Get("envName")
		var env *sdk.Environment
		if envName != "" {
			var err error
			env, err = environment.LoadEnvironmentByName(api.mustDB(), key, envName)
			if err != nil {
				return err

			}
		}

		//Load schedulers
		var schedulers []sdk.PipelineScheduler
		if env == nil {
			var err error
			schedulers, err = scheduler.GetByApplicationPipeline(api.mustDB(), app, pip)
			if err != nil {
				return sdk.WrapError(err, "getSchedulerApplicationPipelineHandler> cmdApplicationPipelineSchedulerAddEnvCannot load pipeline schedulers")

			}
		} else {
			var err error
			schedulers, err = scheduler.GetByApplicationPipelineEnv(api.mustDB(), app, pip, env)
			if err != nil {
				return sdk.WrapError(err, "getSchedulerApplicationPipelineHandler> Cannot load pipeline schedulers")

			}
		}

		return WriteJSON(w, schedulers, http.StatusOK)
	}
}

func (api *API) addSchedulerApplicationPipelineHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars["key"]
		appName := vars["permApplicationName"]
		pipelineName := vars["permPipelineKey"]

		///Load application
		app, errA := application.LoadByName(api.mustDB(), api.Cache, key, appName, getUser(ctx))
		if errA != nil {
			return sdk.WrapError(errA, "addSchedulerApplicationPipelineHandler> Cannot load application %s for project %s from db", appName, key)

		}

		//Load pipeline
		pip, errP := pipeline.LoadPipeline(api.mustDB(), key, pipelineName, false)
		if errP != nil {
			return sdk.WrapError(errP, "addSchedulerApplicationPipelineHandler> Cannot load pipeline %s", pipelineName)

		}

		//Load environment
		if err := r.ParseForm(); err != nil {
			return sdk.WrapError(sdk.ErrWrongRequest, "getSchedulerApplicationPipelineHandler> Cannot parse form")

		}
		envName := r.Form.Get("envName")
		var env *sdk.Environment
		if envName != "" {
			var err error
			env, err = environment.LoadEnvironmentByName(api.mustDB(), key, envName)
			if err != nil {
				return err

			}
		}

		if env != nil {
			if !permission.AccessToEnvironment(key, env.Name, getUser(ctx), permission.PermissionReadExecute) {
				return sdk.WrapError(sdk.ErrForbidden, "getSchedulerApplicationPipelineHandler> Cannot access to this environment")
			}
		}

		//Load schedulers
		var schedulers []sdk.PipelineScheduler
		if env == nil {
			var err error
			env = &sdk.DefaultEnv
			schedulers, err = scheduler.GetByApplicationPipeline(api.mustDB(), app, pip)
			if err != nil {
				return sdk.WrapError(err, "getSchedulerApplicationPipelineHandler> Cannot load pipeline schedulers")

			}
		} else {
			var err error
			schedulers, err = scheduler.GetByApplicationPipelineEnv(api.mustDB(), app, pip, env)
			if err != nil {
				return sdk.WrapError(err, "getSchedulerApplicationPipelineHandler> Cannot load pipeline schedulers")

			}
		}

		if pip.Type != sdk.BuildPipeline && env.ID == sdk.DefaultEnv.ID {
			return sdk.WrapError(sdk.ErrWrongRequest, "Cannot create a scheduler on this pipeline without an environment")
		}

		// Unmarshal args
		s := &sdk.PipelineScheduler{}
		if err := UnmarshalBody(r, s); err != nil {
			return err
		}

		//Check timezone
		if s.Timezone != "" {
			if _, err := time.LoadLocation(s.Timezone); err != nil {
				return sdk.WrapError(sdk.ErrInvalidTimezone, "addSchedulerApplicationPipelineHandler> invalid timezone %s  %s", s.Timezone, err)
			}
		}

		//Parsing cronexpr
		if _, err := cronexpr.Parse(s.Crontab); err != nil {
			return sdk.NewError(sdk.ErrWrongRequest, err)
		}

		// schedulers with same parameters are forbidden
	check:
		for _, os := range schedulers {
			if os.Crontab != s.Crontab {
				continue
			}
			for _, a := range os.Args {
				var same = false
				for _, aa := range s.Args {
					if aa.Name == a.Name && aa.Value == a.Value {
						same = true
						break
					}
				}
				if !same {
					break check
				}
			}
			return sdk.ErrConflict

		}

		//Insert scheduler
		s.ApplicationID = app.ID
		s.PipelineID = pip.ID
		s.EnvironmentID = env.ID

		tx, errBegin := api.mustDB().Begin()
		if errBegin != nil {
			return sdk.WrapError(errBegin, "addSchedulerApplicationPipelineHandler> Cannot open transaction")
		}
		defer tx.Rollback()

		if err := scheduler.Insert(tx, s); err != nil {
			return sdk.WrapError(err, "addSchedulerApplicationPipelineHandler> cannot insert scheduler")

		}

		if err := application.UpdateLastModified(tx, api.Cache, app, getUser(ctx)); err != nil {
			return sdk.WrapError(err, "addSchedulerApplicationPipelineHandler> cannot update application last modified date")
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "addSchedulerApplicationPipelineHandler> cannot commit transaction")
		}

		var errW error
		app.Workflows, errW = workflowv0.LoadCDTree(api.mustDB(), api.Cache, key, appName, getUser(ctx), "", "", 0)
		if errW != nil {
			return sdk.WrapError(errW, "addSchedulerApplicationPipelineHandler> cannot reload workflow")
		}

		return WriteJSON(w, app, http.StatusCreated)
	}
}

func (api *API) updateSchedulerApplicationPipelineHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars["key"]
		appName := vars["permApplicationName"]

		// Unmarshal args
		s := &sdk.PipelineScheduler{}
		if err := UnmarshalBody(r, s); err != nil {
			return err
		}

		//Parsing cronexpr
		if _, err := cronexpr.Parse(s.Crontab); err != nil {
			log.Warning("updateSchedulerApplicationPipelineHandler> %s", err)
			return sdk.NewError(sdk.ErrWrongRequest, err)
		}

		//Load the environment
		envName := s.EnvironmentName
		var env *sdk.Environment
		if envName != "" && envName != sdk.DefaultEnv.Name {
			var err error
			env, err = environment.LoadEnvironmentByName(api.mustDB(), key, envName)
			if err != nil {
				return sdk.WrapError(err, "updateSchedulerApplicationPipelineHandler> ")
			}

			if !permission.AccessToEnvironment(key, env.Name, getUser(ctx), permission.PermissionReadExecute) {
				return sdk.WrapError(sdk.ErrForbidden, "updateSchedulerApplicationPipelineHandler> Cannot access to this environment")
			}
		}

		// Load application
		app, errA := application.LoadByName(api.mustDB(), api.Cache, key, appName, getUser(ctx))
		if errA != nil {
			return sdk.WrapError(errA, "updateSchedulerApplicationPipelineHandler> Cannot load application %s", appName)
		}

		//Load the scheduler
		sOld, err := scheduler.Load(api.mustDB(), s.ID)
		if err != nil {
			return sdk.WrapError(err, "updateSchedulerApplicationPipelineHandler> ")

		}

		//Update it
		sOld.Crontab = s.Crontab
		sOld.Disabled = s.Disabled
		sOld.Args = s.Args

		if env != nil {
			sOld.EnvironmentID = env.ID
		}

		tx, errBegin := api.mustDB().Begin()
		if errBegin != nil {
			return sdk.WrapError(errBegin, "updateSchedulerApplicationPipelineHandler> Cannot start transaction")
		}
		defer tx.Rollback()

		if err := scheduler.Update(tx, sOld); err != nil {
			return sdk.WrapError(err, "updateSchedulerApplicationPipelineHandler> Cannot update scheduler")
		}

		nx, errN := scheduler.LoadNextExecution(tx, sOld.ID, sOld.Timezone)
		if errN != nil {
			return sdk.WrapError(errN, "updateSchedulerApplicationPipelineHandler> Cannot load next execution")
		}

		if nx != nil {
			if err := scheduler.DeleteExecution(tx, nx); err != nil {
				return sdk.WrapError(err, "updateSchedulerApplicationPipelineHandler> Cannot delete next execution")
			}
		}

		if err := application.UpdateLastModified(tx, api.Cache, app, getUser(ctx)); err != nil {
			return sdk.WrapError(err, "updateSchedulerApplicationPipelineHandler> Cannot update application last modified date")
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "updateSchedulerApplicationPipelineHandler> Cannot commit transaction")
		}

		var errW error
		app.Workflows, errW = workflowv0.LoadCDTree(api.mustDB(), api.Cache, key, appName, getUser(ctx), "", "", 0)
		if errW != nil {
			return sdk.WrapError(errW, "updateSchedulerApplicationPipelineHandler> Cannot load workflow")
		}

		return WriteJSON(w, app, http.StatusOK)
	}
}

func (api *API) deleteSchedulerApplicationPipelineHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars["key"]
		appName := vars["permApplicationName"]

		idString := vars["id"]

		id, errInt := strconv.ParseInt(idString, 10, 64)
		if errInt != nil {
			return sdk.ErrInvalidID
		}

		app, errA := application.LoadByName(api.mustDB(), api.Cache, key, appName, getUser(ctx))
		if errA != nil {
			return sdk.WrapError(errA, "deleteSchedulerApplicationPipelineHandler> Cannot load application %s", appName)
		}

		//Load the scheduler
		sOld, err := scheduler.Load(api.mustDB(), id)
		if err != nil {
			return err
		}

		tx, errB := api.mustDB().Begin()
		if errB != nil {
			return sdk.WrapError(errB, "deleteSchedulerApplicationPipelineHandler> Cannot open transaction")
		}
		defer tx.Rollback()

		//Delete all the things
		if err := scheduler.Delete(tx, sOld); err != nil {
			return err
		}

		if err := application.UpdateLastModified(tx, api.Cache, app, getUser(ctx)); err != nil {
			return sdk.WrapError(err, "deleteSchedulerApplicationPipelineHandler> Cannot update application last modified date")
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "deleteSchedulerApplicationPipelineHandler> Cannot commit transaction")
		}

		var errW error
		app.Workflows, errW = workflowv0.LoadCDTree(api.mustDB(), api.Cache, key, appName, getUser(ctx), "", "", 0)
		if errW != nil {
			return sdk.WrapError(errW, "deleteSchedulerApplicationPipelineHandler> Cannot load workflow")
		}

		return WriteJSON(w, app, http.StatusOK)
	}
}
