package api

import (
	"context"
	"net/http"
	"sort"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
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

		wr, errr := workflow.LoadLastRun(api.mustDB(), key, name)
		if errr != nil {
			if errr != sdk.ErrWorkflowNotFound {
				return sdk.WrapError(errr, "getWorkflowTriggerConditionHandler> Unable to load last run workflow")
			}
		}

		if wr != nil {
			params, errp := workflow.NodeBuildParameters(proj, wf, wr, id, getUser(ctx))
			if errp != nil {
				return sdk.WrapError(errp, "getWorkflowTriggerConditionHandler> Unable to load build parameters")
			}

			var statusParamFound bool
			for _, p := range params {
				if p.Name == "cds.status" {
					statusParamFound = true
				}
				data.ConditionNames = append(data.ConditionNames, p.Name)
			}

			if !statusParamFound {
				data.ConditionNames = append(data.ConditionNames, "cds.status")
			}
		}

		data.ConditionNames = append(data.ConditionNames, "cds.dest.pipeline", "cds.manual")

		refNode := wf.GetNode(id)
		if refNode == nil {
			return sdk.WrapError(sdk.ErrWorkflowNodeNotFound, "getWorkflowTriggerConditionHandler> Unable to load workflow node")
		}
		if refNode.Context != nil && refNode.Context.Application != nil {
			data.ConditionNames = append(data.ConditionNames, "cds.dest.application")
		}
		if refNode.Context != nil && refNode.Context.Environment != nil {
			data.ConditionNames = append(data.ConditionNames, "cds.dest.environment")
		}

		ancestorIds := refNode.Ancestors(wf, true)
		for _, aID := range ancestorIds {
			ancestor := wf.GetNode(aID)
			if ancestor == nil {
				continue
			}
			var found bool
			for _, s := range data.ConditionNames {
				if s == "workflow."+ancestor.Name+".status" {
					found = true
				}
			}
			if !found {
				data.ConditionNames = append(data.ConditionNames, "workflow."+ancestor.Name+".status")
			}
		}
		sort.Strings(data.ConditionNames)

		return WriteJSON(w, r, data, http.StatusOK)
	}
}

func (api *API) getWorkflowTriggerJoinConditionHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars["key"]
		name := vars["permWorkflowName"]

		id, errID := requestVarInt(r, "joinID")
		if errID != nil {
			return errID
		}

		proj, errproj := project.Load(api.mustDB(), api.Cache, key, getUser(ctx), project.LoadOptions.WithVariables)
		if errproj != nil {
			return sdk.WrapError(errproj, "getWorkflowTriggerConditionHandler> Unable to load project")
		}

		wf, errw := workflow.Load(api.mustDB(), api.Cache, key, name, getUser(ctx))
		if errw != nil {
			return sdk.WrapError(errw, "getWorkflowTriggerJoinConditionHandler> Unable to load workflow")
		}

		wr, errr := workflow.LoadLastRun(api.mustDB(), key, name)
		if errr != nil {
			if errr != sdk.ErrWorkflowNotFound {
				return sdk.WrapError(errr, "getWorkflowTriggerJoinConditionHandler> Unable to load last run workflow")
			}
		}

		j := wf.GetJoin(id)
		if j == nil {
			return sdk.ErrWorkflowNodeJoinNotFound
		}

		data := struct {
			Operators      map[string]string `json:"operators"`
			ConditionNames []string          `json:"names"`
		}{
			Operators: sdk.WorkflowConditionsOperators,
		}

		//First we merge all build parameters from all source nodes
		allparams := map[string]string{}
		for _, i := range j.SourceNodeIDs {
			params, errp := workflow.NodeBuildParameters(proj, wf, wr, i, getUser(ctx))
			if errp != nil {
				return sdk.WrapError(errr, "getWorkflowTriggerJoinConditionHandler> Unable to load build parameters")
			}
			allparams = sdk.ParametersMapMerge(allparams, sdk.ParametersToMap(params))
		}
		for k := range allparams {
			data.ConditionNames = append(data.ConditionNames, k)
		}

		//Then we push all ancestors status if needed
		for _, i := range j.SourceNodeIDs {
			refNode := wf.GetNode(i)
			if refNode == nil {
				log.Error("getWorkflowTriggerJoinConditionHandler> Unable to get node %d", i)
				continue
			}
			var found bool
			for _, s := range data.ConditionNames {
				if s == "workflow."+refNode.Name+".status" {
					found = true
				}
			}
			if !found {
				data.ConditionNames = append(data.ConditionNames, "workflow."+refNode.Name+".status")
			}
			ancestorIds := refNode.Ancestors(wf, true)
			for _, aID := range ancestorIds {
				ancestor := wf.GetNode(aID)
				if ancestor == nil {
					continue
				}
				var found bool
				for _, s := range data.ConditionNames {
					if s == "workflow."+ancestor.Name+".status" {
						found = true
					}
				}
				if !found {
					data.ConditionNames = append(data.ConditionNames, "workflow."+ancestor.Name+".status")
				}
			}
		}
		data.ConditionNames = append(data.ConditionNames, "cds.manual")
		sort.Strings(data.ConditionNames)

		return WriteJSON(w, r, data, http.StatusOK)
	}
}
