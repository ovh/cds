package api

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
)

// postWorkflowAsCodeHandler Make the workflow as code
// @title Make the workflow as code
func (api *API) postWorkflowAsCodeHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars["key"]
		workflowName := vars["permWorkflowName"]

		u := getUser(ctx)

		proj, errP := project.Load(api.mustDB(), api.Cache, key, u, project.LoadOptions.WithApplicationWithDeploymentStrategies, project.LoadOptions.WithPipelines, project.LoadOptions.WithEnvironments, project.LoadOptions.WithPlatforms)
		if errP != nil {
			return sdk.WrapError(errP, "unable to load project")
		}
		wf, errW := workflow.Load(ctx, api.mustDB(), api.Cache, proj, workflowName, u, workflow.LoadOptions{
			DeepPipeline: true,
			WithLabels:   true,
			WithIcon:     true,
		})
		if errW != nil {
			return sdk.WrapError(errW, "unable to load workflow")
		}

		ope, err := workflow.UpdateAsCode(ctx, api.mustDB(), api.Cache, proj, wf, project.EncryptWithBuiltinKey, u)
		if err != nil {
			return sdk.WrapError(errW, "unable to migrate workflow as code")
		}

		sdk.GoRoutine(ctx, fmt.Sprintf("UpdateWorkflowAsCodeResult-%s", ope.UUID), func(ctx context.Context) {
			workflow.UpdateWorkflowAsCodeResult(context.Background(), api.mustDB(), api.Cache, proj, ope, wf, u)
		}, api.PanicDump())

		return service.WriteJSON(w, wf, http.StatusOK)
	}
}
