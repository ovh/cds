package main

import (
	"database/sql"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/context"
	"github.com/ovh/cds/engine/api/database"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/scheduler"
	"github.com/ovh/cds/engine/log"
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

	//Load schedulers
	schedulers, err := scheduler.GetByApplicationPipeline(database.DBMap(db), app, pip)
	if err != nil {
		log.Warning("getSchedulerApplicationPipelineHandler> Cannot load pipeline schedulers: %s\n", err)
		WriteError(w, r, err)
		return
	}

	WriteJSON(w, r, schedulers, http.StatusOK)
}

func addSchedulerApplicationPipelineHandler(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {

}

func updateSchedulerApplicationPipelineHandler(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {

}

func deleteSchedulerApplicationPipelineHandler(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {

}
