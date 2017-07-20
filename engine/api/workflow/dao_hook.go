package workflow

import (
	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/sdk"
)

// insertHook inserts a hook
func insertHook(db gorp.SqlExecutor, node *sdk.WorkflowNode, hook *sdk.WorkflowNodeHook) error {
	hook.WorkflowNodeID = node.ID

	return nil
}

// DeleteHook deletes a hook
func deleteHook(db gorp.SqlExecutor, node *sdk.WorkflowNode, hook *sdk.WorkflowNodeHook) error {

	return nil
}
