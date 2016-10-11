package main

import (
	"database/sql"
	"io/ioutil"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/context"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/log"
	"github.com/ovh/cds/sdk"
)

func addStageHandler(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {

	// Get project name in URL
	vars := mux.Vars(r)
	projectKey := vars["key"]
	pipelineKey := vars["permPipelineKey"]

	// Get body
	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Warning("addStageHandler> cannot read body: %s", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	stageData, err := sdk.NewStage("").FromJSON(data)
	if err != nil {
		log.Warning("addStageHandler> cannot unmarshal body: %s", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Check if pipeline exist
	pipelineData, err := pipeline.LoadPipeline(db, projectKey, pipelineKey, true)
	if err != nil {
		WriteError(w, r, err)
		return
	}

	stageData.BuildOrder = len(pipelineData.Stages) + 1
	stageData.PipelineID = pipelineData.ID
	err = pipeline.InsertStage(db, stageData)
	if err != nil {
		log.Warning("addStageHandler> Cannot insert stage: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	k := cache.Key("application", projectKey, "*")
	cache.DeleteAll(k)
	cache.Delete(cache.Key("pipeline", projectKey, pipelineKey))

	WriteJSON(w, r, stageData, http.StatusCreated)
}

func getStageHandler(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {
	// Get project name in URL
	vars := mux.Vars(r)
	projectKey := vars["key"]
	pipelineKey := vars["permPipelineKey"]
	stageIDString := vars["stageID"]

	stageID, err := strconv.ParseInt(stageIDString, 10, 60)
	if err != nil {
		log.Warning("getStageHandler> Stage ID must be an int: %s", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Check if pipeline exist
	pipelineData, err := pipeline.LoadPipeline(db, projectKey, pipelineKey, false)
	if err != nil {
		WriteError(w, r, err)
		return
	}

	s, err := pipeline.LoadStage(db, pipelineData.ID, stageID)
	if err != nil {
		httpStatus := http.StatusInternalServerError
		if err == pipeline.ErrNoStage {
			httpStatus = http.StatusNotFound
			log.Warning("getStageHandler> Stage does not exist: %s", err)
		} else {
			log.Warning("getStageHandler> Cannot Load stage: %s", err)
		}
		w.WriteHeader(httpStatus)
		return
	}

	WriteJSON(w, r, s, http.StatusOK)
}

func moveStageHandler(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {

	// Get project name in URL
	vars := mux.Vars(r)
	projectKey := vars["key"]
	pipelineKey := vars["permPipelineKey"]

	// Get body
	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Warning("moveStageHandler> cannot read body: %s", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// get stage to move
	stageData, err := sdk.NewStage("").FromJSON(data)
	if err != nil {
		log.Warning("moveStageHandler> Cannot unmarshal body: %s", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if stageData.BuildOrder != 0 {
		// Check if pipeline exist
		pipelineData, err := pipeline.LoadPipeline(db, projectKey, pipelineKey, false)
		if err != nil {
			WriteError(w, r, err)
			return
		}

		// count stage for this pipeline
		nbStage, err := pipeline.CountStageByPipelineID(db, pipelineData.ID)
		if err != nil {
			log.Warning("moveStageHandler> Cannot count stage for pipeline %s : %s", pipelineData.Name, err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		if stageData.BuildOrder <= nbStage {
			// check if stage exist
			s, err := pipeline.LoadStage(db, pipelineData.ID, stageData.ID)
			if err != nil {
				httpStatus := http.StatusInternalServerError
				if err == pipeline.ErrNoStage {
					httpStatus = http.StatusNotFound
					log.Warning("moveStageHandler> Stage does not exist: %s", err)
				} else {
					log.Warning("moveStageHandler> Cannot Load stage: %s", err)
				}
				w.WriteHeader(httpStatus)
				return
			}

			err = pipeline.MoveStage(db, s, stageData.BuildOrder)
			if err != nil {
				log.Warning("moveStageHandler> Cannot move stage: %s", err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
		}
	}

	k := cache.Key("application", projectKey, "*")
	cache.DeleteAll(k)
	cache.Delete(cache.Key("pipeline", projectKey, pipelineKey))

	w.WriteHeader(http.StatusOK)
}

func updateStageHandler(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {

	// Get project name in URL
	vars := mux.Vars(r)
	projectKey := vars["key"]
	pipelineKey := vars["permPipelineKey"]
	stageIDString := vars["stageID"]

	// Get body
	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Warning("addStageHandler> cannot read body: %s", err)
		WriteError(w, r, err)
		return
	}

	stageData, err := sdk.NewStage("").FromJSON(data)
	if err != nil {
		log.Warning("addStageHandler> Cannot unmarshal body: %s", err)
		WriteError(w, r, err)
		return
	}

	stageID, err := strconv.ParseInt(stageIDString, 10, 60)
	if err != nil {
		log.Warning("addStageHandler> Stage ID must be an int: %s", err)
		WriteError(w, r, sdk.ErrInvalidID)
		return
	}
	if stageID != stageData.ID {
		log.Warning("addStageHandler> Stage ID doest not match")
		WriteError(w, r, sdk.ErrInvalidID)
		return
	}

	// Check if pipeline exist
	pipelineData, err := pipeline.LoadPipeline(db, projectKey, pipelineKey, false)
	if err != nil {
		WriteError(w, r, err)
		return
	}

	// check if stage exist
	s, err := pipeline.LoadStage(db, pipelineData.ID, stageData.ID)
	if err != nil {
		log.Warning("addStageHandler> Cannot Load stage: %s", err)
		WriteError(w, r, err)
		return
	}
	stageData.ID = s.ID

	tx, err := db.Begin()
	if err != nil {
		log.Warning("addStageHandler> Cannot start transaction: %s", err)
		WriteError(w, r, err)
		return
	}
	defer tx.Rollback()

	err = pipeline.UpdateStage(tx, stageData)
	if err != nil {
		log.Warning("addStageHandler> Cannot update stage: %s", err)
		WriteError(w, r, err)
		return
	}

	err = pipeline.UpdatePipelineLastModified(tx, pipelineData.ID)
	if err != nil {
		log.Warning("addStageHandler> Cannot update pipeline last_modified: %s", err)
		WriteError(w, r, err)
		return
	}

	err = tx.Commit()
	if err != nil {
		log.Warning("addStageHandler> Cannot commit transaction: %s", err)
		WriteError(w, r, err)
		return
	}

	k := cache.Key("application", projectKey, "*")
	cache.DeleteAll(k)
	cache.Delete(cache.Key("pipeline", projectKey, pipelineKey))

	w.WriteHeader(http.StatusOK)
}

func deleteStageHandler(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {

	// Get project name in URL
	vars := mux.Vars(r)
	projectKey := vars["key"]
	pipelineKey := vars["permPipelineKey"]
	stageIDString := vars["stageID"]

	// Check if pipeline exist
	pipelineData, err := pipeline.LoadPipeline(db, projectKey, pipelineKey, false)
	if err != nil {
		log.Warning("deleteStageHandler> Cannot load pipeline %s: %s", pipelineKey, err)
		WriteError(w, r, err)
		return
	}

	stageID, err := strconv.ParseInt(stageIDString, 10, 60)
	if err != nil {
		log.Warning("deleteStageHandler> Stage ID must be an int: %s", err)
		WriteError(w, r, sdk.ErrInvalidID)
		return
	}

	// check if stage exist
	s, err := pipeline.LoadStage(db, pipelineData.ID, stageID)
	if err != nil {
		log.Warning("deleteStageHandler> Cannot Load stage: %s", err)
		WriteError(w, r, err)
		return
	}

	tx, err := db.Begin()
	if err != nil {
		log.Warning("deleteStageHandler> Cannot start transaction: %s", err)
		WriteError(w, r, err)
		return
	}
	defer tx.Rollback()

	err = pipeline.DeleteStageByID(tx, s, c.User.ID)
	if err != nil {
		log.Warning("deleteStageHandler> Cannot Delete stage: %s", err)
		WriteError(w, r, err)
		return
	}

	err = pipeline.UpdatePipelineLastModified(tx, pipelineData.ID)
	if err != nil {
		log.Warning("deleteStageHandler> Cannot Update pipeline last_modified: %s", err)
		WriteError(w, r, err)
		return
	}

	err = tx.Commit()
	if err != nil {
		log.Warning("deleteStageHandler> Cannot commit transaction: %s", err)
		WriteError(w, r, err)
		return
	}

	k := cache.Key("application", projectKey, "*")
	cache.DeleteAll(k)
	cache.Delete(cache.Key("pipeline", projectKey, pipelineKey))

	w.WriteHeader(http.StatusOK)
}
