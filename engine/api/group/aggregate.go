package group

import (
	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/sdk"
)

// AggregateOnWorkflowTemplate set group for each workflow template.
func AggregateOnWorkflowTemplate(db gorp.SqlExecutor, wts ...*sdk.WorkflowTemplate) error {
	gs, err := GetAll(db, NewCriteria().IDs(sdk.WorkflowTemplatesToGroupIDs(wts)...))
	if err != nil {
		return err
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
