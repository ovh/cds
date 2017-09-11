package api

import (
	"context"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func (api *API) addStageHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		// Get project name in URL
		vars := mux.Vars(r)
		projectKey := vars["key"]
		pipelineKey := vars["permPipelineKey"]

		var stageData = &sdk.Stage{}
		if err := UnmarshalBody(r, stageData); err != nil {
			return err
		}

		// Check if pipeline exist
		pipelineData, err := pipeline.LoadPipeline(api.mustDB(), projectKey, pipelineKey, false)
		if err != nil {
			return err
		}

		if err := pipeline.LoadPipelineStage(api.mustDB(), pipelineData); err != nil {
			log.Warning("addStageHandler> Cannot load pipeline stages: %s", err)
			return err
		}

		stageData.BuildOrder = len(pipelineData.Stages) + 1
		stageData.PipelineID = pipelineData.ID
		stageData.Enabled = true

		tx, err := api.mustDB().Begin()
		if err != nil {
			log.Warning("addStageHandler> Cannot start transaction: %s", err)
			return err
		}
		defer tx.Rollback()

		if err := pipeline.InsertStage(api.mustDB(), stageData); err != nil {
			log.Warning("addStageHandler> Cannot insert stage: %s", err)
			return err
		}

		proj, errproj := project.Load(api.mustDB(), api.Cache, projectKey, getUser(ctx))
		if errproj != nil {
			return sdk.WrapError(errproj, "addStageHandler> unable to load project")
		}

		if err := pipeline.UpdatePipelineLastModified(tx, proj, pipelineData, getUser(ctx)); err != nil {
			log.Warning("addStageHandler> Cannot update pipeline last modified date: %s", err)
			return err
		}

		if err := tx.Commit(); err != nil {
			log.Warning("addStageHandler> Cannot commit transaction: %s", err)
			return err
		}

		if err := pipeline.LoadPipelineStage(api.mustDB(), pipelineData); err != nil {
			log.Warning("addStageHandler> Cannot load pipeline stages: %s", err)
			return err
		}

		return WriteJSON(w, r, pipelineData, http.StatusCreated)
	}
}

func (api *API) getStageHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
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
		pipelineData, err := pipeline.LoadPipeline(api.mustDB(), projectKey, pipelineKey, false)
		if err != nil {
			return err
		}

		s, err := pipeline.LoadStage(api.mustDB(), pipelineData.ID, stageID)
		if err != nil {
			return err
		}

		return WriteJSON(w, r, s, http.StatusOK)
	}
}

func (api *API) moveStageHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
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
		pipelineData, err := pipeline.LoadPipeline(api.mustDB(), projectKey, pipelineKey, false)
		if err != nil {
			return err
		}

		// count stage for this pipeline
		nbStage, err := pipeline.CountStageByPipelineID(api.mustDB(), pipelineData.ID)
		if err != nil {
			log.Warning("moveStageHandler> Cannot count stage for pipeline %s : %s", pipelineData.Name, err)
			return err
		}

		if stageData.BuildOrder <= nbStage {
			// check if stage exist
			s, err := pipeline.LoadStage(api.mustDB(), pipelineData.ID, stageData.ID)
			if err != nil {
				log.Warning("moveStageHandler> Cannot load stage: %s", err)
				return err
			}

			if err := pipeline.MoveStage(api.mustDB(), s, stageData.BuildOrder, pipelineData); err != nil {
				log.Warning("moveStageHandler> Cannot move stage: %s", err)
				return err
			}
		}

		if err := pipeline.LoadPipelineStage(api.mustDB(), pipelineData); err != nil {
			log.Warning("moveStageHandler> Cannot load stages: %s", err)
			return err
		}

		proj, errproj := project.Load(api.mustDB(), api.Cache, projectKey, getUser(ctx))
		if errproj != nil {
			return sdk.WrapError(errproj, "moveStageHandler> unable to load project")
		}

		if err := pipeline.UpdatePipelineLastModified(api.mustDB(), proj, pipelineData, getUser(ctx)); err != nil {
			return err
		}

		return WriteJSON(w, r, pipelineData, http.StatusOK)
	}
}

func (api *API) updateStageHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
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
		pipelineData, err := pipeline.LoadPipeline(api.mustDB(), projectKey, pipelineKey, false)
		if err != nil {
			return err
		}

		// check if stage exist
		s, err := pipeline.LoadStage(api.mustDB(), pipelineData.ID, stageData.ID)
		if err != nil {
			log.Warning("addStageHandler> Cannot Load stage: %s", err)
			return err
		}
		stageData.ID = s.ID

		tx, err := api.mustDB().Begin()
		if err != nil {
			log.Warning("addStageHandler> Cannot start transaction: %s", err)
			return err
		}
		defer tx.Rollback()

		if err := pipeline.UpdateStage(tx, stageData); err != nil {
			log.Warning("addStageHandler> Cannot update stage: %s", err)
			return err
		}

		proj, errproj := project.Load(api.mustDB(), api.Cache, projectKey, getUser(ctx))
		if errproj != nil {
			return sdk.WrapError(errproj, "addStageHandler> unable to load project")
		}

		if err := pipeline.UpdatePipelineLastModified(tx, proj, pipelineData, getUser(ctx)); err != nil {
			log.Warning("addStageHandler> Cannot update pipeline last_modified: %s", err)
			return err
		}

		err = tx.Commit()
		if err != nil {
			log.Warning("addStageHandler> Cannot commit transaction: %s", err)
			return err
		}

		if err := pipeline.LoadPipelineStage(api.mustDB(), pipelineData); err != nil {
			log.Warning("addStageHandler> Cannot load stages: %s", err)
			return err
		}

		return WriteJSON(w, r, pipelineData, http.StatusOK)
	}
}

func (api *API) deleteStageHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		// Get project name in URL
		vars := mux.Vars(r)
		projectKey := vars["key"]
		pipelineKey := vars["permPipelineKey"]
		stageIDString := vars["stageID"]

		// Check if pipeline exist
		pipelineData, err := pipeline.LoadPipeline(api.mustDB(), projectKey, pipelineKey, false)
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
		s, err := pipeline.LoadStage(api.mustDB(), pipelineData.ID, stageID)
		if err != nil {
			log.Warning("deleteStageHandler> Cannot Load stage: %s", err)
			return err
		}

		tx, err := api.mustDB().Begin()
		if err != nil {
			log.Warning("deleteStageHandler> Cannot start transaction: %s", err)
			return err
		}
		defer tx.Rollback()

		if err := pipeline.DeleteStageByID(tx, s, getUser(ctx).ID); err != nil {
			log.Warning("deleteStageHandler> Cannot Delete stage: %s", err)
			return err
		}

		proj, errproj := project.Load(api.mustDB(), api.Cache, projectKey, getUser(ctx))
		if errproj != nil {
			return sdk.WrapError(errproj, "deleteStageHandler> unable to load project")
		}

		if err := pipeline.UpdatePipelineLastModified(tx, proj, pipelineData, getUser(ctx)); err != nil {
			log.Warning("deleteStageHandler> Cannot Update pipeline last_modified: %s", err)
			return err
		}

		if err := tx.Commit(); err != nil {
			log.Warning("deleteStageHandler> Cannot commit transaction: %s", err)
			return err
		}

		if err := pipeline.LoadPipelineStage(api.mustDB(), pipelineData); err != nil {
			log.Warning("deleteStageHandler> Cannot load stages: %s", err)
			return err
		}

		return WriteJSON(w, r, pipelineData, http.StatusOK)
	}
}
