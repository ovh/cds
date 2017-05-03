package workflow

import (
	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/sdk"
)

// InsertOrUpdateHook inserts or updates a hook
func insertOrUpdateHook(db gorp.SqlExecutor, node *sdk.WorkflowNode, hook *sdk.WorkflowNodeHook) error {
	return nil
}

// DeleteHook deletes a hook
func deleteHook(db gorp.SqlExecutor, hook *sdk.WorkflowNodeHook) error {
	return nil
}
