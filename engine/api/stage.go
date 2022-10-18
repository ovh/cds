package api

import (
	"context"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/event"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
)

func (api *API) addStageHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		// Get project name in URL
		vars := mux.Vars(r)
		projectKey := vars[permProjectKey]
		pipelineKey := vars["pipelineKey"]

		var stageData = &sdk.Stage{}
		if err := service.UnmarshalBody(r, stageData); err != nil {
			return err
		}

		// Check if pipeline exist
		pipelineData, err := pipeline.LoadPipeline(ctx, api.mustDB(), projectKey, pipelineKey, true)
		if err != nil {
			return sdk.WrapError(err, "addStageHandler")
		}
		if pipelineData.FromRepository != "" {
			return sdk.WithStack(sdk.ErrForbidden)
		}

		stageData.BuildOrder = len(pipelineData.Stages) + 1
		stageData.PipelineID = pipelineData.ID
		stageData.Enabled = true

		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WrapError(err, "Cannot start transaction")
		}
		defer tx.Rollback() // nolint

		if err := pipeline.CreateAudit(tx, pipelineData, pipeline.AuditAddStage, getUserConsumer(ctx)); err != nil {
			return sdk.WrapError(err, "Cannot create pipeline audit")
		}

		if err := pipeline.InsertStage(tx, stageData); err != nil {
			return sdk.WrapError(err, "Cannot insert stage")
		}

		if err := pipeline.UpdatePipeline(tx, pipelineData); err != nil {
			return sdk.WrapError(err, "Cannot update pipeline")
		}

		if err := tx.Commit(); err != nil {
			return sdk.WithStack(err)
		}

		if err := pipeline.LoadPipelineStage(ctx, api.mustDB(), pipelineData); err != nil {
			return sdk.WrapError(err, "Cannot load pipeline stages")
		}

		event.PublishPipelineStageAdd(ctx, projectKey, pipelineKey, *stageData, getUserConsumer(ctx))

		return service.WriteJSON(w, pipelineData, http.StatusCreated)
	}
}

func (api *API) getStageHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		// Get project name in URL
		vars := mux.Vars(r)
		projectKey := vars[permProjectKey]
		pipelineKey := vars["pipelineKey"]
		stageIDString := vars["stageID"]

		stageID, err := strconv.ParseInt(stageIDString, 10, 60)
		if err != nil {
			return sdk.WrapError(sdk.ErrInvalidID, "getStageHandler> Stage ID must be an int: %s", err)
		}

		// Check if pipeline exist
		pipelineData, err := pipeline.LoadPipeline(ctx, api.mustDB(), projectKey, pipelineKey, false)
		if err != nil {
			return sdk.WrapError(err, "error on pipeline load")
		}

		s, err := pipeline.LoadStage(api.mustDB(), pipelineData.ID, stageID)
		if err != nil {
			return sdk.WrapError(err, "Error on load stage")
		}

		return service.WriteJSON(w, s, http.StatusOK)
	}
}

func (api *API) moveStageHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		// Get project name in URL
		vars := mux.Vars(r)
		projectKey := vars[permProjectKey]
		pipelineKey := vars["pipelineKey"]

		var stageData = &sdk.Stage{}
		if err := service.UnmarshalBody(r, stageData); err != nil {
			return err
		}

		if stageData.BuildOrder < 1 {
			return sdk.WrapError(sdk.ErrWrongRequest, "moveStageHandler> Build Order must be greater than 0")
		}

		// Check if pipeline exist
		pipelineData, err := pipeline.LoadPipeline(ctx, api.mustDB(), projectKey, pipelineKey, true)
		if err != nil {
			return err
		}
		if pipelineData.FromRepository != "" {
			return sdk.WithStack(sdk.ErrForbidden)
		}

		// count stage for this pipeline
		nbStage, err := pipeline.CountStageByPipelineID(api.mustDB(), pipelineData.ID)
		if err != nil {
			return sdk.WrapError(err, "Cannot count stage for pipeline %s ", pipelineData.Name)
		}

		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WrapError(err, "Cannot start transaction")
		}
		defer tx.Rollback() // nolint

		if err := pipeline.CreateAudit(tx, pipelineData, pipeline.AuditMoveStage, getUserConsumer(ctx)); err != nil {
			return sdk.WrapError(err, "Cannot create pipeline audit")
		}

		var oldStage *sdk.Stage
		if stageData.BuildOrder <= nbStage {
			// check if stage exist
			var err error
			oldStage, err = pipeline.LoadStage(tx, pipelineData.ID, stageData.ID)
			if err != nil {
				return sdk.WrapError(err, "Cannot load stage")
			}

			if err := pipeline.MoveStage(tx, oldStage, stageData.BuildOrder, pipelineData); err != nil {
				return sdk.WrapError(err, "Cannot move stage")
			}
		}

		if err := pipeline.UpdatePipeline(tx, pipelineData); err != nil {
			return sdk.WrapError(err, "Cannot update pipeline")
		}

		if err := pipeline.LoadPipelineStage(ctx, tx, pipelineData); err != nil {
			return sdk.WrapError(err, "Cannot load stages")
		}

		if err := tx.Commit(); err != nil {
			return sdk.WithStack(err)
		}

		event.PublishPipelineStageMove(ctx, projectKey, pipelineKey, *stageData, oldStage.BuildOrder, getUserConsumer(ctx))
		return service.WriteJSON(w, pipelineData, http.StatusOK)
	}
}

func (api *API) updateStageHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		// Get project name in URL
		vars := mux.Vars(r)
		projectKey := vars[permProjectKey]
		pipelineKey := vars["pipelineKey"]
		stageIDString := vars["stageID"]

		var stageData = &sdk.Stage{}
		if err := service.UnmarshalBody(r, stageData); err != nil {
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
		pipelineData, err := pipeline.LoadPipeline(ctx, api.mustDB(), projectKey, pipelineKey, true)
		if err != nil {
			return err
		}
		if pipelineData.FromRepository != "" {
			return sdk.WithStack(sdk.ErrForbidden)
		}

		// check if stage exist
		s, err := pipeline.LoadStage(api.mustDB(), pipelineData.ID, stageData.ID)
		if err != nil {
			return sdk.WrapError(err, "Cannot Load stage")
		}
		stageData.ID = s.ID

		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WrapError(err, "Cannot start transaction")
		}
		defer tx.Rollback() // nolint

		if err := pipeline.CreateAudit(tx, pipelineData, pipeline.AuditUpdateStage, getUserConsumer(ctx)); err != nil {
			return sdk.WrapError(err, "Cannot create audit")
		}

		if err := pipeline.UpdateStage(tx, stageData); err != nil {
			return sdk.WrapError(err, "Cannot update stage")
		}

		if err := pipeline.UpdatePipeline(tx, pipelineData); err != nil {
			return sdk.WrapError(err, "Cannot update pipeline")
		}

		err = tx.Commit()
		if err != nil {
			return sdk.WithStack(err)
		}

		if err := pipeline.LoadPipelineStage(ctx, api.mustDB(), pipelineData); err != nil {
			return sdk.WrapError(err, "Cannot load stages")
		}

		event.PublishPipelineStageUpdate(ctx, projectKey, pipelineKey, *s, *stageData, getUserConsumer(ctx))
		return service.WriteJSON(w, pipelineData, http.StatusOK)
	}
}

func (api *API) deleteStageHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		// Get project name in URL
		vars := mux.Vars(r)
		projectKey := vars[permProjectKey]
		pipelineKey := vars["pipelineKey"]
		stageIDString := vars["stageID"]

		// Check if pipeline exist
		pipelineData, err := pipeline.LoadPipeline(ctx, api.mustDB(), projectKey, pipelineKey, true)
		if err != nil {
			return sdk.WrapError(err, "Cannot load pipeline %s", pipelineKey)
		}
		if pipelineData.FromRepository != "" {
			return sdk.WithStack(sdk.ErrForbidden)
		}

		stageID, err := strconv.ParseInt(stageIDString, 10, 60)
		if err != nil {
			return sdk.WrapError(sdk.ErrInvalidID, "deleteStageHandler> Stage ID must be an int: %s", err)
		}

		// check if stage exist
		s, err := pipeline.LoadStage(api.mustDB(), pipelineData.ID, stageID)
		if err != nil {
			return sdk.WrapError(err, "Cannot Load stage")
		}

		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WrapError(err, "Cannot start transaction")
		}
		defer tx.Rollback() // nolint

		if err := pipeline.CreateAudit(tx, pipelineData, pipeline.AuditDeleteStage, getUserConsumer(ctx)); err != nil {
			return sdk.WrapError(err, "Cannot create audit")
		}

		if err := pipeline.DeleteStageByID(ctx, tx, s); err != nil {
			return sdk.WrapError(err, "Cannot Delete stage")
		}

		if err := pipeline.UpdatePipeline(tx, pipelineData); err != nil {
			return sdk.WrapError(err, "Cannot update pipeline")
		}

		if err := tx.Commit(); err != nil {
			return sdk.WithStack(err)
		}

		if err := pipeline.LoadPipelineStage(ctx, api.mustDB(), pipelineData); err != nil {
			return sdk.WrapError(err, "Cannot load stages")
		}

		event.PublishPipelineStageDelete(ctx, projectKey, pipelineKey, *s, getUserConsumer(ctx))
		return service.WriteJSON(w, pipelineData, http.StatusOK)
	}
}

func (api *API) getStageConditionsHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		// Get project name in URL
		vars := mux.Vars(r)
		projectKey := vars[permProjectKey]
		pipelineKey := vars["pipelineKey"]

		data := struct {
			Operators      map[string]string `json:"operators"`
			ConditionNames []string          `json:"names"`
		}{
			Operators:      sdk.WorkflowConditionsOperators,
			ConditionNames: append(sdk.BasicGitVariableNames, "git.tag"),
		}

		// Check if pipeline exist
		pipelineData, err := pipeline.LoadPipeline(ctx, api.mustDB(), projectKey, pipelineKey, false)
		if err != nil {
			return sdk.WrapError(err, "Cannot load pipeline %s", pipelineKey)
		}

		pipParams, err := pipeline.GetAllParametersInPipeline(ctx, api.mustDB(), pipelineData.ID)
		if err != nil {
			return sdk.WrapError(err, "Cannot get all parameters in pipeline")
		}

		for _, pipParam := range pipParams {
			data.ConditionNames = append(data.ConditionNames, "cds.pip."+pipParam.Name)
		}

		// add cds variable
		data.ConditionNames = append(data.ConditionNames, sdk.BasicVariableNames...)

		return service.WriteJSON(w, data, http.StatusOK)
	}
}
