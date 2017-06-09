package main

import (
	"net/http"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/businesscontext"
)

func getWorkflowTriggerCondition(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *businesscontext.Ctx) error {
	operators := []string{"=", "!=", "<", "<=", ">=", ">"}

	data := struct {
		Operators      []string `json:"operators" db:"id"`
		ConditionNames []string `json:"names" db:"id"`
	}{
		operators,
		[]string{"git.branch"},
	}

	return WriteJSON(w, r, data, http.StatusOK)
}

func getWorkflowTriggerJoinCondition(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *businesscontext.Ctx) error {
	operators := []string{"=", "!=", "<", "<=", ">=", ">"}

	data := struct {
		Operators      []string `json:"operators" db:"id"`
		ConditionNames []string `json:"names" db:"id"`
	}{
		operators,
		[]string{"git.branch"},
	}

	return WriteJSON(w, r, data, http.StatusOK)
}
