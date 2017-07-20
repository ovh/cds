package main

import (
	"net/http"
	"strconv"
	"time"

	"github.com/go-gorp/gorp"
	"github.com/gorhill/cronexpr"
	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/businesscontext"
	"github.com/ovh/cds/engine/api/environment"
	"github.com/ovh/cds/engine/api/permission"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/scheduler"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func getSchedulerApplicationPipelineHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *businesscontext.Ctx) error {
	vars := mux.Vars(r)
	pipelineName := vars["permPipelineKey"]

	//Load pipeline
	pip, errP := pipeline.LoadPipeline(db, c.Project.Key, pipelineName, false)
	if errP != nil {
		log.Warning("getSchedulerApplicationPipelineHandler> Cannot load pipeline %s: %s", pipelineName, errP)
		return errP

	}

	//Load environment
	if err := r.ParseForm(); err != nil {
		log.Warning("getSchedulerApplicationPipelineHandler> Cannot parse form: %s", err)
		return sdk.ErrUnknownError

	}
	envName := r.Form.Get("envName")
	var env *sdk.Environment
	if envName != "" {
		var err error
		env, err = environment.LoadEnvironmentByName(db, c.Project.Key, envName)
		if err != nil {
			return err

		}
	}

	//Load schedulers
	var schedulers []sdk.PipelineScheduler
	if env == nil {
		var err error
		schedulers, err = scheduler.GetByApplicationPipeline(db, c.Application, pip)
		if err != nil {
			log.Warning("getSchedulerApplicationPipelineHandler> cmdApplicationPipelineSchedulerAddEnvCannot load pipeline schedulers: %s", err)
			return err

		}
	} else {
		var err error
		schedulers, err = scheduler.GetByApplicationPipelineEnv(db, c.Application, pip, env)
		if err != nil {
			log.Warning("getSchedulerApplicationPipelineHandler> Cannot load pipeline schedulers: %s", err)
			return err

		}
	}

	return WriteJSON(w, r, schedulers, http.StatusOK)
}

func addSchedulerApplicationPipelineHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *businesscontext.Ctx) error {
	vars := mux.Vars(r)
	pipelineName := vars["permPipelineKey"]

	//Load pipeline
	pip, errP := pipeline.LoadPipeline(db, c.Project.Key, pipelineName, false)
	if errP != nil {
		log.Warning("addSchedulerApplicationPipelineHandler> Cannot load pipeline %s: %s", pipelineName, errP)
		return errP

	}

	//Load environment
	if err := r.ParseForm(); err != nil {
		log.Warning("getSchedulerApplicationPipelineHandler> Cannot parse form: %s", err)
		return sdk.ErrUnknownError

	}
	envName := r.Form.Get("envName")
	var env *sdk.Environment
	if envName != "" {
		var err error
		env, err = environment.LoadEnvironmentByName(db, c.Project.Key, envName)
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
		schedulers, err = scheduler.GetByApplicationPipeline(db, c.Application, pip)
		if err != nil {
			log.Warning("getSchedulerApplicationPipelineHandler> Cannot load pipeline schedulers: %s", err)
			return err

		}
	} else {
		var err error
		schedulers, err = scheduler.GetByApplicationPipelineEnv(db, c.Application, pip, env)
		if err != nil {
			log.Warning("getSchedulerApplicationPipelineHandler> Cannot load pipeline schedulers: %s", err)
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
			log.Warning("addSchedulerApplicationPipelineHandler> invalid timezone %s  %s", s.Timezone, err)
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
	s.ApplicationID = c.Application.ID
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

	if err := application.UpdateLastModified(tx, c.Application, c.User); err != nil {
		return sdk.WrapError(err, "addSchedulerApplicationPipelineHandler> cannot update application last modified date")
	}

	if err := tx.Commit(); err != nil {
		return sdk.WrapError(err, "addSchedulerApplicationPipelineHandler> cannot commit transaction")
	}

	var errW error
	c.Application.Workflows, errW = workflow.LoadCDTree(db, c.Project.Key, c.Application.Name, c.User, "", 0)
	if errW != nil {
		return sdk.WrapError(errW, "addSchedulerApplicationPipelineHandler> cannot reload workflow")
	}

	return WriteJSON(w, r, c.Application, http.StatusCreated)
}

func updateSchedulerApplicationPipelineHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *businesscontext.Ctx) error {
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
		env, err = environment.LoadEnvironmentByName(db, c.Project.Key, envName)
		if err != nil {
			log.Warning("updateSchedulerApplicationPipelineHandler> %s", err)
			return err
		}

		if !permission.AccessToEnvironment(env.ID, c.User, permission.PermissionReadExecute) {
			log.Warning("updateSchedulerApplicationPipelineHandler> Cannot access to this environment")
			return sdk.ErrForbidden

		}
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

	if err := application.UpdateLastModified(tx, c.Application, c.User); err != nil {
		return sdk.WrapError(err, "updateSchedulerApplicationPipelineHandler> Cannot update application last modified date")
	}

	if err := tx.Commit(); err != nil {
		return sdk.WrapError(err, "updateSchedulerApplicationPipelineHandler> Cannot commit transaction")
	}

	var errW error
	c.Application.Workflows, errW = workflow.LoadCDTree(db, c.Project.Key, c.Application.Name, c.User, "", 0)
	if errW != nil {
		return sdk.WrapError(errW, "updateSchedulerApplicationPipelineHandler> Cannot load workflow")
	}

	return WriteJSON(w, r, c.Application, http.StatusOK)
}

func deleteSchedulerApplicationPipelineHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *businesscontext.Ctx) error {
	vars := mux.Vars(r)

	idString := vars["id"]

	id, errInt := strconv.ParseInt(idString, 10, 64)
	if errInt != nil {
		return sdk.ErrInvalidID
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

	if err := application.UpdateLastModified(tx, c.Application, c.User); err != nil {
		return sdk.WrapError(err, "deleteSchedulerApplicationPipelineHandler> Cannot update application last modified date")
	}

	if err := tx.Commit(); err != nil {
		return sdk.WrapError(err, "deleteSchedulerApplicationPipelineHandler> Cannot commit transaction")
	}

	var errW error
	c.Application.Workflows, errW = workflow.LoadCDTree(db, c.Project.Key, c.Application.Name, c.User, "", 0)
	if errW != nil {
		return sdk.WrapError(errW, "deleteSchedulerApplicationPipelineHandler> Cannot load workflow")
	}

	return WriteJSON(w, r, c.Application, http.StatusOK)
}
