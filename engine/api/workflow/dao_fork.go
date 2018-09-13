package workflow

import (
	"context"
	"database/sql"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func insertFork(db gorp.SqlExecutor, store cache.Store, w *sdk.Workflow, node *sdk.WorkflowNode, fork *sdk.WorkflowNodeFork, u *sdk.User) error {
	fork.WorkflowNodeID = node.ID

	dbFork := dbNodeFork(*fork)
	if err := db.Insert(&dbFork); err != nil {
		return sdk.WrapError(err, "insertFork> Unable to insert fork")
	}
	*fork = sdk.WorkflowNodeFork(dbFork)

	//Setup destination triggers
	for i := range fork.Triggers {
		t := &fork.Triggers[i]
		if errJT := insertForkTrigger(db, store, w, fork, t, u); errJT != nil {
			return sdk.WrapError(errJT, "insertFork> Unable to insert or update trigger")
		}
	}

	return nil
}

func insertForkTrigger(db gorp.SqlExecutor, store cache.Store, w *sdk.Workflow, fork *sdk.WorkflowNodeFork, trigger *sdk.WorkflowNodeForkTrigger, u *sdk.User) error {
	trigger.WorkflowForkID = fork.ID
	trigger.ID = 0
	trigger.WorkflowDestNodeID = 0

	//Setup destination node
	if err := insertNode(db, store, w, &trigger.WorkflowDestNode, u, false); err != nil {
		return sdk.WrapError(err, "insertForkTrigger> Unable to setup destination node %d on trigger %d", trigger.WorkflowDestNode.ID, trigger.ID)
	}
	trigger.WorkflowDestNodeID = trigger.WorkflowDestNode.ID

	//Insert trigger
	dbt := dbNodeForkTrigger(*trigger)
	if err := db.Insert(&dbt); err != nil {
		return sdk.WrapError(err, "insertForkTrigger> Unable to insert trigger")
	}
	trigger.ID = dbt.ID
	trigger.WorkflowDestNode.TriggerSrcID = trigger.ID

	// Update node trigger ID
	if err := updateWorkflowForkTriggerSrc(db, &trigger.WorkflowDestNode); err != nil {
		return sdk.WrapError(err, "insertForkTrigger> Unable to update node %d for trigger %d", trigger.WorkflowDestNode.ID, trigger.ID)
	}

	return nil
}

func updateWorkflowForkTriggerSrc(db gorp.SqlExecutor, n *sdk.WorkflowNode) error {
	//Update node
	query := "UPDATE workflow_node SET workflow_fork_trigger_src_id = $1 WHERE id = $2"
	if _, err := db.Exec(query, n.TriggerSrcID, n.ID); err != nil {
		return sdk.WrapError(err, "updateWorkflowForkTriggerSrc> Unable to set  workflow_fork_trigger_src_id ON node %d", n.ID)
	}
	return nil
}

func loadForks(ctx context.Context, db gorp.SqlExecutor, store cache.Store, proj *sdk.Project, w *sdk.Workflow, node *sdk.WorkflowNode, u *sdk.User, opts LoadOptions) ([]sdk.WorkflowNodeFork, error) {
	res := []dbNodeFork{}
	if _, err := db.Select(&res, "select * FROM workflow_node_fork where workflow_node_id = $1", node.ID); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, sdk.WrapError(err, "loadForks")
	}

	forks := make([]sdk.WorkflowNodeFork, len(res))
	for i := range res {
		res[i].WorkflowNodeID = node.ID
		forks[i] = sdk.WorkflowNodeFork(res[i])

		//Select triggers id
		var triggerIDs []int64
		if _, err := db.Select(&triggerIDs, "select id from workflow_node_fork_trigger where workflow_node_fork_id = $1", forks[i].ID); err != nil {
			if err == sql.ErrNoRows {
				continue
			}
			return nil, sdk.WrapError(err, "loadForks> Unable to load for triggers id for hook %d", forks[i].ID)
		}

		//Load triggers
		for _, t := range triggerIDs {
			jt, err := loadForkTrigger(ctx, db, store, proj, w, t, u, opts)
			if err != nil {
				if err == sql.ErrNoRows {
					log.Warning("loadForks> trigger %d not found", t)
					continue
				}
				return nil, sdk.WrapError(err, "loadForks> Unable to load hook trigger %d", t)
			}

			forks[i].Triggers = append(forks[i].Triggers, jt)
		}
	}

	return forks, nil
}

func loadForkTrigger(ctx context.Context, db gorp.SqlExecutor, store cache.Store, proj *sdk.Project, w *sdk.Workflow, id int64, u *sdk.User, opts LoadOptions) (sdk.WorkflowNodeForkTrigger, error) {
	var t sdk.WorkflowNodeForkTrigger

	dbtrigger := dbNodeForkTrigger{}
	//Load the trigger
	if err := db.SelectOne(&dbtrigger, "select * from workflow_node_fork_trigger where id = $1", id); err != nil {
		if err == sql.ErrNoRows {
			return t, nil
		}
		return t, sdk.WrapError(err, "loadForkTrigger> Unable to load trigger %d", id)
	}

	t = sdk.WorkflowNodeForkTrigger(dbtrigger)
	//Load node destination
	if t.WorkflowDestNodeID != 0 {
		dest, err := loadNode(ctx, db, store, proj, w, t.WorkflowDestNodeID, u, opts)
		if err != nil {
			return t, sdk.WrapError(err, "loadForkTrigger> Unable to load destination node %d", t.WorkflowDestNodeID)
		}
		t.WorkflowDestNode = *dest
	}

	return t, nil
}
