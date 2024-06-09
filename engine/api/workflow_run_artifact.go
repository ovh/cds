package api

import (
	"context"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/cdn"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
)

func (api *API) getWorkflowRunArtifactLinksHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)

		projectKey := vars["key"]
		workflowName := vars["permWorkflowNameAdvanced"]
		runNumber, err := requestVarInt(r, "number")
		if err != nil {
			return err
		}

		wr, err := workflow.LoadRun(ctx, api.mustDB(), projectKey, workflowName, runNumber, workflow.LoadRunOptions{DisableDetailedNodeRun: true})
		if err != nil {
			return err
		}

		result, err := cdn.ListItems(ctx, api.mustDB(), sdk.CDNTypeItemRunResult, map[string]string{cdn.ParamRunID: strconv.Itoa(int(wr.ID))})
		if err != nil {
			return err
		}

		return service.WriteJSON(w, result, http.StatusOK)
	}
}
