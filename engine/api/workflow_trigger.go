package main

import (
	"net/http"

	"github.com/go-gorp/gorp"
	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/businesscontext"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func getWorkflowTriggerConditionHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *businesscontext.Ctx) error {
	vars := mux.Vars(r)
	key := vars["permProjectKey"]
	name := vars["workflowName"]

	id, errID := requestVarInt(r, "nodeID")
	if errID != nil {
		return errID
	}

	wf, errw := workflow.Load(db, key, name, c.User)
	if errw != nil {
		return sdk.WrapError(errw, "getWorkflowTriggerConditionHandler> Unable to load workflow")
	}

	refNode := wf.GetNode(id)
	if refNode == nil {
		return sdk.WrapError(sdk.ErrWorkflowNodeNotFound, "getWorkflowTriggerConditionHandler> Unable to load workflow node")
	}

	data := struct {
		Operators      []string `json:"operators" db:"id"`
		ConditionNames []string `json:"names" db:"id"`
	}{
		Operators: sdk.WorkflowConditionsOperators,
	}

	//TODO what should we do if they is not last run ?
	wr, errr := workflow.LoadLastRun(db, key, name)
	if errr != nil {
		if errr != sdk.ErrWorkflowNotFound {
			return sdk.WrapError(errr, "getWorkflowTriggerConditionHandler> Unable to load las run workflow")
		}
		log.Warning("getWorkflowTriggerConditionHandler> Unable to find last run")
	}

	if wr != nil {
		for nodeID, nodeRuns := range wr.WorkflowNodeRuns {
			oldNode := wr.Workflow.GetNode(nodeID)
			if oldNode == nil {
				log.Warning("getWorkflowTriggerConditionHandler> Unable to find last run")
				break
			}
			if oldNode.EqualsTo(refNode) {
				for _, p := range nodeRuns[0].BuildParameters {
					data.ConditionNames = append(data.ConditionNames, p.Name)
				}
				break
			}
		}
	}

	data.ConditionNames = append(data.ConditionNames, "cds.dest.pip")
	if refNode.Context != nil && refNode.Context.Application != nil {
		data.ConditionNames = append(data.ConditionNames, "cds.dest.app")
	}
	if refNode.Context != nil && refNode.Context.Environment != nil {
		data.ConditionNames = append(data.ConditionNames, "cds.dest.env")
	}

	return WriteJSON(w, r, data, http.StatusOK)
}

func getWorkflowTriggerJoinConditionHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *businesscontext.Ctx) error {
	data := struct {
		Operators      []string `json:"operators" db:"id"`
		ConditionNames []string `json:"names" db:"id"`
	}{
		sdk.WorkflowConditionsOperators,
		[]string{"git.branch"},
	}

	return WriteJSON(w, r, data, http.StatusOK)
}
