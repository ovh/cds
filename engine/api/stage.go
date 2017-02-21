package main

import (
	"net/http"
	"strconv"

	"github.com/go-gorp/gorp"
	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/context"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/log"
	"github.com/ovh/cds/sdk"
)

func addStageHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	// Get project name in URL
	vars := mux.Vars(r)
	projectKey := vars["key"]
	pipelineKey := vars["permPipelineKey"]

	var stageData = &sdk.Stage{}
	if err := UnmarshalBody(r, stageData); err != nil {
		return err
	}

	// Check if pipeline exist
	pipelineData, err := pipeline.LoadPipeline(db, projectKey, pipelineKey, false)
	if err != nil {
		return err
	}

	if err := pipeline.LoadPipelineStage(db, pipelineData); err != nil {
		log.Warning("addStageHandler> Cannot load pipeline stages: %s", err)
		return err
	}

	stageData.BuildOrder = len(pipelineData.Stages) + 1
	stageData.PipelineID = pipelineData.ID

	tx, err := db.Begin()
	if err != nil {
		log.Warning("addStageHandler> Cannot start transaction: %s", err)
		return err
	}
	defer tx.Rollback()

	if err := pipeline.InsertStage(db, stageData); err != nil {
		log.Warning("addStageHandler> Cannot insert stage: %s", err)
		return err
	}

	if err := pipeline.UpdatePipelineLastModified(tx, pipelineData); err != nil {
		log.Warning("addStageHandler> Cannot update pipeline last modified date: %s", err)
		return err
	}

	if err := tx.Commit(); err != nil {
		log.Warning("addStageHandler> Cannot commit transaction: %s", err)
		return err
	}

	if err := pipeline.LoadPipelineStage(db, pipelineData); err != nil {
		log.Warning("addStageHandler> Cannot load pipeline stages: %s", err)
		return err
	}

	k := cache.Key("application", projectKey, "*")
	cache.DeleteAll(k)
	cache.Delete(cache.Key("pipeline", projectKey, pipelineKey))

	return WriteJSON(w, r, pipelineData, http.StatusCreated)
}

func getStageHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	// Get project name in URL
	vars := mux.Vars(r)
	projectKey := vars["key"]
	pipelineKey := vars["permPipelineKey"]
	stageIDString := vars["stageID"]

	stageID, err := strconv.ParseInt(stageIDString, 10, 60)
	if err != nil {
		log.Warning("getStageHandler> Stage ID must be an int: %s", err)
		return sdk.ErrWrongRequest
	}

	// Check if pipeline exist
	pipelineData, err := pipeline.LoadPipeline(db, projectKey, pipelineKey, false)
	if err != nil {
		return err
	}

	s, err := pipeline.LoadStage(db, pipelineData.ID, stageID)
	if err != nil {
		return err
	}

	return WriteJSON(w, r, s, http.StatusOK)
}

func moveStageHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	// Get project name in URL
	vars := mux.Vars(r)
	projectKey := vars["key"]
	pipelineKey := vars["permPipelineKey"]

	var stageData = &sdk.Stage{}
	if err := UnmarshalBody(r, stageData); err != nil {
		return err
	}

	if stageData.BuildOrder < 1 {
		log.Warning("moveStageHandler> Build Order must be greater than 0")
		return sdk.ErrWrongRequest
	}

	// Check if pipeline exist
	pipelineData, err := pipeline.LoadPipeline(db, projectKey, pipelineKey, false)
	if err != nil {
		return err
	}

	// count stage for this pipeline
	nbStage, err := pipeline.CountStageByPipelineID(db, pipelineData.ID)
	if err != nil {
		log.Warning("moveStageHandler> Cannot count stage for pipeline %s : %s", pipelineData.Name, err)
		return err
	}

	if stageData.BuildOrder <= nbStage {
		// check if stage exist
		s, err := pipeline.LoadStage(db, pipelineData.ID, stageData.ID)
		if err != nil {
			log.Warning("moveStageHandler> Cannot load stage: %s", err)
			return err
		}

		if err := pipeline.MoveStage(db, s, stageData.BuildOrder, pipelineData); err != nil {
			log.Warning("moveStageHandler> Cannot move stage: %s", err)
			return err
		}
	}

	if err := pipeline.LoadPipelineStage(db, pipelineData); err != nil {
		log.Warning("moveStageHandler> Cannot load stages: %s", err)
		return err
	}

	k := cache.Key("application", projectKey, "*")
	cache.DeleteAll(k)
	cache.Delete(cache.Key("pipeline", projectKey, pipelineKey))

	return WriteJSON(w, r, pipelineData, http.StatusOK)
}

func updateStageHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	// Get project name in URL
	vars := mux.Vars(r)
	projectKey := vars["key"]
	pipelineKey := vars["permPipelineKey"]
	stageIDString := vars["stageID"]

	var stageData = &sdk.Stage{}
	if err := UnmarshalBody(r, stageData); err != nil {
		return err
	}

	stageID, err := strconv.ParseInt(stageIDString, 10, 60)
	if err != nil {
		log.Warning("addStageHandler> Stage ID must be an int: %s", err)
		return sdk.ErrInvalidID
	}
	if stageID != stageData.ID {
		log.Warning("addStageHandler> Stage ID doest not match")
		return sdk.ErrInvalidID
	}

	// Check if pipeline exist
	pipelineData, err := pipeline.LoadPipeline(db, projectKey, pipelineKey, false)
	if err != nil {
		return err
	}

	// check if stage exist
	s, err := pipeline.LoadStage(db, pipelineData.ID, stageData.ID)
	if err != nil {
		log.Warning("addStageHandler> Cannot Load stage: %s", err)
		return err
	}
	stageData.ID = s.ID

	tx, err := db.Begin()
	if err != nil {
		log.Warning("addStageHandler> Cannot start transaction: %s", err)
		return err
	}
	defer tx.Rollback()

	if err := pipeline.UpdateStage(tx, stageData); err != nil {
		log.Warning("addStageHandler> Cannot update stage: %s", err)
		return err
	}

	if err := pipeline.UpdatePipelineLastModified(tx, pipelineData); err != nil {
		log.Warning("addStageHandler> Cannot update pipeline last_modified: %s", err)
		return err
	}

	err = tx.Commit()
	if err != nil {
		log.Warning("addStageHandler> Cannot commit transaction: %s", err)
		return err
	}

	if err := pipeline.LoadPipelineStage(db, pipelineData); err != nil {
		log.Warning("addStageHandler> Cannot load stages: %s", err)
		return err
	}

	k := cache.Key("application", projectKey, "*")
	cache.DeleteAll(k)
	cache.Delete(cache.Key("pipeline", projectKey, pipelineKey))

	return WriteJSON(w, r, pipelineData, http.StatusOK)
}

func deleteStageHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {

	// Get project name in URL
	vars := mux.Vars(r)
	projectKey := vars["key"]
	pipelineKey := vars["permPipelineKey"]
	stageIDString := vars["stageID"]

	// Check if pipeline exist
	pipelineData, err := pipeline.LoadPipeline(db, projectKey, pipelineKey, false)
	if err != nil {
		log.Warning("deleteStageHandler> Cannot load pipeline %s: %s", pipelineKey, err)
		return err
	}

	stageID, err := strconv.ParseInt(stageIDString, 10, 60)
	if err != nil {
		log.Warning("deleteStageHandler> Stage ID must be an int: %s", err)
		return sdk.ErrInvalidID
	}

	// check if stage exist
	s, err := pipeline.LoadStage(db, pipelineData.ID, stageID)
	if err != nil {
		log.Warning("deleteStageHandler> Cannot Load stage: %s", err)
		return err
	}

	tx, err := db.Begin()
	if err != nil {
		log.Warning("deleteStageHandler> Cannot start transaction: %s", err)
		return err
	}
	defer tx.Rollback()

	if err := pipeline.DeleteStageByID(tx, s, c.User.ID); err != nil {
		log.Warning("deleteStageHandler> Cannot Delete stage: %s", err)
		return err
	}

	if err := pipeline.UpdatePipelineLastModified(tx, pipelineData); err != nil {
		log.Warning("deleteStageHandler> Cannot Update pipeline last_modified: %s", err)
		return err
	}

	if err := tx.Commit(); err != nil {
		log.Warning("deleteStageHandler> Cannot commit transaction: %s", err)
		return err
	}

	if err := pipeline.LoadPipelineStage(db, pipelineData); err != nil {
		log.Warning("deleteStageHandler> Cannot load stages: %s", err)
		return err
	}

	k := cache.Key("application", projectKey, "*")
	cache.DeleteAll(k)
	cache.Delete(cache.Key("pipeline", projectKey, pipelineKey))

	return WriteJSON(w, r, pipelineData, http.StatusOK)
}
