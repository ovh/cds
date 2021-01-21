package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/api/services"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/engine/featureflipping"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
)

func (api *API) getWorkflowRunArtifactLinksHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)

		projectKey := vars["key"]
		enabled := featureflipping.IsEnabled(ctx, gorpmapping.Mapper, api.mustDB(), "cdn-artifact", map[string]string{
			"project_key": projectKey,
		})
		if !enabled {
			return sdk.NewErrorFrom(sdk.ErrNotFound, "cdn is not enable for project %s", projectKey)
		}
		workflowName := vars["permWorkflowName"]
		runNumber, err := requestVarInt(r, "number")
		if err != nil {
			return err
		}

		wr, err := workflow.LoadRun(ctx, api.mustDB(), projectKey, workflowName, runNumber, workflow.LoadRunOptions{DisableDetailledNodeRun: true})
		if err != nil {
			return err
		}

		srvs, err := services.LoadAllByType(ctx, api.mustDB(), sdk.TypeCDN)
		if err != nil {
			return err
		}
		if len(srvs) == 0 {
			return sdk.WrapError(sdk.ErrNotFound, "No service found")
		}

		path := fmt.Sprintf("/service/item/%s?runid=%d", sdk.CDNTypeItemArtifact, wr.ID)
		btes, _, _, err := services.DoRequest(ctx, api.mustDB(), srvs, http.MethodGet, path, nil)
		if err != nil {
			return err
		}
		var cdnItems []sdk.CDNItem
		if err := json.Unmarshal(btes, &cdnItems); err != nil {
			return sdk.WithStack(err)
		}

		httpURL, err := services.GetCDNPublicHTTPAdress(ctx, api.mustDB())
		if err != nil {
			return err
		}

		resp := sdk.CDNItemLinks{
			CDNHttpURL: httpURL,
			Items:      cdnItems,
		}

		return service.WriteJSON(w, resp, http.StatusOK)
	}
}
