package main

import (
	"net/http"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/context"
)

func getWorkflowTriggerCondition(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
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
