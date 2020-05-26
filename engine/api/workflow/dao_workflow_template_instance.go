package workflow

import (
	"context"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/sdk"
)

// AggregateOnWorkflowTemplateInstance set workflow for each workflow template instance.
func AggregateOnWorkflowTemplateInstance(ctx context.Context, db gorp.SqlExecutor, wtis ...*sdk.WorkflowTemplateInstance) error {
	ws, err := LoadAllByIDs(ctx, db, sdk.WorkflowTemplateInstancesToWorkflowIDs(wtis))
	if err != nil {
		return err
	}

	m := make(map[int64]sdk.Workflow, len(ws))
	for i := range ws {
		m[ws[i].ID] = ws[i]
	}

	for _, wti := range wtis {
		if wti.WorkflowID != nil {
			if w, ok := m[*wti.WorkflowID]; ok {
				wti.Workflow = &w
			}
		}
	}

	return nil
}
