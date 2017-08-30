package api

import (
	"net/http"

	"github.com/go-gorp/gorp"
	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/businesscontext"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/sdk"
)

func getWorkflowTriggerConditionHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *businesscontext.Ctx) error {
	vars := mux.Vars(r)
	key := vars["permProjectKey"]
	name := vars["workflowName"]

	id, errID := requestVarInt(r, "nodeID")
	if errID != nil {
		return errID
	}

	proj, errproj := project.Load(db, key, c.User, project.LoadOptions.WithVariables)
	if errproj != nil {
		return sdk.WrapError(errproj, "getWorkflowTriggerConditionHandler> Unable to load project")
	}

	wf, errw := workflow.Load(db, key, name, c.User)
	if errw != nil {
		return sdk.WrapError(errw, "getWorkflowTriggerConditionHandler> Unable to load workflow")
	}

	wr, errr := workflow.LoadLastRun(db, key, name)
	if errr != nil {
		if errr != sdk.ErrWorkflowNotFound {
			return sdk.WrapError(errr, "getWorkflowTriggerConditionHandler> Unable to load last run workflow")
		}
	}

	params, errp := workflow.NodeBuildParameters(proj, wf, wr, id, c.User)
	if errp != nil {
		return sdk.WrapError(errr, "getWorkflowTriggerConditionHandler> Unable to load build parameters")
	}

	data := struct {
		Operators      map[string]string `json:"operators"`
		ConditionNames []string          `json:"names"`
	}{
		Operators: sdk.WorkflowConditionsOperators,
	}

	for _, p := range params {
		data.ConditionNames = append(data.ConditionNames, p.Name)
	}

	data.ConditionNames = append(data.ConditionNames, "cds.dest.pipeline")

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

	return WriteJSON(w, r, data, http.StatusOK)
}

func getWorkflowTriggerJoinConditionHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *businesscontext.Ctx) error {
	vars := mux.Vars(r)
	key := vars["permProjectKey"]
	name := vars["workflowName"]

	id, errID := requestVarInt(r, "joinID")
	if errID != nil {
		return errID
	}

	proj, errproj := project.Load(db, key, c.User, project.LoadOptions.WithVariables)
	if errproj != nil {
		return sdk.WrapError(errproj, "getWorkflowTriggerConditionHandler> Unable to load project")
	}

	wf, errw := workflow.Load(db, key, name, c.User)
	if errw != nil {
		return sdk.WrapError(errw, "getWorkflowTriggerJoinConditionHandler> Unable to load workflow")
	}

	wr, errr := workflow.LoadLastRun(db, key, name)
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

	allparams := map[string]string{}
	for _, i := range j.SourceNodeIDs {
		params, errp := workflow.NodeBuildParameters(proj, wf, wr, i, c.User)
		if errp != nil {
			return sdk.WrapError(errr, "getWorkflowTriggerJoinConditionHandler> Unable to load build parameters")
		}
		allparams = sdk.ParametersMapMerge(allparams, sdk.ParametersToMap(params))
	}

	for k := range allparams {
		data.ConditionNames = append(data.ConditionNames, k)
	}

	return WriteJSON(w, r, data, http.StatusOK)
}
