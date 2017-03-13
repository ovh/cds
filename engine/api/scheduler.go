package main

import (
	"net/http"
	"strconv"
	"time"

	"github.com/go-gorp/gorp"
	"github.com/gorhill/cronexpr"
	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/context"
	"github.com/ovh/cds/engine/api/environment"
	"github.com/ovh/cds/engine/api/permission"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/scheduler"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/engine/log"
	"github.com/ovh/cds/sdk"
)

func getSchedulerApplicationPipelineHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	vars := mux.Vars(r)
	key := vars["key"]
	appName := vars["permApplicationName"]
	pipelineName := vars["permPipelineKey"]

	///Load application
	app, errA := application.LoadByName(db, key, appName, c.User)
	if errA != nil {
		log.Warning("getSchedulerApplicationPipelineHandler> Cannot load application %s for project %s from db: %s\n", appName, key, errA)
		return errA

	}

	//Load pipeline
	pip, errP := pipeline.LoadPipeline(db, key, pipelineName, false)
	if errP != nil {
		log.Warning("getSchedulerApplicationPipelineHandler> Cannot load pipeline %s: %s\n", pipelineName, errP)
		return errP

	}

	//Load environment
	if err := r.ParseForm(); err != nil {
		log.Warning("getSchedulerApplicationPipelineHandler> Cannot parse form: %s\n", err)
		return sdk.ErrUnknownError

	}
	envName := r.Form.Get("envName")
	var env *sdk.Environment
	if envName != "" {
		var err error
		env, err = environment.LoadEnvironmentByName(db, key, envName)
		if err != nil {
			return err

		}
	}

	//Load schedulers
	var schedulers []sdk.PipelineScheduler
	if env == nil {
		var err error
		schedulers, err = scheduler.GetByApplicationPipeline(db, app, pip)
		if err != nil {
			log.Warning("getSchedulerApplicationPipelineHandler> cmdApplicationPipelineSchedulerAddEnvCannot load pipeline schedulers: %s\n", err)
			return err

		}
	} else {
		var err error
		schedulers, err = scheduler.GetByApplicationPipelineEnv(db, app, pip, env)
		if err != nil {
			log.Warning("getSchedulerApplicationPipelineHandler> Cannot load pipeline schedulers: %s\n", err)
			return err

		}
	}

	return WriteJSON(w, r, schedulers, http.StatusOK)
}

func addSchedulerApplicationPipelineHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	vars := mux.Vars(r)
	key := vars["key"]
	appName := vars["permApplicationName"]
	pipelineName := vars["permPipelineKey"]

	///Load application
	app, errA := application.LoadByName(db, key, appName, c.User)
	if errA != nil {
		log.Warning("addSchedulerApplicationPipelineHandler> Cannot load application %s for project %s from db: %s\n", appName, key, errA)
		return errA

	}

	//Load pipeline
	pip, errP := pipeline.LoadPipeline(db, key, pipelineName, false)
	if errP != nil {
		log.Warning("addSchedulerApplicationPipelineHandler> Cannot load pipeline %s: %s\n", pipelineName, errP)
		return errP

	}

	//Load environment
	if err := r.ParseForm(); err != nil {
		log.Warning("getSchedulerApplicationPipelineHandler> Cannot parse form: %s\n", err)
		return sdk.ErrUnknownError

	}
	envName := r.Form.Get("envName")
	var env *sdk.Environment
	if envName != "" {
		var err error
		env, err = environment.LoadEnvironmentByName(db, key, envName)
		if err != nil {
			return err

		}
	}

	if env != nil {
		if !permission.AccessToEnvironment(env.ID, c.User, permission.PermissionReadExecute) {
			log.Warning("getSchedulerApplicationPipelineHandler> Cannot access to this environment")
			return sdk.ErrForbidden
		}
	}

	//Load schedulers
	var schedulers []sdk.PipelineScheduler
	if env == nil {
		var err error
		env = &sdk.DefaultEnv
		schedulers, err = scheduler.GetByApplicationPipeline(db, app, pip)
		if err != nil {
			log.Warning("getSchedulerApplicationPipelineHandler> Cannot load pipeline schedulers: %s\n", err)
			return err

		}
	} else {
		var err error
		schedulers, err = scheduler.GetByApplicationPipelineEnv(db, app, pip, env)
		if err != nil {
			log.Warning("getSchedulerApplicationPipelineHandler> Cannot load pipeline schedulers: %s\n", err)
			return err

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
			log.Warning("addSchedulerApplicationPipelineHandler> invalid timezone %s  %s\n", s.Timezone, err)
			return sdk.ErrInvalidTimezone
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

	tx, errBegin := db.Begin()
	if errBegin != nil {
		return sdk.WrapError(errBegin, "addSchedulerApplicationPipelineHandler> Cannot open transaction")
	}
	defer tx.Rollback()

	if err := scheduler.Insert(tx, s); err != nil {
		return sdk.WrapError(err, "addSchedulerApplicationPipelineHandler> cannot insert scheduler")

	}

	if err := application.UpdateLastModified(tx, app, c.User); err != nil {
		return sdk.WrapError(err, "addSchedulerApplicationPipelineHandler> cannot update application last modified date")
	}

	if err := tx.Commit(); err != nil {
		return sdk.WrapError(err, "addSchedulerApplicationPipelineHandler> cannot commit transaction")
	}

	var errW error
	app.Workflows, errW = workflow.LoadCDTree(db, key, appName, c.User)
	if errW != nil {
		return sdk.WrapError(errW, "addSchedulerApplicationPipelineHandler> cannot reload workflow")
	}

	return WriteJSON(w, r, app, http.StatusCreated)
}

func updateSchedulerApplicationPipelineHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
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
		env, err = environment.LoadEnvironmentByName(db, key, envName)
		if err != nil {
			log.Warning("updateSchedulerApplicationPipelineHandler> %s", err)
			return err
		}

		if !permission.AccessToEnvironment(env.ID, c.User, permission.PermissionReadExecute) {
			log.Warning("updateSchedulerApplicationPipelineHandler> Cannot access to this environment")
			return sdk.ErrForbidden

		}
	}

	// Load application
	app, errA := application.LoadByName(db, key, appName, c.User)
	if errA != nil {
		return sdk.WrapError(errA, "updateSchedulerApplicationPipelineHandler> Cannot load application %s", appName)
	}

	//Load the scheduler
	sOld, err := scheduler.Load(db, s.ID)
	if err != nil {
		log.Warning("updateSchedulerApplicationPipelineHandler> %s", err)
		return err

	}

	//Update it
	sOld.Crontab = s.Crontab
	sOld.Disabled = s.Disabled
	sOld.Args = s.Args

	if env != nil {
		sOld.EnvironmentID = env.ID
	}

	tx, errBegin := db.Begin()
	if errBegin != nil {
		return sdk.WrapError(errBegin, "updateSchedulerApplicationPipelineHandler> Cannot start transaction")
	}
	defer tx.Rollback()

	if err := scheduler.Update(tx, sOld); err != nil {
		return sdk.WrapError(err, "updateSchedulerApplicationPipelineHandler> Cannot update scheduler")
	}

	if err := scheduler.LockPipelineExecutions(tx); err != nil {
		return sdk.WrapError(err, "updateSchedulerApplicationPipelineHandler> Cannot lock pipeline execution")
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

	if err := application.UpdateLastModified(tx, app, c.User); err != nil {
		return sdk.WrapError(err, "updateSchedulerApplicationPipelineHandler> Cannot update application last modified date")
	}

	if err := tx.Commit(); err != nil {
		return sdk.WrapError(err, "updateSchedulerApplicationPipelineHandler> Cannot commit transaction")
	}

	var errW error
	app.Workflows, errW = workflow.LoadCDTree(db, key, appName, c.User)
	if errW != nil {
		return sdk.WrapError(errW, "updateSchedulerApplicationPipelineHandler> Cannot load workflow")
	}

	return WriteJSON(w, r, app, http.StatusOK)
}

func deleteSchedulerApplicationPipelineHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	vars := mux.Vars(r)
	key := vars["key"]
	appName := vars["permApplicationName"]

	idString := vars["id"]

	id, errInt := strconv.ParseInt(idString, 10, 64)
	if errInt != nil {
		return sdk.ErrInvalidID
	}

	app, errA := application.LoadByName(db, key, appName, c.User)
	if errA != nil {
		return sdk.WrapError(errA, "deleteSchedulerApplicationPipelineHandler> Cannot load application %s", appName)
	}

	//Load the scheduler
	sOld, err := scheduler.Load(db, id)
	if err != nil {
		return err
	}

	tx, errB := db.Begin()
	if errB != nil {
		return sdk.WrapError(errB, "deleteSchedulerApplicationPipelineHandler> Cannot open transaction")
	}
	defer tx.Rollback()

	//Delete all the things
	if err := scheduler.Delete(tx, sOld); err != nil {
		return err
	}

	if err := application.UpdateLastModified(tx, app, c.User); err != nil {
		return sdk.WrapError(err, "deleteSchedulerApplicationPipelineHandler> Cannot update application last modified date")
	}

	if err := tx.Commit(); err != nil {
		return sdk.WrapError(err, "deleteSchedulerApplicationPipelineHandler> Cannot commit transaction")
	}

	var errW error
	app.Workflows, errW = workflow.LoadCDTree(db, key, appName, c.User)
	if errW != nil {
		return sdk.WrapError(errW, "deleteSchedulerApplicationPipelineHandler> Cannot load workflow")
	}

	return WriteJSON(w, r, app, http.StatusOK)
}
