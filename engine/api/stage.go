package api

import (
	"context"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/event"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/sdk"
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
		pipelineData, err := pipeline.LoadPipeline(api.mustDB(), projectKey, pipelineKey, true)
		if err != nil {
			return sdk.WrapError(err, "addStageHandler")
		}

		stageData.BuildOrder = len(pipelineData.Stages) + 1
		stageData.PipelineID = pipelineData.ID
		stageData.Enabled = true

		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WrapError(err, "addStageHandler> Cannot start transaction")
		}
		defer tx.Rollback()

		if err := pipeline.CreateAudit(tx, pipelineData, pipeline.AuditAddStage, getUser(ctx)); err != nil {
			return sdk.WrapError(err, "addStageHandler> Cannot create pipeline audit")
		}

		if err := pipeline.InsertStage(tx, stageData); err != nil {
			return sdk.WrapError(err, "addStageHandler> Cannot insert stage")
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "addStageHandler> Cannot commit transaction")
		}

		if err := pipeline.LoadPipelineStage(ctx, api.mustDB(), pipelineData); err != nil {
			return sdk.WrapError(err, "addStageHandler> Cannot load pipeline stages")
		}

		event.PublishPipelineStageAdd(projectKey, pipelineKey, *stageData, getUser(ctx))

		return WriteJSON(w, pipelineData, http.StatusCreated)
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
			return sdk.WrapError(sdk.ErrInvalidID, "getStageHandler> Stage ID must be an int: %s", err)
		}

		// Check if pipeline exist
		pipelineData, err := pipeline.LoadPipeline(api.mustDB(), projectKey, pipelineKey, false)
		if err != nil {
			return sdk.WrapError(err, "getStageHandler> error on pipeline load")
		}

		s, err := pipeline.LoadStage(api.mustDB(), pipelineData.ID, stageID)
		if err != nil {
			return sdk.WrapError(err, "getStageHandler> Error on load stage")
		}

		return WriteJSON(w, s, http.StatusOK)
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
			return sdk.WrapError(sdk.ErrWrongRequest, "moveStageHandler> Build Order must be greater than 0")
		}

		// Check if pipeline exist
		pipelineData, err := pipeline.LoadPipeline(api.mustDB(), projectKey, pipelineKey, true)
		if err != nil {
			return err
		}

		// count stage for this pipeline
		nbStage, err := pipeline.CountStageByPipelineID(api.mustDB(), pipelineData.ID)
		if err != nil {
			return sdk.WrapError(err, "moveStageHandler> Cannot count stage for pipeline %s ", pipelineData.Name)
		}

		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WrapError(err, "addStageHandler> Cannot start transaction")
		}
		defer tx.Rollback()

		if err := pipeline.CreateAudit(tx, pipelineData, pipeline.AuditMoveStage, getUser(ctx)); err != nil {
			return sdk.WrapError(err, "addStageHandler> Cannot create pipeline audit")
		}

		var oldStage *sdk.Stage
		if stageData.BuildOrder <= nbStage {
			// check if stage exist
			var err error
			oldStage, err = pipeline.LoadStage(tx, pipelineData.ID, stageData.ID)
			if err != nil {
				return sdk.WrapError(err, "moveStageHandler> Cannot load stage")
			}

			if err := pipeline.MoveStage(tx, oldStage, stageData.BuildOrder, pipelineData); err != nil {
				return sdk.WrapError(err, "moveStageHandler> Cannot move stage")
			}
		}

		if err := pipeline.LoadPipelineStage(ctx, tx, pipelineData); err != nil {
			return sdk.WrapError(err, "moveStageHandler> Cannot load stages")
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "moveStageHandler> Cannot commit transaction")
		}

		event.PublishPipelineStageMove(projectKey, pipelineKey, *stageData, oldStage.BuildOrder, getUser(ctx))
		return WriteJSON(w, pipelineData, http.StatusOK)
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
			return sdk.WrapError(sdk.ErrInvalidID, "updateStageHandler> Stage ID must be an int: %s", err)
		}
		if stageID != stageData.ID {
			return sdk.WrapError(sdk.ErrInvalidID, "updateStageHandler> Stage ID doest not match")
		}

		// Check if pipeline exist
		pipelineData, err := pipeline.LoadPipeline(api.mustDB(), projectKey, pipelineKey, true)
		if err != nil {
			return err
		}

		// check if stage exist
		s, err := pipeline.LoadStage(api.mustDB(), pipelineData.ID, stageData.ID)
		if err != nil {
			return sdk.WrapError(err, "updateStageHandler> Cannot Load stage")
		}
		stageData.ID = s.ID

		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WrapError(err, "updateStageHandler> Cannot start transaction")
		}
		defer tx.Rollback()

		if err := pipeline.CreateAudit(tx, pipelineData, pipeline.AuditUpdateStage, getUser(ctx)); err != nil {
			return sdk.WrapError(err, "updateStageHandler> Cannot create audit")
		}

		if err := pipeline.UpdateStage(tx, stageData); err != nil {
			return sdk.WrapError(err, "updateStageHandler> Cannot update stage")
		}

		err = tx.Commit()
		if err != nil {
			return sdk.WrapError(err, "updateStageHandler> Cannot commit transaction")
		}

		if err := pipeline.LoadPipelineStage(ctx, api.mustDB(), pipelineData); err != nil {
			return sdk.WrapError(err, "updateStageHandler> Cannot load stages")
		}

		event.PublishPipelineStageUpdate(projectKey, pipelineKey, *s, *stageData, getUser(ctx))
		return WriteJSON(w, pipelineData, http.StatusOK)
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
		pipelineData, err := pipeline.LoadPipeline(api.mustDB(), projectKey, pipelineKey, true)
		if err != nil {
			return sdk.WrapError(err, "deleteStageHandler> Cannot load pipeline %s", pipelineKey)
		}

		stageID, err := strconv.ParseInt(stageIDString, 10, 60)
		if err != nil {
			return sdk.WrapError(sdk.ErrInvalidID, "deleteStageHandler> Stage ID must be an int: %s", err)
		}

		// check if stage exist
		s, err := pipeline.LoadStage(api.mustDB(), pipelineData.ID, stageID)
		if err != nil {
			return sdk.WrapError(err, "deleteStageHandler> Cannot Load stage")
		}

		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WrapError(err, "deleteStageHandler> Cannot start transaction")
		}
		defer tx.Rollback()

		if err := pipeline.CreateAudit(tx, pipelineData, pipeline.AuditDeleteStage, getUser(ctx)); err != nil {
			return sdk.WrapError(err, "deleteStageHandler> Cannot create audit")
		}

		if err := pipeline.DeleteStageByID(tx, s, getUser(ctx).ID); err != nil {
			return sdk.WrapError(err, "deleteStageHandler> Cannot Delete stage")
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "deleteStageHandler> Cannot commit transaction")
		}

		if err := pipeline.LoadPipelineStage(ctx, api.mustDB(), pipelineData); err != nil {
			return sdk.WrapError(err, "deleteStageHandler> Cannot load stages")
		}

		event.PublishPipelineStageDelete(projectKey, pipelineKey, *s, getUser(ctx))
		return WriteJSON(w, pipelineData, http.StatusOK)
	}
}
