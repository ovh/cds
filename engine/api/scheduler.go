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

	if err := scheduler.Insert(db, s); err != nil {
		log.Warning("addSchedulerApplicationPipelineHandler> cannot insert scheduler : %s", err)
		return err

	}

	return WriteJSON(w, r, s, http.StatusCreated)
}

func updateSchedulerApplicationPipelineHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	vars := mux.Vars(r)
	key := vars["key"]

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

	if err := scheduler.Update(db, sOld); err != nil {
		log.Warning("updateSchedulerApplicationPipelineHandler> %s", err)
		return err
	}

	return WriteJSON(w, r, s, http.StatusOK)
}

func deleteSchedulerApplicationPipelineHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
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

	//Delete all the things
	if err := scheduler.Delete(db, sOld); err != nil {
		return err
	}

	return nil
}
