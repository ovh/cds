package workflowtemplate

import (
	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/sdk"
)

// AggregateOnWorkflowTemplate set group for each workflow template.
func AggregateOnWorkflowTemplate(db gorp.SqlExecutor, wts ...*sdk.WorkflowTemplate) error {
	gs := []sdk.Group{}

	if err := gorpmapping.GetAll(db,
		gorpmapping.NewQuery(`SELECT * FROM "group" WHERE id = ANY(string_to_array($1, ',')::int[])`).
			Args(gorpmapping.IDsToQueryString(sdk.WorkflowTemplatesToGroupIDs(wts))),
		&gs,
	); err != nil {
		return sdk.WrapError(err, "cannot get groups")
	}

	m := make(map[int64]sdk.Group, len(gs))
	for i := range gs {
		m[gs[i].ID] = gs[i]
	}

	for _, wt := range wts {
		if g, ok := m[wt.GroupID]; ok {
			wt.Group = &g
		}
	}

	return nil
}
