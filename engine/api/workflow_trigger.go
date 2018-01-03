package api

import (
	"context"
	"net/http"
	"sort"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/sdk"
)

func (api *API) getWorkflowTriggerConditionHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars["key"]
		name := vars["permWorkflowName"]

		id, errID := requestVarInt(r, "nodeID")
		if errID != nil {
			return errID
		}

		proj, errproj := project.Load(api.mustDB(), api.Cache, key, getUser(ctx), project.LoadOptions.WithVariables)
		if errproj != nil {
			return sdk.WrapError(errproj, "getWorkflowTriggerConditionHandler> Unable to load project")
		}

		wf, errw := workflow.Load(api.mustDB(), api.Cache, key, name, getUser(ctx))
		if errw != nil {
			return sdk.WrapError(errw, "getWorkflowTriggerConditionHandler> Unable to load workflow")
		}

		data := struct {
			Operators      map[string]string `json:"operators"`
			ConditionNames []string          `json:"names"`
		}{
			Operators: sdk.WorkflowConditionsOperators,
		}

		wr, errr := workflow.LoadLastRun(api.mustDB(), key, name, false)
		if errr != nil {
			if errr != sdk.ErrWorkflowNotFound {
				return sdk.WrapError(errr, "getWorkflowTriggerConditionHandler> Unable to load last run workflow")
			}
		}

		refNode := wf.GetNode(id)
		if refNode == nil {
			return sdk.WrapError(sdk.ErrWorkflowNodeNotFound, "getWorkflowTriggerConditionHandler> Unable to load workflow node")
		}

		params := []sdk.Parameter{}
		// If there is a workflow run, try to get build parameter from it
		if wr != nil {
			var errp error
			params, errp = workflow.NodeBuildParametersFromRun(*wr, id)
			if errp != nil {
				return sdk.WrapError(errp, "getWorkflowTriggerConditionHandler> Unable to load build parameters from workflow run")
			}
		}

		// If node node found in last workflow run
		if len(params) == 0 {
			var errp error
			ancestorIds := refNode.Ancestors(wf, true)
			params, errp = workflow.NodeBuildParametersFromWorkflow(api.mustDB(), api.Cache, proj, wf, refNode, ancestorIds)
			if errp != nil {
				return sdk.WrapError(errp, "getWorkflowTriggerConditionHandler> Unable to load build parameters from workflow")
			}
			sdk.AddParameter(&params, "cds.dest.pipeline", sdk.StringParameter, "")
			sdk.AddParameter(&params, "cds.status", sdk.StringParameter, "")
			sdk.AddParameter(&params, "cds.manual", sdk.StringParameter, "")

			if refNode.Context != nil && refNode.Context.Application != nil {
				sdk.AddParameter(&params, "cds.dest.application", sdk.StringParameter, "")
			}
			if refNode.Context != nil && refNode.Context.Environment != nil {
				sdk.AddParameter(&params, "cds.dest.environment", sdk.StringParameter, "")
			}
		}
		for _, p := range params {
			data.ConditionNames = append(data.ConditionNames, p.Name)
		}

		sort.Strings(data.ConditionNames)
		return WriteJSON(w, r, data, http.StatusOK)
	}
}
