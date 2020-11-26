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
		id := service.FormInt64(r, "nodeID")

		proj, err := project.Load(ctx, api.mustDB(), key, project.LoadOptions.WithVariables, project.LoadOptions.WithIntegrations, project.LoadOptions.WithKeys)
		if err != nil {
			return sdk.WrapError(err, "unable to load project")
		}

		wf, err := workflow.Load(ctx, api.mustDB(), api.Cache, *proj, name, workflow.LoadOptions{})
		if err != nil {
			return sdk.WrapError(err, "unable to load workflow")
		}

		data := struct {
			Operators      map[string]string `json:"operators"`
			ConditionNames []string          `json:"names"`
		}{
			Operators: sdk.WorkflowConditionsOperators,
		}

		wr, err := workflow.LoadLastRun(api.mustDB(), key, name, workflow.LoadRunOptions{})
		if err != nil {
			if !sdk.ErrorIs(err, sdk.ErrNotFound) {
				return sdk.WrapError(err, "unable to load last run workflow")
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
			ancestorIds := refNode.Ancestors(wf.WorkflowData)

			params, err = workflow.NodeBuildParametersFromWorkflow(*proj, wf, refNode, ancestorIds)
			if err != nil {
				return sdk.WrapError(err, "unable to load build parameters from workflow")
			}

			sdk.AddParameter(&params, "cds.dest.pipeline", sdk.StringParameter, "")
			sdk.AddParameter(&params, "cds.status", sdk.StringParameter, "")
			sdk.AddParameter(&params, "cds.manual", sdk.StringParameter, "")

			if refNode != nil {
				if refNode.Context != nil && refNode.Context.ApplicationID != 0 {
					sdk.AddParameter(&params, "cds.dest.application", sdk.StringParameter, "")
				}
				if refNode.Context != nil && refNode.Context.EnvironmentID != 0 {
					sdk.AddParameter(&params, "cds.dest.environment", sdk.StringParameter, "")
				}
			}
		}

		if sdk.ParameterFind(params, "git.repository") == nil {
			data.ConditionNames = append(data.ConditionNames, sdk.BasicGitVariableNames...)
		}
		if sdk.ParameterFind(params, "git.tag") == nil {
			data.ConditionNames = append(data.ConditionNames, "git.tag")
		}

		for _, p := range params {
			data.ConditionNames = append(data.ConditionNames, p.Name)
		}

		sort.Strings(data.ConditionNames)
		return service.WriteJSON(w, data, http.StatusOK)
	}
}

func (api *API) getWorkflowTriggerHookConditionHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		data := struct {
			Operators      map[string]string `json:"operators"`
			ConditionNames []string          `json:"names"`
		}{
			Operators: sdk.WorkflowConditionsOperators,
		}

		data.ConditionNames = append(data.ConditionNames, sdk.BasicGitVariableNames...)
		data.ConditionNames = append(data.ConditionNames, "git.tag")
		data.ConditionNames = append(data.ConditionNames, "payload")

		sort.Strings(data.ConditionNames)
		return service.WriteJSON(w, data, http.StatusOK)
	}
}
