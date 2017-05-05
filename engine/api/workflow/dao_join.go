package workflow

import (
	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/sdk"
)

func loadJoins(db gorp.SqlExecutor, w *sdk.Workflow, u *sdk.User) ([]sdk.WorkflowNodeJoin, error) {
	return nil, nil
}

func loadJoin(db gorp.SqlExecutor, w *sdk.Workflow, id int64, u *sdk.User) (*sdk.WorkflowNodeJoin, error) {
	return nil, nil
}

func loadJoinTrigger(db gorp.SqlExecutor, w *sdk.Workflow, id int64, u *sdk.User) (*sdk.WorkflowNodeJoinTrigger, error) {
	return nil, nil
}

func insertOrUpdateJoin(db gorp.SqlExecutor, w *sdk.Workflow, n *sdk.WorkflowNodeJoin, u *sdk.User) error {
	return nil
}

func insertOrUpdateJoinTrigger(db gorp.SqlExecutor, w *sdk.Workflow, j *sdk.WorkflowNodeJoin, trigger *sdk.WorkflowNodeJoinTrigger, u *sdk.User) error {
	return nil
}

func deleteJoin(db gorp.SqlExecutor, w *sdk.Workflow, n *sdk.WorkflowNodeJoin) error {
	return nil
}

func deleteJoinTrigger(db gorp.SqlExecutor, w *sdk.Workflow, n *sdk.WorkflowNodeJoinTrigger) error {
	return nil
}
