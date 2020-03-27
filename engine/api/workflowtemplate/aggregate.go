package workflowtemplate

import (
	"context"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/sdk"
)

// AggregateTemplateInstanceOnWorkflow set template instance data for each workflow.
func AggregateTemplateInstanceOnWorkflow(ctx context.Context, db gorp.SqlExecutor, ws ...*sdk.Workflow) error {
	if len(ws) == 0 {
		return nil
	}

	wtis, err := LoadInstancesByWorkflowIDs(ctx, db, sdk.WorkflowToIDs(ws))
	if err != nil {
		return err
	}
	if len(wtis) == 0 {
		return nil
	}

	mWorkflowTemplateInstances := make(map[int64]sdk.WorkflowTemplateInstance, len(wtis))
	for _, wti := range wtis {
		if wti.WorkflowID != nil {
			mWorkflowTemplateInstances[*wti.WorkflowID] = wti
		}
	}

	for _, w := range ws {
		if wti, ok := mWorkflowTemplateInstances[w.ID]; ok {
			w.TemplateInstance = &wti
		}
	}

	return nil
}
