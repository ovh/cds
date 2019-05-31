package api

import (
	"context"
	"net/http"
	"sort"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
)

func (api *API) getWorkflowTriggerConditionHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars["key"]
		name := vars["permWorkflowName"]

		id, errID := requestVarInt(r, "nodeID")
		if errID != nil {
			return errID
		}

		proj, errproj := project.Load(api.mustDB(), api.Cache, key, project.LoadOptions.WithVariables, project.LoadOptions.WithIntegrations)
		if errproj != nil {
			return sdk.WrapError(errproj, "getWorkflowTriggerConditionHandler> Unable to load project")
		}

		wf, errw := workflow.Load(ctx, api.mustDB(), api.Cache, proj, name, workflow.LoadOptions{})
		if errw != nil {
			return sdk.WrapError(errw, "getWorkflowTriggerConditionHandler> Unable to load workflow")
		}

		data := struct {
			Operators      map[string]string `json:"operators"`
			ConditionNames []string          `json:"names"`
		}{
			Operators: sdk.WorkflowConditionsOperators,
		}

		wr, errr := workflow.LoadLastRun(api.mustDB(), key, name, workflow.LoadRunOptions{})
		if errr != nil {
			if !sdk.ErrorIs(errr, sdk.ErrWorkflowNotFound) {
				return sdk.WrapError(errr, "getWorkflowTriggerConditionHandler> Unable to load last run workflow")
			}
		}

		params := []sdk.Parameter{}
		var refNode *sdk.Node
		if wr != nil {
			refNode = wr.Workflow.WorkflowData.NodeByID(id)
			var errp error
			params, errp = workflow.NodeBuildParametersFromRun(*wr, id)
			if errp != nil {
				return sdk.WrapError(errp, "getWorkflowTriggerConditionHandler> Unable to load build parameters from workflow run")
			}
			if len(params) == 0 {
				refNode = nil
			}
		}
		if refNode == nil {
			refNode = wf.WorkflowData.NodeByID(id)
			var errp error
			ancestorIds := refNode.Ancestors(wf.WorkflowData)
			params, errp = workflow.NodeBuildParametersFromWorkflow(ctx, api.mustDB(), api.Cache, proj, wf, refNode, ancestorIds)
			if errp != nil {
				return sdk.WrapError(errp, "getWorkflowTriggerConditionHandler> Unable to load build parameters from workflow")
			}
			sdk.AddParameter(&params, "cds.dest.pipeline", sdk.StringParameter, "")
			sdk.AddParameter(&params, "cds.status", sdk.StringParameter, "")
			sdk.AddParameter(&params, "cds.manual", sdk.StringParameter, "")

			if refNode.Context != nil && refNode.Context.ApplicationID != 0 {
				sdk.AddParameter(&params, "cds.dest.application", sdk.StringParameter, "")
			}
			if refNode.Context != nil && refNode.Context.EnvironmentID != 0 {
				sdk.AddParameter(&params, "cds.dest.environment", sdk.StringParameter, "")
			}
		}

		if refNode == nil {
			return sdk.WrapError(sdk.ErrWorkflowNodeNotFound, "getWorkflowTriggerConditionHandler> Unable to load workflow node")
		}

		if sdk.ParameterFind(&params, "git.repository") == nil {
			data.ConditionNames = append(data.ConditionNames, "git.repository")
			data.ConditionNames = append(data.ConditionNames, "git.branch")
			data.ConditionNames = append(data.ConditionNames, "git.message")
			data.ConditionNames = append(data.ConditionNames, "git.author")
			data.ConditionNames = append(data.ConditionNames, "git.hash")
			data.ConditionNames = append(data.ConditionNames, "git.hash.short")
		}
		if sdk.ParameterFind(&params, "git.tag") == nil {
			data.ConditionNames = append(data.ConditionNames, "git.tag")
		}

		for _, p := range params {
			data.ConditionNames = append(data.ConditionNames, p.Name)
		}

		sort.Strings(data.ConditionNames)
		return service.WriteJSON(w, data, http.StatusOK)
	}
}
