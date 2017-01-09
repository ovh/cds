package main

import (
	"database/sql"
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/gorhill/cronexpr"
	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/context"
	"github.com/ovh/cds/engine/api/database"
	"github.com/ovh/cds/engine/api/environment"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/scheduler"
	"github.com/ovh/cds/engine/log"
	"github.com/ovh/cds/sdk"
)

func getSchedulerApplicationPipelineHandler(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {
	vars := mux.Vars(r)
	key := vars["key"]
	appName := vars["permApplicationName"]
	pipelineName := vars["permPipelineKey"]

	///Load application
	app, err := application.LoadApplicationByName(db, key, appName)
	if err != nil {
		log.Warning("getSchedulerApplicationPipelineHandler> Cannot load application %s for project %s from db: %s\n", appName, key, err)
		WriteError(w, r, err)
		return
	}

	//Load pipeline
	pip, err := pipeline.LoadPipeline(db, key, pipelineName, false)
	if err != nil {
		log.Warning("getSchedulerApplicationPipelineHandler> Cannot load pipeline %s: %s\n", pipelineName, err)
		WriteError(w, r, err)
		return
	}

	//Load environment
	if err := r.ParseForm(); err != nil {
		log.Warning("getSchedulerApplicationPipelineHandler> Cannot parse form: %s\n", err)
		WriteError(w, r, sdk.ErrUnknownError)
		return
	}
	envName := r.Form.Get("envName")
	var env *sdk.Environment
	if envName != "" {
		env, err = environment.LoadEnvironmentByName(db, key, envName)
		if err != nil {
			WriteError(w, r, err)
			return
		}
	}

	//Load schedulers
	var schedulers []sdk.PipelineScheduler
	if env == nil {
		schedulers, err = scheduler.GetByApplicationPipeline(database.DBMap(db), app, pip)
		if err != nil {
			log.Warning("getSchedulerApplicationPipelineHandler> Cannot load pipeline schedulers: %s\n", err)
			WriteError(w, r, err)
			return
		}
	} else {
		schedulers, err = scheduler.GetByApplicationPipelineEnv(database.DBMap(db), app, pip, env)
		if err != nil {
			log.Warning("getSchedulerApplicationPipelineHandler> Cannot load pipeline schedulers: %s\n", err)
			WriteError(w, r, err)
			return
		}
	}

	WriteJSON(w, r, schedulers, http.StatusOK)
}

func addSchedulerApplicationPipelineHandler(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {
	vars := mux.Vars(r)
	key := vars["key"]
	appName := vars["permApplicationName"]
	pipelineName := vars["permPipelineKey"]

	///Load application
	app, err := application.LoadApplicationByName(db, key, appName)
	if err != nil {
		log.Warning("addSchedulerApplicationPipelineHandler> Cannot load application %s for project %s from db: %s\n", appName, key, err)
		WriteError(w, r, err)
		return
	}

	//Load pipeline
	pip, err := pipeline.LoadPipeline(db, key, pipelineName, false)
	if err != nil {
		log.Warning("addSchedulerApplicationPipelineHandler> Cannot load pipeline %s: %s\n", pipelineName, err)
		WriteError(w, r, err)
		return
	}

	//Load environment
	if err := r.ParseForm(); err != nil {
		log.Warning("getSchedulerApplicationPipelineHandler> Cannot parse form: %s\n", err)
		WriteError(w, r, sdk.ErrUnknownError)
		return
	}
	envName := r.Form.Get("envName")
	var env *sdk.Environment
	if envName != "" {
		env, err = environment.LoadEnvironmentByName(db, key, envName)
		if err != nil {
			WriteError(w, r, err)
			return
		}
	}

	//Load schedulers
	var schedulers []sdk.PipelineScheduler
	if env == nil {
		env = &sdk.DefaultEnv
		schedulers, err = scheduler.GetByApplicationPipeline(database.DBMap(db), app, pip)
		if err != nil {
			log.Warning("getSchedulerApplicationPipelineHandler> Cannot load pipeline schedulers: %s\n", err)
			WriteError(w, r, err)
			return
		}
	} else {
		schedulers, err = scheduler.GetByApplicationPipelineEnv(database.DBMap(db), app, pip, env)
		if err != nil {
			log.Warning("getSchedulerApplicationPipelineHandler> Cannot load pipeline schedulers: %s\n", err)
			WriteError(w, r, err)
			return
		}
	}

	// Get args in body
	data, errRead := ioutil.ReadAll(r.Body)
	if errRead != nil {
		log.Warning("addSchedulerApplicationPipelineHandler> cannot read body: %s\n", errRead)
		WriteError(w, r, sdk.ErrWrongRequest)
		return
	}

	// Unmarshal args
	s := &sdk.PipelineScheduler{}
	if err := json.Unmarshal(data, s); err != nil {
		log.Warning("addSchedulerApplicationPipelineHandler> cannot unmarshal body:  %s\n", err)
		WriteError(w, r, sdk.ErrWrongRequest)
		return
	}

	//Parsing cronexpr
	if _, err := cronexpr.Parse(s.Crontab); err != nil {
		WriteError(w, r, sdk.NewError(sdk.ErrWrongRequest, err))
		return
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
		WriteError(w, r, sdk.ErrConflict)
		return
	}

	//Insert scheduler
	s.ApplicationID = app.ID
	s.PipelineID = pip.ID
	s.EnvironmentID = env.ID

	if err := scheduler.Insert(database.DBMap(db), s); err != nil {
		log.Warning("addSchedulerApplicationPipelineHandler> cannot insert scheduler : %s", err)
		WriteError(w, r, err)
		return
	}

	WriteJSON(w, r, s, http.StatusCreated)
}

func updateSchedulerApplicationPipelineHandler(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {
	// Get args in body
	data, errRead := ioutil.ReadAll(r.Body)
	if errRead != nil {
		log.Warning("addSchedulerApplicationPipelineHandler> cannot read body: %s\n", errRead)
		WriteError(w, r, sdk.ErrWrongRequest)
		return
	}

	// Unmarshal args
	s := &sdk.PipelineScheduler{}
	if err := json.Unmarshal(data, s); err != nil {
		log.Warning("addSchedulerApplicationPipelineHandler> cannot unmarshal body:  %s\n", err)
		WriteError(w, r, sdk.ErrWrongRequest)
		return
	}

	//Parsing cronexpr
	if _, err := cronexpr.Parse(s.Crontab); err != nil {
		WriteError(w, r, sdk.NewError(sdk.ErrWrongRequest, err))
		return
	}

	//Load the scheduler
	sOld, err := scheduler.Load(database.DBMap(db), s.ID)
	if err != nil {
		WriteError(w, r, err)
		return
	}

	//Update it
	sOld.Crontab = s.Crontab
	sOld.Disabled = s.Disabled
	sOld.Args = s.Args
	if err := scheduler.Update(database.DBMap(db), sOld); err != nil {
		WriteError(w, r, err)
		return
	}

	WriteJSON(w, r, s, http.StatusOK)
}

func deleteSchedulerApplicationPipelineHandler(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {
	// Get args in body
	data, errRead := ioutil.ReadAll(r.Body)
	if errRead != nil {
		log.Warning("addSchedulerApplicationPipelineHandler> cannot read body: %s\n", errRead)
		WriteError(w, r, sdk.ErrWrongRequest)
		return
	}

	// Unmarshal args
	s := &sdk.PipelineScheduler{}
	if err := json.Unmarshal(data, s); err != nil {
		log.Warning("addSchedulerApplicationPipelineHandler> cannot unmarshal body:  %s\n", err)
		WriteError(w, r, sdk.ErrWrongRequest)
		return
	}

	//Load the scheduler
	sOld, err := scheduler.Load(database.DBMap(db), s.ID)
	if err != nil {
		WriteError(w, r, err)
		return
	}

	//Delete all the things
	if err := scheduler.Delete(database.DBMap(db), sOld); err != nil {
		WriteError(w, r, err)
		return
	}
}
